-- Migration: Cria tabela leads para carrinho perdido
-- Data: 2026-02-03
-- Descrição: Captura dados parciais do checkout para remarketing

CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    phone VARCHAR(50),
    status VARCHAR(20) DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'RECOVERED', 'CONVERTED')),
    email_stage INT DEFAULT 0 CHECK (email_stage IN (0, 1, 2)),
    last_email_sent_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Índices para performance do worker de remarketing
CREATE INDEX IF NOT EXISTS idx_leads_status_stage 
ON leads(status, email_stage, updated_at);

CREATE INDEX IF NOT EXISTS idx_leads_email 
ON leads(email);

-- Comentários
COMMENT ON TABLE leads IS 'Leads capturados no checkout (carrinho perdido)';
COMMENT ON COLUMN leads.email_stage IS '0=nenhum email, 1=enviado 15min, 2=enviado 48h';
COMMENT ON COLUMN leads.status IS 'PENDING=aguardando, RECOVERED=voltou ao checkout, CONVERTED=virou cliente';
