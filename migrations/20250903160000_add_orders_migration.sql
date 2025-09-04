-- Orders for subscription payments via TBank
CREATE TABLE IF NOT EXISTS orders (
    id BIGSERIAL PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'pending', -- pending | paid | failed | canceled
    user_id UUID NOT NULL REFERENCES users(id),
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'RUB',
    -- TBank fields
    tb_order_id TEXT, -- OrderId from TBank
    tb_rebill_id TEXT, -- RebillId from TBank (for recurring charges)
    payment_url TEXT, -- URL returned by Init
    paid_at TIMESTAMP,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_plan_id ON orders(plan_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_tb_order_id ON orders(tb_order_id) WHERE tb_order_id IS NOT NULL;

