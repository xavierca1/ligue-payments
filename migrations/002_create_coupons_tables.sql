-- Tabela de cupons por vendedor
CREATE TABLE IF NOT EXISTS coupons (
    id UUID PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    seller_name VARCHAR(255) NOT NULL,
    discount_percent INTEGER NOT NULL DEFAULT 10,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    starts_at TIMESTAMP NULL,
    ends_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_coupons_discount_percent CHECK (discount_percent >= 0 AND discount_percent <= 100)
);

CREATE INDEX IF NOT EXISTS idx_coupons_active ON coupons (is_active);
CREATE INDEX IF NOT EXISTS idx_coupons_code_upper ON coupons ((UPPER(code)));
CREATE INDEX IF NOT EXISTS idx_coupons_seller_name ON coupons (seller_name);

-- Log de vendas vinculadas a cupom (atribuição para vendedor)
CREATE TABLE IF NOT EXISTS coupon_sales (
    id UUID PRIMARY KEY,
    coupon_code VARCHAR(64) NOT NULL,
    seller_name VARCHAR(255) NOT NULL,
    customer_id UUID NOT NULL,
    subscription_id UUID NOT NULL,
    plan_id UUID NOT NULL,
    original_amount_cents INTEGER NOT NULL,
    discount_percent INTEGER NOT NULL,
    discount_amount_cents INTEGER NOT NULL,
    final_amount_cents INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_coupon_sales_coupon_code FOREIGN KEY (coupon_code) REFERENCES coupons(code),
    CONSTRAINT chk_coupon_sales_discount_percent CHECK (discount_percent >= 0 AND discount_percent <= 100),
    CONSTRAINT chk_coupon_sales_original_amount CHECK (original_amount_cents >= 0),
    CONSTRAINT chk_coupon_sales_discount_amount CHECK (discount_amount_cents >= 0),
    CONSTRAINT chk_coupon_sales_final_amount CHECK (final_amount_cents >= 0)
);

CREATE INDEX IF NOT EXISTS idx_coupon_sales_coupon_code ON coupon_sales (coupon_code);
CREATE INDEX IF NOT EXISTS idx_coupon_sales_seller_name ON coupon_sales (seller_name);
CREATE INDEX IF NOT EXISTS idx_coupon_sales_created_at ON coupon_sales (created_at);
CREATE INDEX IF NOT EXISTS idx_coupon_sales_subscription_id ON coupon_sales (subscription_id);
