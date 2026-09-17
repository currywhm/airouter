package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

const (
	registrationAbuseIPBurstLimit             = 3
	registrationAbuseIPHourLimit              = 10
	registrationAbuseIPDayLimit               = 30
	registrationAbuseFingerprintLimit         = 8
	registrationAbuseEmailDomainLimit         = 30
	registrationAbuseInviterBurstLimit        = 5
	registrationAbuseInviterHourLimit         = 12
	registrationAbuseEventQueryLimit          = 500
	registrationAbuseFingerprintMinimumLength = 8
	registrationAbuseFingerprintMaximumLength = 256
)

// RegistrationRiskContext carries request-level signals into the successful
// registration path. Raw values stay in memory; the store persists hashes only.
type RegistrationRiskContext struct {
	ClientIP    string
	Fingerprint string
}

type registrationAbuseEvent struct {
	UserID          int64
	InviterID       *int64
	ClientIPHash    string
	FingerprintHash string
	EmailDomain     string
}

type registrationAbuseFinding struct {
	Blocked           bool
	Signals           []string
	OffendingUserIDs  []int64
	InviterID         *int64
	InviterQuotaReset bool
}

// RegistrationAbuseDecision is returned after a successful registration has
// been evaluated and any confirmed abusive accounts have been processed.
type RegistrationAbuseDecision struct {
	Blocked           bool
	Signals           []string
	DeletedUserIDs    []int64
	InviterID         *int64
	InviterQuotaReset bool
	CleanupDegraded   bool
}

func (d *RegistrationAbuseDecision) Metadata() map[string]string {
	if d == nil {
		return nil
	}
	metadata := map[string]string{
		"signals":          strings.Join(d.Signals, ","),
		"deleted_accounts": strconv.Itoa(len(d.DeletedUserIDs)),
		"cleanup_degraded": strconv.FormatBool(d.CleanupDegraded),
	}
	if d.InviterID != nil {
		metadata["inviter_id"] = strconv.FormatInt(*d.InviterID, 10)
	}
	if d.InviterQuotaReset {
		metadata["inviter_quota_cleared"] = "true"
	}
	return metadata
}

type registrationAbuseStore interface {
	RecordAndEvaluate(ctx context.Context, event registrationAbuseEvent) (*registrationAbuseFinding, error)
	ResetInviterQuota(ctx context.Context, inviterID int64) error
	MarkAccountsDeleted(ctx context.Context, userIDs []int64) error
	DisableAccounts(ctx context.Context, userIDs []int64) error
}

type registrationAbuseUserDeleter interface {
	DeleteUser(ctx context.Context, id int64) error
}

// CleanupRejectedRegistration removes a user created by a registration whose
// abuse evaluation could not complete. If deletion is unavailable, the user
// is disabled and its balance is cleared so it cannot be used as a fallback.
func (s *RegistrationAbuseService) CleanupRejectedRegistration(ctx context.Context, userID int64) error {
	if s == nil || userID <= 0 {
		return nil
	}
	if s.userDeleter != nil {
		if err := s.userDeleter.DeleteUser(ctx, userID); err == nil {
			return nil
		} else if s.store != nil {
			if disableErr := s.store.DisableAccounts(ctx, []int64{userID}); disableErr != nil {
				return errors.Join(err, disableErr)
			}
			return err
		}
	}
	if s.store != nil {
		return s.store.DisableAccounts(ctx, []int64{userID})
	}
	return errors.New("registration abuse user cleanup is not configured")
}

type RegistrationAbuseService struct {
	store       registrationAbuseStore
	userDeleter registrationAbuseUserDeleter
}

func NewRegistrationAbuseService(client *dbent.Client, admin AdminService) *RegistrationAbuseService {
	return &RegistrationAbuseService{
		store:       &registrationAbuseSQLStore{client: client},
		userDeleter: admin,
	}
}

func (s *RegistrationAbuseService) ProcessSuccessfulRegistration(
	ctx context.Context,
	input RegistrationAbuseEventInput,
) (*RegistrationAbuseDecision, error) {
	if s == nil || s.store == nil || input.UserID <= 0 {
		return &RegistrationAbuseDecision{}, nil
	}

	event := registrationAbuseEvent{
		UserID:          input.UserID,
		InviterID:       cloneRegistrationAbuseInt64(input.InviterID),
		ClientIPHash:    hashRegistrationAbuseValue(input.ClientIP),
		FingerprintHash: hashRegistrationAbuseValue(input.Fingerprint),
		EmailDomain:     normalizeRegistrationAbuseDomain(input.Email),
	}
	finding, err := s.store.RecordAndEvaluate(ctx, event)
	if err != nil {
		return nil, err
	}
	if finding == nil || !finding.Blocked {
		return &RegistrationAbuseDecision{}, nil
	}

	decision := &RegistrationAbuseDecision{
		Blocked:   true,
		Signals:   append([]string(nil), finding.Signals...),
		InviterID: cloneRegistrationAbuseInt64(finding.InviterID),
	}
	if finding.InviterID != nil && finding.InviterQuotaReset {
		if err := s.store.ResetInviterQuota(ctx, *finding.InviterID); err != nil {
			decision.CleanupDegraded = true
			slog.Error("failed to reset abusive inviter quota",
				"inviter_id", *finding.InviterID,
				"error", err,
			)
		} else {
			decision.InviterQuotaReset = true
		}
	}

	failedIDs := make([]int64, 0)
	for _, userID := range uniquePositiveInt64s(finding.OffendingUserIDs) {
		if s.userDeleter == nil {
			failedIDs = append(failedIDs, userID)
			continue
		}
		if err := s.userDeleter.DeleteUser(ctx, userID); err != nil {
			failedIDs = append(failedIDs, userID)
			slog.Warn("failed to delete abusive registration account",
				"user_id", userID,
				"error", err,
			)
			continue
		}
		decision.DeletedUserIDs = append(decision.DeletedUserIDs, userID)
	}

	if len(failedIDs) > 0 {
		decision.CleanupDegraded = true
		if err := s.store.DisableAccounts(ctx, failedIDs); err != nil {
			slog.Error("failed to disable accounts after anti-abuse deletion failure",
				"user_ids", failedIDs,
				"error", err,
			)
		}
	}
	if len(decision.DeletedUserIDs) > 0 {
		if err := s.store.MarkAccountsDeleted(ctx, decision.DeletedUserIDs); err != nil {
			slog.Warn("failed to mark anti-abuse events as deleted",
				"user_ids", decision.DeletedUserIDs,
				"error", err,
			)
		}
	}
	if finding.InviterID != nil && finding.InviterQuotaReset {
		if err := s.store.ResetInviterQuota(ctx, *finding.InviterID); err != nil {
			decision.CleanupDegraded = true
			slog.Error("failed to reconcile abusive inviter quota after account deletion",
				"inviter_id", *finding.InviterID,
				"error", err,
			)
		}
	}
	return decision, nil
}

type RegistrationAbuseEventInput struct {
	UserID      int64
	Email       string
	ClientIP    string
	Fingerprint string
	InviterID   *int64
}

type registrationAbuseSQLStore struct {
	client *dbent.Client
}

func (s *registrationAbuseSQLStore) RecordAndEvaluate(
	ctx context.Context,
	event registrationAbuseEvent,
) (*registrationAbuseFinding, error) {
	if s == nil || s.client == nil {
		return &registrationAbuseFinding{}, nil
	}

	tx, err := s.client.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin registration abuse transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()

	if _, err := txClient.ExecContext(ctx, `
INSERT INTO registration_abuse_events (
    user_id,
    inviter_id,
    client_ip_hash,
    fingerprint_hash,
    email_domain,
    created_at
)
VALUES ($1, $2, $3, $4, $5, NOW())`,
		event.UserID,
		event.InviterID,
		event.ClientIPHash,
		event.FingerprintHash,
		event.EmailDomain,
	); err != nil {
		return nil, fmt.Errorf("record registration abuse event: %w", err)
	}

	finding := &registrationAbuseFinding{}
	now := time.Now().UTC()

	if event.ClientIPHash != "" {
		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE client_ip_hash = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			event.ClientIPHash, now.Add(-10*time.Minute), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseIPBurstLimit {
			finding.addSignal("ip_burst_10m", ids)
		}
		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE client_ip_hash = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			event.ClientIPHash, now.Add(-time.Hour), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseIPHourLimit {
			finding.addSignal("ip_burst_1h", ids)
		}
		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE client_ip_hash = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			event.ClientIPHash, now.Add(-24*time.Hour), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseIPDayLimit {
			finding.addSignal("ip_burst_24h", ids)
		}
	}

	if event.FingerprintHash != "" {
		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE fingerprint_hash = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			event.FingerprintHash, now.Add(-30*time.Minute), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseFingerprintLimit {
			finding.addSignal("fingerprint_burst", ids)
		}
	}

	if event.EmailDomain != "" && !isSharedRegistrationEmailDomain(event.EmailDomain) {
		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE email_domain = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			event.EmailDomain, now.Add(-time.Hour), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseEmailDomainLimit {
			finding.addSignal("email_domain_burst", ids)
		}
	}

	if event.InviterID != nil && *event.InviterID > 0 {
		inviterID := *event.InviterID
		finding.InviterID = &inviterID

		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE inviter_id = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			inviterID, now.Add(-10*time.Minute), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseInviterBurstLimit {
			finding.InviterQuotaReset = true
			finding.addSignal("inviter_burst_10m", ids)
		}
		if ids, err := queryRegistrationAbuseUserIDs(ctx, txClient,
			`SELECT user_id
			   FROM registration_abuse_events
			  WHERE inviter_id = $1 AND created_at >= $2
			  ORDER BY created_at DESC, id DESC
			  LIMIT $3`,
			inviterID, now.Add(-time.Hour), registrationAbuseEventQueryLimit,
		); err != nil {
			return nil, err
		} else if len(ids) >= registrationAbuseInviterHourLimit {
			finding.InviterQuotaReset = true
			finding.addSignal("inviter_burst_1h", ids)
		}
	}

	finding.OffendingUserIDs = uniquePositiveInt64s(finding.OffendingUserIDs)
	finding.Blocked = len(finding.Signals) > 0
	if finding.Blocked && !registrationAbuseContainsInt64(finding.OffendingUserIDs, event.UserID) {
		finding.OffendingUserIDs = append(finding.OffendingUserIDs, event.UserID)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit registration abuse transaction: %w", err)
	}
	return finding, nil
}

func (f *registrationAbuseFinding) addSignal(signal string, userIDs []int64) {
	f.Signals = append(f.Signals, signal)
	f.OffendingUserIDs = append(f.OffendingUserIDs, userIDs...)
}

func (s *registrationAbuseSQLStore) ResetInviterQuota(ctx context.Context, inviterID int64) error {
	if s == nil || s.client == nil || inviterID <= 0 {
		return nil
	}
	tx, err := s.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin abusive inviter quota reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txClient := tx.Client()

	result, err := txClient.ExecContext(ctx, `
UPDATE users
   SET balance = 0,
       frozen_balance = 0,
       updated_at = NOW()
 WHERE id = $1
   AND role <> 'admin'`, inviterID)
	if err != nil {
		return fmt.Errorf("clear abusive inviter wallet: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return tx.Commit()
	}

	if _, err := txClient.ExecContext(ctx, `
UPDATE user_affiliates
   SET aff_quota = 0,
       aff_frozen_quota = 0,
       aff_history_quota = 0,
       aff_count = (
           SELECT COUNT(*)::integer
             FROM user_affiliates child
             JOIN users child_user ON child_user.id = child.user_id
            WHERE child.inviter_id = $1
              AND child_user.deleted_at IS NULL
       ),
       updated_at = NOW()
 WHERE user_id = $1`, inviterID); err != nil {
		return fmt.Errorf("clear abusive inviter affiliate quota: %w", err)
	}
	if _, err := txClient.ExecContext(ctx, `
UPDATE user_platform_quotas
   SET daily_limit_usd = 0,
       weekly_limit_usd = 0,
       monthly_limit_usd = 0,
       daily_usage_usd = 0,
       weekly_usage_usd = 0,
       monthly_usage_usd = 0,
       updated_at = NOW()
 WHERE user_id = $1
   AND deleted_at IS NULL`, inviterID); err != nil {
		return fmt.Errorf("clear abusive inviter platform quota: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit abusive inviter quota reset: %w", err)
	}
	return nil
}

func (s *registrationAbuseSQLStore) MarkAccountsDeleted(ctx context.Context, userIDs []int64) error {
	if s == nil || s.client == nil || len(userIDs) == 0 {
		return nil
	}
	for _, userID := range uniquePositiveInt64s(userIDs) {
		if _, err := s.client.ExecContext(ctx, `
UPDATE registration_abuse_events
   SET deleted_at = COALESCE(deleted_at, NOW())
 WHERE user_id = $1`, userID); err != nil {
			return fmt.Errorf("mark registration abuse event deleted: user_id=%d: %w", userID, err)
		}
	}
	return nil
}

func (s *registrationAbuseSQLStore) DisableAccounts(ctx context.Context, userIDs []int64) error {
	if s == nil || s.client == nil || len(userIDs) == 0 {
		return nil
	}
	for _, userID := range uniquePositiveInt64s(userIDs) {
		if _, err := s.client.ExecContext(ctx, `
UPDATE users
   SET status = 'disabled',
       balance = 0,
       frozen_balance = 0,
       updated_at = NOW()
 WHERE id = $1
   AND role <> 'admin'`, userID); err != nil {
			return fmt.Errorf("disable abusive registration account: user_id=%d: %w", userID, err)
		}
	}
	return nil
}

type registrationAbuseQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func queryRegistrationAbuseUserIDs(
	ctx context.Context,
	queryer registrationAbuseQuerier,
	query string,
	args ...any,
) ([]int64, error) {
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query registration abuse events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ids := make([]int64, 0)
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan registration abuse event: %w", err)
		}
		ids = append(ids, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate registration abuse events: %w", err)
	}
	return ids, nil
}

func hashRegistrationAbuseValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.ToLower(value)))
	return hex.EncodeToString(sum[:])
}

func normalizeRegistrationAbuseDomain(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at >= len(email)-1 {
		return ""
	}
	domain := strings.TrimSpace(email[at+1:])
	if len(domain) > 255 {
		return ""
	}
	return domain
}

func normalizeRegistrationAbuseFingerprint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < registrationAbuseFingerprintMinimumLength ||
		len(value) > registrationAbuseFingerprintMaximumLength {
		return ""
	}
	return value
}

func isSharedRegistrationEmailDomain(domain string) bool {
	switch strings.ToLower(strings.TrimSpace(domain)) {
	case "gmail.com", "googlemail.com", "outlook.com", "hotmail.com",
		"live.com", "msn.com", "qq.com", "163.com", "126.com", "yeah.net",
		"icloud.com", "me.com", "yahoo.com", "proton.me", "protonmail.com":
		return true
	default:
		return false
	}
}

func uniquePositiveInt64s(values []int64) []int64 {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(values))
	result := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func registrationAbuseContainsInt64(values []int64, target int64) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneRegistrationAbuseInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func normalizeRegistrationRiskContext(input RegistrationRiskContext) RegistrationRiskContext {
	input.ClientIP = strings.TrimSpace(input.ClientIP)
	input.Fingerprint = normalizeRegistrationAbuseFingerprint(input.Fingerprint)
	return input
}

func registrationAbuseCleanupError(decision *RegistrationAbuseDecision) error {
	if decision == nil || !decision.CleanupDegraded {
		return nil
	}
	return errors.New("registration abuse cleanup degraded")
}
