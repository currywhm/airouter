-- Durable successful-registration events used by anti-abuse cleanup.
-- Raw IPs, browser fingerprints, and emails are never stored here; hashes and
-- normalized domains are enough to correlate bursts without expanding the
-- platform's personal-data footprint.

CREATE TABLE IF NOT EXISTS registration_abuse_events (
    id                 BIGSERIAL PRIMARY KEY,
    user_id            BIGINT NOT NULL,
    inviter_id         BIGINT,
    client_ip_hash     CHAR(64) NOT NULL DEFAULT '',
    fingerprint_hash   CHAR(64) NOT NULL DEFAULT '',
    email_domain       VARCHAR(255) NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_registration_abuse_events_created_at
    ON registration_abuse_events (created_at);
CREATE INDEX IF NOT EXISTS idx_registration_abuse_events_ip_window
    ON registration_abuse_events (client_ip_hash, created_at);
CREATE INDEX IF NOT EXISTS idx_registration_abuse_events_fingerprint_window
    ON registration_abuse_events (fingerprint_hash, created_at);
CREATE INDEX IF NOT EXISTS idx_registration_abuse_events_domain_window
    ON registration_abuse_events (email_domain, created_at);
CREATE INDEX IF NOT EXISTS idx_registration_abuse_events_inviter_window
    ON registration_abuse_events (inviter_id, created_at);

COMMENT ON TABLE registration_abuse_events IS
    '成功注册的风控事件；用于检测并清理批量注册和异常邀请';
COMMENT ON COLUMN registration_abuse_events.client_ip_hash IS
    '客户端 IP 的 SHA-256，不保存原始 IP';
COMMENT ON COLUMN registration_abuse_events.fingerprint_hash IS
    '浏览器注册指纹的 SHA-256，不保存原始指纹';
COMMENT ON COLUMN registration_abuse_events.deleted_at IS
    '关联账户被风控清理的时间；事件仍保留用于后续重复行为检测';
