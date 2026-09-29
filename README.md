# Teste SRE — API de operações matemáticas

Solução do teste técnico para a vaga de SRE: uma API HTTP em Go que expõe as
quatro operações básicas da matemática (soma, subtração, multiplicação e
divisão), com testes unitários e manifests para deploy em Kubernetes.

## Endpoints

A API roda na porta `8000` e expõe:

| Operação       | Endpoint                                                |
| -------------- | -------------------------------------------------------- |
| Adição         | `GET /api/sum?term_one=<int>&term_two=<int>`             |
| Subtração      | `GET /api/sub?term_one=<int>&term_two=<int>`             |
| Multiplicação  | `GET /api/mul?term_one=<int>&term_two=<int>`             |
| Divisão        | `GET /api/div?term_one=<int>&term_two=<int>`             |
| Healthcheck    | `GET /healthz`                                            |

Toda operação retorna um JSON no formato:

```json
{ "result": 3 }
```

Exemplo:

```bash
curl "http://localhost:8000/api/sub?term_one=4&term_two=1"
# {"result":3}
```

Erros (parâmetro ausente, inválido ou divisão por zero) retornam HTTP `400`
com `{"error": "<mensagem>"}`.

## Estrutura do projeto

```
cmd/server/          # entrypoint da aplicação (main.go)
internal/mathops/     # implementação das 4 operações + testes unitários
internal/api/          # camada HTTP (handlers, roteamento) + testes unitários
deploy/k8s/            # manifests Kubernetes (Deployment, Service, Namespace)
Dockerfile              # build multi-stage, imagem final distroless
.github/workflows/ci.yml # pipeline de testes/build no GitHub Actions
```

## Rodando localmente

Requisitos: Go 1.22+.

```bash
make run
# ou: go run ./cmd/server
```

## Testes e cobertura

```bash
make test    # go test ./... -race -cover
make cover   # gera coverage.out e coverage.html
```

Cobertura atual: `internal/mathops` 100% e `internal/api` ~96%.

## Build da imagem Docker

```bash
make docker-build
# ou: docker build -t math-api:local .
```

A imagem final é baseada em `distroless/static`, roda como usuário
não-root e expõe apenas a porta `8000`.

## Deploy no Kubernetes

Os manifests estão em `deploy/k8s/` e usam Kustomize:

- `namespace.yaml`: cria o namespace `math-api`.
- `deployment.yaml`: 2 réplicas da aplicação, com `readinessProbe` e
  `livenessProbe` apontando para `/healthz`.
- `service.yaml`: `Service` do tipo `ClusterIP`, garantindo que a
  aplicação seja acessível **somente por outras aplicações dentro do
  cluster** (sem exposição externa via LoadBalancer/Ingress).
- `networkpolicy.yaml`: reforça a restrição acima em nível de rede,
  permitindo tráfego de entrada apenas de pods dentro do cluster (nenhum
  ingress externo é liberado para os pods da aplicação).

### Publicar a imagem

O CI já publica automaticamente a imagem no **GHCR** a cada push na `main`
(veja `.github/workflows/ci.yml`, job `publish`), em:

```
ghcr.io/betodalas/teste-sre-roberto:latest
ghcr.io/betodalas/teste-sre-roberto:<sha-do-commit>
```

`deploy/k8s/deployment.yaml` já referencia essa mesma imagem — **não
precisa editar nada** para usar a versão publicada pelo CI.

**Importante — na primeira publicação**, o pacote no GHCR normalmente
nasce **privado**. Torne-o público (senão o `kubectl`/cluster não consegue
puxar a imagem sem um `imagePullSecret`):

1. Acesse `https://github.com/betodalas?tab=packages` (ou a aba
   **Packages** do repositório).
2. Abra o pacote `teste-sre-roberto` → **Package settings**.
3. Em **Danger Zone**, mude a visibilidade para **Public**.

Se preferir publicar manualmente em outro registry (ou não usar o GHCR),
o processo continua sendo:

```bash
docker build -t <seu-registry>/teste-sre-roberto:latest .
docker push <seu-registry>/teste-sre-roberto:latest
```

Nesse caso, atualize o campo `image` em `deploy/k8s/deployment.yaml` para
apontar pro seu registry.

### Aplicar os manifests

```bash
kubectl apply -k deploy/k8s
# ou: make k8s-deploy
```

Verifique o rollout:

```bash
kubectl -n math-api get pods,svc,deploy
```

### Testar a API dentro do cluster

Como o `Service` é `ClusterIP`, o teste deve ser feito de dentro do
cluster, por exemplo com um pod temporário:

```bash
kubectl -n math-api run curl-test --rm -it --image=curlimages/curl --restart=Never -- \
  curl "http://math-api:8000/api/sub?term_one=4&term_two=1"
```

Ou via port-forward, para testes manuais a partir da sua máquina:

```bash
kubectl -n math-api port-forward svc/math-api 8000:8000
curl "http://localhost:8000/healthz"
```

### Atualizar a aplicação

Após alterar o código, gere uma nova imagem com uma nova tag, atualize o
`image` em `deploy/k8s/deployment.yaml` (ou use
`kubectl -n math-api set image deployment/math-api math-api=<nova-imagem>`)
e reaplique:

```bash
kubectl apply -k deploy/k8s
kubectl -n math-api rollout status deployment/math-api
```

### Rollback

Se um novo deploy apresentar problema, reverta para a revisão anterior do
`Deployment` sem precisar reaplicar manifests antigos:

```bash
kubectl -n math-api rollout undo deployment/math-api
kubectl -n math-api rollout status deployment/math-api
```

Para ver o histórico de revisões: `kubectl -n math-api rollout history deployment/math-api`.

### Remover a aplicação

```bash
kubectl delete -k deploy/k8s
# ou: make k8s-remove
```

Isso remove o `Deployment`, o `Service` e o `Namespace` `math-api`.

## CI/CD: como o build e o publish estão sendo feitos hoje

O workflow [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) roda em
dois jobs, a cada push/PR:

1. **`test`** (todo push e PR): `go vet`, testes com cobertura
   (`go test ./... -race -coverprofile`), `docker build` local e um teste
   de integração real — o job sobe o container, aguarda o `/healthz`
   ficar `ok` e faz `curl` nos 4 endpoints (`sum`, `sub`, `mul`, `div`,
   incluindo divisão por zero retornando `400`) contra a imagem recém
   buildada. Essa imagem é descartada no final do job, não fica salva em
   lugar nenhum.
2. **`publish`** (só em push na `main`, depois do `test` passar): faz login
   no **GHCR** (`ghcr.io`) com o próprio `GITHUB_TOKEN` do workflow — sem
   precisar cadastrar credenciais — e builda + publica a imagem em:

   ```
   ghcr.io/betodalas/teste-sre-roberto:latest
   ghcr.io/betodalas/teste-sre-roberto:<sha-do-commit>
   ```

Ou seja: **o build e o push são automáticos** (CI), mas o **apply no
cluster é manual** — você (ou o avaliador) roda `kubectl apply -k
deploy/k8s` quando quiser, apontando pra imagem que o CI já deixou pronta
em `ghcr.io/betodalas/teste-sre-roberto:latest` (é exatamente o que
`deploy/k8s/deployment.yaml` referencia, então não precisa editar nada).
Esse fluxo funciona em qualquer cluster Kubernetes (local ou remoto) que
você já tenha acesso via `kubectl` — não depende de AWS/EKS.

## Bônus: como seria com pipeline 100% automatizada (ECR + EKS)

Esta seção é só **referência/documentação** — não há nenhum workflow ativo
no repositório fazendo isso. Hoje o CI já builda e publica a imagem
automaticamente (no GHCR, seção acima); o que faltaria pra "tudo
automático" é o **próprio apply no cluster** também rodar no pipeline, sem
intervenção manual. Descreve como isso ficaria publicando no Amazon ECR e
aplicando direto num cluster EKS a cada push, sem Argo CD.

### 1. Autenticação sem secrets estáticos (IAM Role + OIDC do GitHub)

Em vez de `AWS_ACCESS_KEY_ID`/`SECRET` fixos, uma IAM Role com trust policy
restrita a este repositório, assumida via OIDC a cada execução do workflow:

```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Federated": "arn:aws:iam::<conta>:oidc-provider/token.actions.githubusercontent.com" },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": { "token.actions.githubusercontent.com:aud": "sts.amazonaws.com" },
      "StringLike":  { "token.actions.githubusercontent.com:sub": "repo:betodalas/teste-sre-roberto:*" }
    }
  }]
}
```

A role teria permissão mínima: push/pull no repositório ECR `math-api` e
`eks:DescribeCluster` no cluster alvo.

### 2. Acesso ao Kubernetes (EKS Access Entry)

IAM sozinho não autoriza chamadas ao `kube-apiserver`; é preciso registrar a
role como um "usuário" do cluster (API moderna de Access Entries,
substituindo o antigo `aws-auth` ConfigMap):

```bash
aws eks create-access-entry --cluster-name <cluster> \
  --principal-arn <role-arn> --type STANDARD

aws eks associate-access-policy --cluster-name <cluster> \
  --principal-arn <role-arn> \
  --policy-arn arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy \
  --access-scope type=cluster
```

`AmazonEKSEditPolicy` dá permissão para criar/atualizar recursos, mas não é
cluster-admin nem mexe em RBAC/identidades.

### 3. Workflow do GitHub Actions

```yaml
name: Build and deploy (ECR + EKS)

on:
  push:
    branches: [main]

permissions:
  id-token: write   # necessário pro OIDC
  contents: read

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-region: ${{ vars.AWS_REGION }}
          role-to-assume: ${{ vars.AWS_DEPLOY_ROLE_ARN }}

      - uses: aws-actions/amazon-ecr-login@v2
        id: ecr

      - name: Build and push image
        run: |
          IMAGE="${{ steps.ecr.outputs.registry }}/${{ vars.ECR_REPOSITORY }}:${{ github.sha }}"
          docker build -t "$IMAGE" .
          docker push "$IMAGE"
          echo "IMAGE=$IMAGE" >> "$GITHUB_ENV"

      - name: Update kubeconfig
        run: aws eks update-kubeconfig --region ${{ vars.AWS_REGION }} --name ${{ vars.EKS_CLUSTER_NAME }}

      - name: Deploy
        run: |
          kubectl kustomize deploy/k8s \
            | sed "s#ghcr.io/betodalas/teste-sre-roberto:latest#${IMAGE}#" \
            | kubectl apply -f -
          kubectl -n math-api rollout status deployment/math-api --timeout=120s
```

### 4. Variáveis do repositório (Settings → Secrets and variables → Actions → Variables)

| Variável              | Exemplo                                    |
| --------------------- | ------------------------------------------- |
| `AWS_DEPLOY_ROLE_ARN` | `arn:aws:iam::<conta>:role/teste-sre-roberto-deploy` |
| `AWS_REGION`          | `us-east-1`                                 |
| `EKS_CLUSTER_NAME`    | `<nome-do-cluster>`                         |
| `ECR_REPOSITORY`      | `math-api`                                  |

Com isso, todo push em `main` buildaria, publicaria no ECR e aplicaria os
manifests automaticamente — sem intervenção manual e sem Argo CD.
