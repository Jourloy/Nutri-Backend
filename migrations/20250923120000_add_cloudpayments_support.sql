ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'cloudpayments',
    ADD COLUMN IF NOT EXISTS cp_order_id TEXT,
    ADD COLUMN IF NOT EXISTS cp_transaction_id TEXT,
    ADD COLUMN IF NOT EXISTS cp_subscription_id TEXT;

UPDATE orders
SET provider = 'tbank'
WHERE tb_order_id IS NOT NULL;

UPDATE orders
SET provider = 'cloudpayments'
WHERE provider IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_cp_order_id ON orders(cp_order_id) WHERE cp_order_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_orders_cp_subscription_id ON orders(cp_subscription_id);
CREATE INDEX IF NOT EXISTS idx_orders_cp_transaction_id ON orders(cp_transaction_id);

CREATE TABLE IF NOT EXISTS cloudpayments_notifications (
    id BIGSERIAL PRIMARY KEY,
    type TEXT NOT NULL,
    invoice_id TEXT,
    order_id BIGINT REFERENCES orders(id),
    subscription_id TEXT,
    payload JSONB NOT NULL,
    headers JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cp_notifications_type ON cloudpayments_notifications(type);
CREATE INDEX IF NOT EXISTS idx_cp_notifications_invoice_id ON cloudpayments_notifications(invoice_id);
CREATE INDEX IF NOT EXISTS idx_cp_notifications_subscription_id ON cloudpayments_notifications(subscription_id);
CREATE INDEX IF NOT EXISTS idx_cp_notifications_order_id ON cloudpayments_notifications(order_id);
