# Ligue Payments — API de Pagamentos

API de checkout, assinaturas e gestão de clientes integrada com Asaas, Doc24, Kommo CRM, DocuSeal e Supabase.

---

## Stack

- **Runtime:** Go 1.25
- **Fila:** RabbitMQ (worker de ativações)
- **Banco:** PostgreSQL via Supabase
- **Pagamentos:** Asaas
- **Contratos:** DocuSeal + Supabase Storage
- **Email:** Microsoft Graph API (Azure)
- **CRM:** Kommo
- **Telemedicina:** Doc24
- **Observabilidade:** Datadog
- **Infra:** Docker Swarm + Portainer + Traefik (KVM Cuidai)

---

## Deploy na KVM

### Pré-requisitos (já configurados no servidor)

- Docker Swarm inicializado
- Portainer rodando
- Traefik rodando na rede `cuidai_net` com certresolver `letsencryptresolver`
- DNS `api-ligue-payments.cuidai.xyz` apontando para o IP da KVM (Cloudflare, nuvem cinza)

### Primeiro deploy

```bash
# 1. Clonar o repo
git clone https://github.com/xavierca1/ligue-payments.git ~/ligue-payments
cd ~/ligue-payments

# 2. Build das imagens
docker build -t ligue-payments:latest .
docker build -f Dockerfile.rabbitmq -t ligue-rabbitmq:latest .

# 3. Exportar variáveis e fazer deploy
export DATABASE_URL='...' && export ASAAS_API_KEY='...' && \
  docker stack deploy -c stack.yml ligue --resolve-image never
```

### Redeploy (após push no main)

O GitHub Actions faz o deploy automaticamente. Para fazer manualmente:

```bash
cd ~/ligue-payments && git pull
docker build -t ligue-payments:latest .
docker stack deploy -c stack.yml ligue --resolve-image never
```

### Comando de deploy completo com todas as variáveis

> Substitua os valores pelas credenciais reais antes de rodar.

```bash
export DATABASE_URL='postgresql://...' && \
export RABBITMQ_USER='guest' && \
export RABBITMQ_PASS='guest' && \
export ASAAS_API_KEY='$aact_prod_...' && \
export ASAAS_URL='https://asaas.com/api/v3' && \
export ASAAS_WEBHOOK_SECRET='whsec_...' && \
export ASAAS_WEBHOOK_SKIP_SIGNATURE='false' && \
export ASAAS_WEBHOOK_DEBUG='false' && \
export KOMMO_ACCOUNT_ID='liguemedicina' && \
export KOMMO_API_TOKEN='eyJ...' && \
export KOMMO_PIPELINE_B2C_ID='...' && \
export KOMMO_FIELD_PRODUTO_ID='...' && \
export KOMMO_FIELD_ORIGEM_ID='...' && \
export DOC24_CLIENT_ID='liguemed' && \
export DOC24_CLIENT_SECRET='...' && \
export USE_GRAPH_EMAIL='true' && \
export AZURE_CLIENT_ID='...' && \
export AZURE_TENANT_ID='...' && \
export AZURE_CLIENT_SECRET='...' && \
export MAIL_FROM='no-reply@liguemedicina.com' && \
export SUPABASE_CONTRACTS_PROJECT_URL='https://...supabase.co' && \
export SUPABASE_CONTRACTS_BUCKET='contracts' && \
export SUPABASE_CONTRACTS_SERVICE_ROLE_KEY='eyJ...' && \
export DOCUSEAL_API_URL='https://api.docuseal.com' && \
export DOCUSEAL_API_KEY='...' && \
export SUPABASE_STORAGE_URL='https://...supabase.co/storage/...' && \
export DD_API_KEY='...' && \
export DD_ENV='production' && \
docker stack deploy -c stack.yml ligue --resolve-image never
```

**Importante:** Usar sempre aspas simples `'` nos valores para evitar que o shell interprete caracteres especiais como `$`.

---

## Verificação pós-deploy

```bash
# Saúde da API
curl https://api-ligue-payments.cuidai.xyz/healthz

# Status dos serviços
docker service ls
docker service ps ligue_api

# Logs em tempo real
docker service logs ligue_api -f

# Logs de erro específico
docker service logs ligue_api --tail 50 2>&1 | grep -i "error\|fatal\|checkout"
```

---

## Problemas conhecidos e soluções

### 1. `404 page not found` no domínio

**Causa:** Traefik não está roteando para o serviço.

**Diagnóstico:**
```bash
docker service inspect ligue_api | grep "rule\|certresolver"
```

**Soluções:**
- Label `rule` vazio → variável `API_DOMAIN` não foi substituída. Usar valor hardcoded no `stack.yml`
- CertResolver errado → usar `letsencryptresolver` (não `le` nem `letsencrypt`)
- Rede errada → o serviço deve estar na rede `cuidai_net` (não `traefik-public`)
- Fazer stack rm + redeploy para forçar atualização dos labels:
  ```bash
  docker stack rm ligue && sleep 15
  docker stack deploy -c stack.yml ligue --resolve-image never
  ```

### 2. `DATABASE_URL não encontrada`

**Causa:** O `source .env` do bash interpreta o `$` como variável e esvazia a key.

**Solução:** Sempre usar `export VAR='valor'` com aspas simples, nunca `source .env` diretamente para variáveis com `$` no valor (ex: `ASAAS_API_KEY`).

### 3. `401 Unauthorized` no Asaas

**Causa:** `ASAAS_API_KEY` chegou vazia no container.

**Verificação:**
```bash
docker service logs ligue_api --tail 20 | grep "asaas\|401"
```

**Solução:** Garantir que o export use aspas simples:
```bash
export ASAAS_API_KEY='$aact_prod_...'  # aspas simples = literal
```

### 4. `Gateway Timeout`

**Causa:** Traefik está roteando mas o container não está healthy ou não está na rede certa.

**Diagnóstico:**
```bash
docker service ps ligue_api  # verificar se está running
docker service inspect ligue_api | grep NetworkID
docker network ls | grep cuidai_net  # confirmar ID da rede
```

### 5. Labels do Traefik não atualizam após `docker stack deploy`

**Causa:** Docker Swarm às vezes cacheia o spec anterior.

**Solução:**
```bash
docker stack rm ligue
sleep 15
docker stack deploy -c stack.yml ligue --resolve-image never
```

### 6. Imagem não encontrada no deploy

**Causa:** Swarm tenta buscar a imagem em um registry, mas é local.

**Solução:** Sempre usar a flag `--resolve-image never`:
```bash
docker stack deploy -c stack.yml ligue --resolve-image never
```

---

## GitHub Actions (CI/CD automático)

O arquivo [.github/workflows/deploy.yml](.github/workflows/deploy.yml) faz deploy automático a cada push na branch `main`.

### Secrets necessários no GitHub

Configure em **Settings > Secrets and variables > Actions**:

| Secret | Descrição |
|--------|-----------|
| `KVM_HOST` | IP da KVM |
| `KVM_USER` | Usuário SSH (root) |
| `KVM_SSH_KEY` | Chave privada SSH |
| `DATABASE_URL` | URL do PostgreSQL (Supabase) |
| `ASAAS_API_KEY` | Chave da API do Asaas |
| `ASAAS_WEBHOOK_SECRET` | Secret do webhook Asaas |
| `KOMMO_ACCOUNT_ID` | ID da conta Kommo |
| `KOMMO_API_TOKEN` | Token JWT do Kommo |
| `KOMMO_PIPELINE_B2C_ID` | ID do pipeline B2C |
| `KOMMO_FIELD_PRODUTO_ID` | ID do campo produto |
| `KOMMO_FIELD_ORIGEM_ID` | ID do campo origem |
| `DOC24_CLIENT_ID` | Client ID do Doc24 |
| `DOC24_CLIENT_SECRET` | Secret do Doc24 |
| `AZURE_CLIENT_ID` | Client ID Azure (email) |
| `AZURE_TENANT_ID` | Tenant ID Azure |
| `AZURE_CLIENT_SECRET` | Secret Azure |
| `SUPABASE_CONTRACTS_PROJECT_URL` | URL do projeto Supabase |
| `SUPABASE_CONTRACTS_SERVICE_ROLE_KEY` | Service role key Supabase |
| `SUPABASE_STORAGE_URL` | URL do storage público |
| `DOCUSEAL_API_KEY` | Chave API DocuSeal |
| `DD_API_KEY` | Chave API Datadog |

---

## Webhook Asaas

URL de produção para configurar no painel Asaas:

```
https://api-ligue-payments.cuidai.xyz/asaas/webhook
```

---

## Rotas principais

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/healthz` | Health check simples |
| `GET` | `/health` | Health check com status de dependências |
| `POST` | `/checkout` | Criar cliente + assinatura |
| `POST` | `/asaas/webhook` | Receber eventos do Asaas |
| `GET` | `/customers/lookup-cpf` | Buscar cliente por CPF |
| `GET` | `/customers/lookup-email` | Buscar cliente por email |
| `POST` | `/coupons/validate` | Validar cupom de desconto |

---

## Testando com Postman

Os arquivos de coleção e payloads estão em [postman/](postman/).

### Checkout PIX

**`POST /checkout`**

```json
{
  "name": "Maria Oliveira Costa",
  "email": "maria.oliveira@test.com",
  "cpf": "08104389173",
  "plan_id": "UUID_DO_PLANO",
  "phone": "(21) 99999-8888",
  "birth_date": "1985-03-22",
  "gender": "F",
  "payment_method": "PIX",
  "street": "Avenida Paulista",
  "number": "1000",
  "complement": "Sala 500",
  "district": "Bela Vista",
  "city": "São Paulo",
  "state": "SP",
  "zip_code": "01311-100",
  "terms_accepted": true,
  "terms_accepted_at": "2026-05-07T10:35:00Z",
  "terms_version": "1.0",
  "coupon_code": "",
  "dependents": []
}
```

**Resposta de sucesso:**
```json
{
  "id": "cus_...",
  "status": "PENDING",
  "payment_method": "PIX",
  "pix_qr_code": "00020126...",
  "pix_expiration": "2026-05-07T11:05:00Z"
}
```

---

### Checkout Cartão de Crédito

**`POST /checkout`**

```json
{
  "name": "Maria Oliveira Costa",
  "email": "maria.oliveira@test.com",
  "cpf": "08104389173",
  "plan_id": "UUID_DO_PLANO",
  "phone": "(21) 99999-8888",
  "birth_date": "1985-03-22",
  "gender": "F",
  "payment_method": "CREDIT_CARD",
  "street": "Avenida Paulista",
  "number": "1000",
  "complement": "Sala 500",
  "district": "Bela Vista",
  "city": "São Paulo",
  "state": "SP",
  "zip_code": "01311-100",
  "card_holder": "MARIA OLIVEIRA COSTA",
  "card_number": "4111111111111111",
  "card_month": "12",
  "card_year": "2027",
  "card_cvv": "123",
  "terms_accepted": true,
  "terms_accepted_at": "2026-05-07T10:35:00Z",
  "terms_version": "1.0",
  "coupon_code": "",
  "dependents": []
}
```

**Resposta de sucesso:**
```json
{
  "id": "cus_...",
  "status": "ACTIVE",
  "payment_method": "CREDIT_CARD"
}
```

---

### Checkout com Dependentes

Adicione o array `dependents` em qualquer método de pagamento:

```json
"dependents": [
  {
    "name": "Filho Silva",
    "birth_date": "2010-06-15",
    "gender": "M",
    "kinship": "FILHO",
    "cpf": "12345678902"
  }
]
```

Valores válidos para `kinship`: `FILHO`, `CONJUGE`, `PAI`, `MAE`, `OUTRO`

---

### Validar Cupom

**`POST /coupons/validate`**

```json
{
  "code": "PROMO10",
  "plan_id": "UUID_DO_PLANO"
}
```
