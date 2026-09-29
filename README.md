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

### Publicar a imagem

Build e publique a imagem em um registry acessível pelo cluster (ajuste o
nome conforme seu registry) e atualize o campo `image` em
`deploy/k8s/deployment.yaml`:

```bash
docker build -t <seu-registry>/teste-sre-roberto:latest .
docker push <seu-registry>/teste-sre-roberto:latest
```

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

### Remover a aplicação

```bash
kubectl delete -k deploy/k8s
# ou: make k8s-remove
```

Isso remove o `Deployment`, o `Service` e o `Namespace` `math-api`.

## CI

O workflow [`.github/workflows/ci.yml`](./.github/workflows/ci.yml) roda em
todo push/PR: `go vet`, testes com cobertura e o build da imagem Docker.
Não há deploy automático — o apply no cluster é sempre manual, seguindo os
passos da seção "Deploy no Kubernetes" acima, em qualquer cluster
Kubernetes (local ou remoto) que você já tenha acesso via `kubectl`.
