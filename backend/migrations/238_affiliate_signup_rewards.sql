-- Award a fixed wallet credit once when an invited user completes signup.
-- source_user_id stores the invited user so retries and repeated OAuth
-- callbacks cannot issue the same reward twice.
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_affiliate_ledger_signup_reward_uniq
    ON user_affiliate_ledger (source_user_id)
    WHERE action = 'signup_reward' AND source_user_id IS NOT NULL;

COMMENT ON INDEX idx_user_affiliate_ledger_signup_reward_uniq IS
    'One signup reward per invited user';
