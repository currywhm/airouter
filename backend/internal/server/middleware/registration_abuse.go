package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	registrationAbuseBodyLimit = 64 << 10
	registrationFingerprintMax = 256
)

type registrationAbuseAllower interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (middleware.AllowResult, error)
}

// RegistrationAbuseGuard adds layered anti-automation checks to public signup.
// It intentionally combines independent signals instead of relying on one IP
// counter, so distributed attempts and repeated signups from one browser are
// still visible and rate limited.
type RegistrationAbuseGuard struct {
	limiter registrationAbuseAllower
}

func NewRegistrationAbuseGuard(redisClient *redis.Client) *RegistrationAbuseGuard {
	if redisClient == nil {
		return &RegistrationAbuseGuard{}
	}
	return &RegistrationAbuseGuard{limiter: middleware.NewRateLimiter(redisClient)}
}

type registrationAbuseCheck struct {
	scope   string
	subject string
	limit   int
	window  time.Duration
}

func (g *RegistrationAbuseGuard) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if g == nil || g.limiter == nil {
			c.Next()
			return
		}

		email, fingerprint := readRegistrationIdentity(c)
		checks := make([]registrationAbuseCheck, 0, 5)

		clientIP := SecurityClientIP(c)
		if isPubliclyRoutableClientIP(clientIP) {
			checks = append(checks,
				registrationAbuseCheck{scope: "ip-burst", subject: clientIP, limit: 3, window: 10 * time.Minute},
				registrationAbuseCheck{scope: "ip-hour", subject: clientIP, limit: 10, window: time.Hour},
				registrationAbuseCheck{scope: "ip-day", subject: clientIP, limit: 30, window: 24 * time.Hour},
			)
		}
		if fingerprint != "" {
			checks = append(checks, registrationAbuseCheck{
				scope: "fingerprint", subject: fingerprint, limit: 8, window: 30 * time.Minute,
			})
		}
		if domain := registrationEmailDomain(email); domain != "" {
			checks = append(checks, registrationAbuseCheck{
				scope: "email-domain", subject: domain, limit: 30, window: time.Hour,
			})
		}

		for _, check := range checks {
			result, err := g.limiter.Allow(
				c.Request.Context(),
				"registration-abuse:"+check.scope+":"+stableRegistrationKey(check.subject),
				check.limit,
				check.window,
			)
			if err != nil {
				slog.Error("registration abuse guard unavailable",
					"scope", check.scope,
					"error", err,
				)
				abortRegistrationAbuse(c, "guard_unavailable", result.Count, check.window)
				return
			}
			if !result.Allowed {
				slog.Warn("registration blocked by abuse guard",
					"scope", check.scope,
					"count", result.Count,
					"limit", check.limit,
					"client_ip", clientIP,
				)
				abortRegistrationAbuse(c, check.scope, result.Count, result.RetryAfter)
				return
			}
		}

		c.Next()
	}
}

func readRegistrationIdentity(c *gin.Context) (string, string) {
	fingerprint := normalizeRegistrationFingerprint(c.GetHeader("X-Client-Fingerprint"))
	if c.Request == nil || c.Request.Body == nil {
		return "", fingerprint
	}

	orig := c.Request.Body
	raw, err := io.ReadAll(io.LimitReader(orig, registrationAbuseBodyLimit+1))
	if err != nil {
		c.Request.Body = orig
		return "", fingerprint
	}
	c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), orig))

	var body struct {
		Email             string `json:"email"`
		ClientFingerprint string `json:"client_fingerprint"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", fingerprint
	}
	if fingerprint == "" {
		fingerprint = normalizeRegistrationFingerprint(body.ClientFingerprint)
	}
	return strings.ToLower(strings.TrimSpace(body.Email)), fingerprint
}

func normalizeRegistrationFingerprint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 8 || len(value) > registrationFingerprintMax {
		return ""
	}
	return value
}

func registrationEmailDomain(email string) string {
	at := strings.LastIndexByte(email, '@')
	if at <= 0 || at >= len(email)-1 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(email[at+1:]))
}

func stableRegistrationKey(subject string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(subject))))
	return hex.EncodeToString(sum[:])
}

func abortRegistrationAbuse(c *gin.Context, signal string, count int64, retryAfter time.Duration) {
	SetAuditExtra(c, map[string]any{
		"abuse_signal":  signal,
		"attempt_count": count,
	})
	if retryAfter <= 0 {
		retryAfter = time.Minute
	}
	seconds := int64(retryAfter / time.Second)
	if retryAfter%time.Second > 0 {
		seconds++
	}
	c.Header("Retry-After", strconv.FormatInt(seconds, 10))
	AbortWithError(
		c,
		http.StatusTooManyRequests,
		"REGISTRATION_RATE_LIMITED",
		"Too many registration attempts from this source. Please try again later.",
	)
}
