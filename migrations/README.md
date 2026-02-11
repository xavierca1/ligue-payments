# Feature: Dependentes

## 📋 Visão Geral

Permite que clientes adicionem dependentes (cônjuge, filhos, pais, etc.) ao seu plano. Os dependentes são **opcionais** e enviados no mesmo payload do checkout.

## 🗄️ Estrutura do Banco

### Tabela: `dependents`

```sql
CREATE TABLE dependents (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL (FK -> customers),
    name VARCHAR(255) NOT NULL,
    cpf VARCHAR(14) NOT NULL,
    birth_date DATE NOT NULL,
    gender INTEGER NOT NULL, -- 1=Masculino, 2=Feminino, 3=Outro
    kinship VARCHAR(50) NOT NULL, -- FILHO, CONJUGE, PAI, MAE
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Índices:
- `idx_dependents_customer_id` - Performance nas consultas por cliente
- `idx_dependents_cpf` - Validações e buscas por CPF

## 📦 Payload da API

### Endpoint: `POST /checkout`

```json
{
  "name": "João Silva",
  "email": "joao@example.com",
  "cpf": "123.456.789-00",
  "plan_id": "plan-123",
  "payment_method": "PIX",
  ...
  "dependents": [ // OPCIONAL
    {
      "name": "Maria Silva",
      "cpf": "987.654.321-00",
      "birth_date": "1995-03-15",
      "gender": "2",
      "kinship": "CONJUGE"
    },
    {
      "name": "Pedro Silva",
      "cpf": "111.222.333-44",
      "birth_date": "2010-07-20",
      "gender": "1",
      "kinship": "FILHO"
    }
  ]
}
```

### Campos do Dependente:
- `name` (string, obrigatório) - Nome completo
- `cpf` (string, obrigatório) - Formato: 000.000.000-00
- `birth_date` (string, obrigatório) - Formato: YYYY-MM-DD
- `gender` (string, obrigatório) - "1" (Masculino), "2" (Feminino), "3" (Outro)
- `kinship` (string, obrigatório) - Tipo de parentesco (FILHO, CONJUGE, PAI, MAE, IRMAO, etc.)

## 🔄 Fluxo de Processamento

1. **Recebe checkout** com ou sem dependentes
2. **Valida** dados do titular
3. **Cria customer** no banco
4. **Cria subscription**
5. **Salva dependentes** (se houver) em transação
6. **Retorna** resposta ao frontend

### Transação ACID:
- Se falhar em qualquer ponto, faz rollback de tudo
- Dependentes são salvos após customer e subscription
- Usa o mesmo `customer_id` para todos os dependentes

## 📁 Arquivos Criados/Modificados

### Criados:
- `/migrations/001_create_dependents_table.sql` - Schema do banco
- `/internal/entity/dependent.go` - Entidade Dependent
- `/internal/infra/database/dependent_repository.go` - Repository pattern
- `/migrations/README.md` - Esta documentação

### Modificados:
- `/internal/usecase/interfaces.go` - Adicionado `DependentInput` e `DependentRepo`
- `/internal/usecase/create_customer.go` - Lógica para salvar dependentes
- `/cmd/api/main.go` - Injeção do `DependentRepository`
- `/tests/create_customer_usecase_test.go` - Mock do DependentRepository

## 🧪 Testando

### Criar checkout SEM dependentes:
```bash
curl -X POST http://localhost:8080/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "name": "João Silva",
    "email": "joao@example.com",
    "cpf": "123.456.789-00",
    "plan_id": "plan-123",
    "payment_method": "PIX",
    ...
  }'
```

### Criar checkout COM dependentes:
```bash
curl -X POST http://localhost:8080/checkout \
  -H "Content-Type: application/json" \
  -d '{
    "name": "João Silva",
    "email": "joao@example.com",
    "cpf": "123.456.789-00",
    "plan_id": "plan-123",
    "payment_method": "PIX",
    ...
    "dependents": [
      {
        "name": "Maria Silva",
        "cpf": "987.654.321-00",
        "birth_date": "1995-03-15",
        "gender": "2",
        "kinship": "CONJUGE"
      }
    ]
  }'
```

## 🔮 Próximos Passos (Futuro)

1. **Validações adicionais:**
   - Limite de dependentes por plano
   - Validação de idade (ex: filhos menores de 21 anos)
   - CPF único global (não permitir mesmo CPF em clientes diferentes)

2. **Integrações:**
   - Enviar dependentes para Doc24 (se aplicável)
   - Enviar dependentes para Asaas (se houver cobrança adicional)
   - Incluir dependentes no email de boas-vindas

3. **Endpoints adicionais:**
   - `GET /customers/:id/dependents` - Listar dependentes
   - `POST /customers/:id/dependents` - Adicionar dependente depois
   - `DELETE /dependents/:id` - Remover dependente

## 🎯 Regras de Negócio

- **Obrigatoriedade**: Dependentes são **OPCIONAIS**
- **Limite**: Sem limite por enquanto (pode ser configurado por plano)
- **Exclusão**: Cascade delete - se customer é deletado, dependentes também
- **Cobrança**: Dependentes não alteram o preço (mesma subscription)
- **Ativação**: Dependentes são salvos junto com o customer, não precisam de ativação separada

## 🚨 Atenção

- Execute a migration `001_create_dependents_table.sql` no Supabase antes de fazer deploy
- Certifique-se que a tabela `customers` já existe (FK constraint)
- Testes foram atualizados para incluir o `MockDependentRepository`
