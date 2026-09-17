package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	internalmiddleware "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type registrationAbuseAllowerStub struct {
	keys       []string
	blockScope string
}

func (s *registrationAbuseAllowerStub) Allow(_ context.Context, key string, _ int, _ time.Duration) (internalmiddleware.AllowResult, error) {
	s.keys = append(s.keys, key)
	if s.blockScope != "" && strings.Contains(key, ":"+s.blockScope+":") {
		return internalmiddleware.AllowResult{
			Allowed:    false,
			Count:      9,
			RetryAfter: time.Minute,
		}, nil
	}
	return internalmiddleware.AllowResult{Allowed: true, Count: 1}, nil
}

func TestRegistrationAbuseGuardPreservesBodyAndUsesLayeredSignals(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := &registrationAbuseAllowerStub{}
	guard := &RegistrationAbuseGuard{limiter: limiter}

	router := gin.New()
	router.POST("/register", guard.Middleware(), func(c *gin.Context) {
		var body struct {
			Email string `json:"email"`
		}
		require.NoError(t, c.ShouldBindJSON(&body))
		c.String(http.StatusOK, body.Email)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		bytes.NewBufferString(`{"email":"new-user@example.com"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Fingerprint", "browser-fingerprint-123")
	req.RemoteAddr = "203.0.113.10:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "new-user@example.com", recorder.Body.String())
	require.Len(t, limiter.keys, 5)
	require.Condition(t, func() bool {
		for _, key := range limiter.keys {
			if strings.Contains(key, ":fingerprint:") {
				return true
			}
		}
		return false
	})
	require.Condition(t, func() bool {
		for _, key := range limiter.keys {
			if strings.Contains(key, ":email-domain:") {
				return true
			}
		}
		return false
	})
}

func TestRegistrationAbuseGuardBlocksSuspiciousBurst(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := &registrationAbuseAllowerStub{blockScope: "ip-burst"}
	guard := &RegistrationAbuseGuard{limiter: limiter}

	router := gin.New()
	router.POST("/register", guard.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		bytes.NewBufferString(`{"email":"burst@example.com"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "203.0.113.11:12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Contains(t, recorder.Body.String(), "REGISTRATION_RATE_LIMITED")
	require.NotEmpty(t, recorder.Header().Get("Retry-After"))
}
