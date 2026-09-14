-- Re-enable affiliate binding, signup rewards, and recharge rebates.
INSERT INTO settings (key, value, updated_at)
VALUES ('affiliate_enabled', 'true', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at;
