-- Daily wallet check-in rewards. The unique key makes the claim idempotent
-- across refreshes, retries, and concurrent requests.
CREATE TABLE IF NOT EXISTS daily_checkins (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date DATE NOT NULL,
    reward_amount DECIMAL(20,8) NOT NULL CHECK (reward_amount >= 1 AND reward_amount <= 3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT daily_checkins_user_date_unique UNIQUE (user_id, checkin_date)
);

CREATE INDEX IF NOT EXISTS idx_daily_checkins_user_date
    ON daily_checkins(user_id, checkin_date DESC);

COMMENT ON TABLE daily_checkins IS '每日签到钱包奖励记录';
