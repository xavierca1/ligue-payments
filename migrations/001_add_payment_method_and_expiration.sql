-- Migration: Adiciona campo payment_method e status EXPIRED
-- Data: 2026-02-03
-- Descrição: Suporte para expiração de PIX e tracking de método de pagamento

-- Adiciona coluna payment_method se não existir
ALTER TABLE subscriptions 
ADD COLUMN IF NOT EXISTS payment_method VARCHAR(20);

-- Atualiza valores existentes
UPDATE subscriptions 
SET payment_method = 'PIX' 
WHERE payment_method IS NULL;

-- Cria índice para buscar PIX expirados (usado pelo worker)
CREATE INDEX IF NOT EXISTS idx_subscriptions_expiration 
ON subscriptions(status, payment_method, created_at)
WHERE status = 'PENDING' AND payment_method = 'PIX';

-- Comentários para documentação
COMMENT ON COLUMN subscriptions.payment_method IS 'PIX, CREDIT_CARD, BOLETO';
COMMENT ON INDEX idx_subscriptions_expiration IS 'Usado pelo worker de expiração de PIX (30min)';
