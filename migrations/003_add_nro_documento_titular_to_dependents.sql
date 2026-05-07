-- Migration: Adicionar documento do titular nos dependentes
-- Data: 2026-04-29
-- Descrição: Armazena o nro_documento_titular para vincular cada dependente ao documento do titular

ALTER TABLE dependents
ADD COLUMN IF NOT EXISTS nro_documento_titular VARCHAR(14);

UPDATE dependents d
SET nro_documento_titular = c.cpf_cnpj
FROM customers c
WHERE d.customer_id = c.id
  AND (d.nro_documento_titular IS NULL OR d.nro_documento_titular = '');

ALTER TABLE dependents
ALTER COLUMN nro_documento_titular SET NOT NULL;

COMMENT ON COLUMN dependents.nro_documento_titular IS 'CPF/CNPJ do titular vinculado ao dependente';