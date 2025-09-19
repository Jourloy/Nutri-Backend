ALTER TABLE orders ADD COLUMN IF NOT EXISTS ad_code TEXT;
CREATE INDEX IF NOT EXISTS idx_orders_ad_code ON orders(ad_code);

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS ad_code TEXT;
CREATE INDEX IF NOT EXISTS idx_subscriptions_ad_code ON subscriptions(ad_code);

