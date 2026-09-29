#!/usr/bin/env bash
# Cria os recursos AWS necessários para o repositório teste-sre-roberto
# publicar imagens e fazer deploy diretamente (via kubectl) no MESMO cluster
# EKS usado pelo hello-eks-observability, sem depender do Argo CD.
#
# Requisitos: aws cli configurado com credenciais que já têm permissão
# administrativa na conta (ex: as mesmas usadas para dar `terraform apply`
# no hello-eks-observability).
#
# Reaproveita da conta existente:
#   - o mesmo OIDC provider do GitHub (token.actions.githubusercontent.com)
#   - o mesmo cluster EKS (hello-observability-prod)
# Não altera nada do repositório/infra do hello-eks-observability.
set -euo pipefail

GITHUB_REPO="betodalas/teste-sre-roberto"
CLUSTER_NAME="hello-observability-prod"
AWS_REGION="us-east-1"
ROLE_NAME="teste-sre-roberto-deploy"
ECR_REPO="math-api"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
OIDC_PROVIDER_ARN="arn:aws:iam::${ACCOUNT_ID}:oidc-provider/token.actions.githubusercontent.com"

echo "==> Verificando OIDC provider do GitHub na conta ${ACCOUNT_ID}..."
if ! aws iam get-open-id-connect-provider --open-id-connect-provider-arn "$OIDC_PROVIDER_ARN" >/dev/null 2>&1; then
  echo "Provider não encontrado. Se o hello-eks-observability já criou um, confira a região/conta." >&2
  exit 1
fi

echo "==> Criando/atualizando trust policy da role ${ROLE_NAME}..."
TRUST_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Principal": { "Federated": "${OIDC_PROVIDER_ARN}" },
    "Action": "sts:AssumeRoleWithWebIdentity",
    "Condition": {
      "StringEquals": { "token.actions.githubusercontent.com:aud": "sts.amazonaws.com" },
      "StringLike":  { "token.actions.githubusercontent.com:sub": "repo:${GITHUB_REPO}:*" }
    }
  }]
}
EOF
)

if aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
  aws iam update-assume-role-policy --role-name "$ROLE_NAME" --policy-document "$TRUST_POLICY"
else
  aws iam create-role --role-name "$ROLE_NAME" --assume-role-policy-document "$TRUST_POLICY"
fi
ROLE_ARN=$(aws iam get-role --role-name "$ROLE_NAME" --query 'Role.Arn' --output text)

echo "==> Anexando policy inline (ECR push + EKS describe)..."
PERMISSIONS_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EcrAuth",
      "Effect": "Allow",
      "Action": ["ecr:GetAuthorizationToken"],
      "Resource": "*"
    },
    {
      "Sid": "EcrPushPull",
      "Effect": "Allow",
      "Action": [
        "ecr:CreateRepository",
        "ecr:DescribeRepositories",
        "ecr:BatchCheckLayerAvailability",
        "ecr:InitiateLayerUpload",
        "ecr:UploadLayerPart",
        "ecr:CompleteLayerUpload",
        "ecr:PutImage",
        "ecr:BatchGetImage",
        "ecr:GetDownloadUrlForLayer"
      ],
      "Resource": "arn:aws:ecr:${AWS_REGION}:${ACCOUNT_ID}:repository/${ECR_REPO}"
    },
    {
      "Sid": "EksDescribe",
      "Effect": "Allow",
      "Action": ["eks:DescribeCluster"],
      "Resource": "arn:aws:eks:${AWS_REGION}:${ACCOUNT_ID}:cluster/${CLUSTER_NAME}"
    }
  ]
}
EOF
)
aws iam put-role-policy --role-name "$ROLE_NAME" --policy-name "${ROLE_NAME}-permissions" --policy-document "$PERMISSIONS_POLICY"

echo "==> Criando repositório ECR '${ECR_REPO}' (se não existir)..."
aws ecr describe-repositories --repository-names "$ECR_REPO" --region "$AWS_REGION" >/dev/null 2>&1 \
  || aws ecr create-repository --repository-name "$ECR_REPO" --region "$AWS_REGION" \
       --image-tag-mutability IMMUTABLE --image-scanning-configuration scanOnPush=true

# Escopo "cluster" com AmazonEKSEditPolicy (não é cluster-admin) porque o
# workflow também cria o Namespace math-api, que é um recurso cluster-scoped
# — um Access Entry com escopo de namespace não teria permissão pra isso.
echo "==> Registrando Access Entry no EKS (${CLUSTER_NAME})..."
aws eks create-access-entry \
  --cluster-name "$CLUSTER_NAME" \
  --region "$AWS_REGION" \
  --principal-arn "$ROLE_ARN" \
  --type STANDARD >/dev/null 2>&1 || true

aws eks associate-access-policy \
  --cluster-name "$CLUSTER_NAME" \
  --region "$AWS_REGION" \
  --principal-arn "$ROLE_ARN" \
  --policy-arn arn:aws:eks::aws:cluster-access-policy/AmazonEKSEditPolicy \
  --access-scope type=cluster

echo
echo "==> Pronto! Cadastre no GitHub (Settings > Secrets and variables > Actions > Variables) do repo ${GITHUB_REPO}:"
echo "AWS_DEPLOY_ROLE_ARN = ${ROLE_ARN}"
echo "AWS_REGION          = ${AWS_REGION}"
echo "EKS_CLUSTER_NAME     = ${CLUSTER_NAME}"
echo "ECR_REPOSITORY       = ${ECR_REPO}"
