-- Remove the built-in payment system while preserving redeem codes and
-- affiliate balances earned from redeem-code usage.

ALTER TABLE IF EXISTS user_affiliate_ledger
    DROP COLUMN IF EXISTS source_order_id;

DROP TABLE IF EXISTS payment_audit_logs CASCADE;
DROP TABLE IF EXISTS payment_orders CASCADE;
DROP TABLE IF EXISTS payment_provider_instances CASCADE;

DELETE FROM settings
WHERE LEFT(key, 8) = 'payment_'
   OR key IN ('purchase_subscription_enabled', 'purchase_subscription_url');

DO $$
BEGIN
    UPDATE settings AS s
    SET value = COALESCE((
        SELECT jsonb_agg(item)
        FROM jsonb_array_elements(s.value::jsonb) AS item
        WHERE item ->> 'id' <> 'migrated_purchase_subscription'
    ), '[]'::jsonb)::text
    WHERE s.key = 'custom_menu_items'
      AND s.value <> ''
      AND jsonb_typeof(s.value::jsonb) = 'array';
EXCEPTION
    WHEN invalid_text_representation THEN
        RAISE NOTICE '[migration-158] custom_menu_items is not valid JSON; skipped legacy purchase item cleanup';
END $$;
