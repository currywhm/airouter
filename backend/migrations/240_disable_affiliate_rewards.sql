-- Disable affiliate rebates on upgrade. Existing wallet balances and ledger
-- history are retained; new registrations and top-ups no longer accrue them.
INSERT INTO settings (key, value, updated_at)
VALUES ('affiliate_enabled', 'false', NOW())
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    updated_at = EXCLUDED.updated_at;
