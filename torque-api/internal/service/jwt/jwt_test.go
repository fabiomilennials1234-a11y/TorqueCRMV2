package jwt_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
	"github.com/milennials/torque-api/internal/service/jwt"
)

const testSecret = "test-secret-please-use-at-least-thirty-two-bytes"

func mustSvc(t *testing.T, ttl time.Duration) *jwt.Service {
	t.Helper()
	s, err := jwt.New([]byte(testSecret), ttl)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return s
}

func sampleSession() (uuid.UUID, domain.Session) {
	userID := uuid.New()
	return userID, domain.Session{
		UserID:         userID,
		OrganizationID: uuid.New(),
		TeamMemberID:   uuid.New(),
		Role:           domain.RoleAdmin,
		IsMaster:       false,
		UIMode:         domain.UIModeManager,
	}
}

func TestNew_RejectsShortSecret(t *testing.T) {
	t.Parallel()
	if _, err := jwt.New([]byte("short"), 15*time.Minute); err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestIssueParse_RoundTrip(t *testing.T) {
	t.Parallel()
	svc := mustSvc(t, 15*time.Minute)
	uid, sess := sampleSession()
	raw, exp, err := svc.Issue(uid, sess)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if raw == "" || exp.Before(time.Now()) {
		t.Fatalf("bad output: raw=%q exp=%v", raw, exp)
	}
	claims, err := svc.Parse(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sess2, err := claims.Session()
	if err != nil {
		t.Fatalf("session rehydrate: %v", err)
	}
	if sess2.UserID != sess.UserID || sess2.OrganizationID != sess.OrganizationID ||
		sess2.Role != sess.Role || sess2.TeamMemberID != sess.TeamMemberID {
		t.Fatalf("round-trip drift: %+v != %+v", sess2, sess)
	}
}

func TestParse_RejectsBadSignature(t *testing.T) {
	t.Parallel()
	svc := mustSvc(t, 15*time.Minute)
	uid, sess := sampleSession()
	raw, _, _ := svc.Issue(uid, sess)
	// Flip the last signature byte.
	tampered := raw[:len(raw)-1] + swapChar(raw[len(raw)-1])
	if _, err := svc.Parse(tampered); !errors.Is(err, jwt.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParse_RejectsWrongKey(t *testing.T) {
	t.Parallel()
	svc := mustSvc(t, 15*time.Minute)
	uid, sess := sampleSession()
	raw, _, _ := svc.Issue(uid, sess)
	// Different secret → invalid signature.
	other, _ := jwt.New([]byte(strings.Repeat("x", 32)), 15*time.Minute)
	if _, err := other.Parse(raw); !errors.Is(err, jwt.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestParse_RejectsExpired(t *testing.T) {
	t.Parallel()
	svc := mustSvc(t, 1*time.Millisecond)
	uid, sess := sampleSession()
	raw, _, _ := svc.Issue(uid, sess)
	time.Sleep(10 * time.Millisecond)
	if _, err := svc.Parse(raw); !errors.Is(err, jwt.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for expired, got %v", err)
	}
}

func swapChar(c byte) string {
	if c == 'A' {
		return "B"
	}
	return "A"
}
