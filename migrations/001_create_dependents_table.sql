-- Migration: Criar tabela de dependentes
-- Data: 2026-02-11
-- Descrição: Tabela para armazenar dependentes vinculados aos clientes

-- Criar tabela de dependentes
CREATE TABLE IF NOT EXISTS dependents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    cpf VARCHAR(14) NOT NULL,
    birth_date DATE NOT NULL,
    gender INTEGER NOT NULL, -- 1=Masculino, 2=Feminino, 3=Outro
    kinship VARCHAR(50) NOT NULL, -- Grau de parentesco: "FILHO", "CONJUGE", "PAI", "MAE", etc
    
    -- Metadados
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Foreign Key
    CONSTRAINT fk_customer FOREIGN KEY (customer_id) 
        REFERENCES customers(id) 
        ON DELETE CASCADE,
    
    -- Constraints
    CONSTRAINT chk_gender CHECK (gender IN (1, 2, 3)),
    CONSTRAINT chk_cpf_format CHECK (cpf ~ '^\d{3}\.\d{3}\.\d{3}-\d{2}$')
);

-- Índices para performance
CREATE INDEX IF NOT EXISTS idx_dependents_customer_id ON dependents(customer_id);
CREATE INDEX IF NOT EXISTS idx_dependents_cpf ON dependents(cpf);

-- Comentários da tabela
COMMENT ON TABLE dependents IS 'Dependentes vinculados aos clientes titulares';
COMMENT ON COLUMN dependents.id IS 'ID único do dependente (UUID)';
COMMENT ON COLUMN dependents.customer_id IS 'ID do cliente titular (FK para customers)';
COMMENT ON COLUMN dependents.name IS 'Nome completo do dependente';
COMMENT ON COLUMN dependents.cpf IS 'CPF do dependente (formato: 000.000.000-00)';
COMMENT ON COLUMN dependents.birth_date IS 'Data de nascimento do dependente';
COMMENT ON COLUMN dependents.gender IS 'Gênero: 1=Masculino, 2=Feminino, 3=Outro';
COMMENT ON COLUMN dependents.kinship IS 'Grau de parentesco com o titular';
