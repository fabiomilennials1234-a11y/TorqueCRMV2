// Package jwt issues and verifies Torque session JWTs.
//
// A Torque access token is a short-lived (15 min) HS256 JWT. Refresh tokens
// are opaque (see repository/refresh) and live in cookies only — never JWTs.
//
// The signing key is loaded once from env and held as []byte. The service
// itself is stateless; a fresh value may be constructed per test and discarded.
package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/milennials/torque-api/internal/domain"
)

// Issuer is the canonical `iss` claim, also used as an audience sanity check.
const Issuer = "torque-api"

// Claims is the Torque-specific JWT claim bundle.
//
// We store the minimum to avoid a DB roundtrip on every request. Permissions
// are NOT in the token — they can revoke between issue and request, so the
// middleware loads them fresh (cached upstream) instead.
type Claims struct {
	OrgID        uuid.UUID    `json:"org_id"`
	TeamMemberID uuid.UUID    `json:"tm_id"`
	Role         domain.Role  `json:"role"`
	IsMaster     bool         `json:"master,omitempty"`
	UIMode       domain.UIMode `json:"ui_mode"`
	jwtv5.RegisteredClaims
}

// Service issues and validates access tokens.
type Service struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time // seam for deterministic tests
}

// New returns a service bound to the provided secret.
// The secret MUST be at least 32 bytes; short keys are refused so a misconfigured
// env var never silently reduces strength.
func New(secret []byte, ttl time.Duration) (*Service, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("jwt secret must be >= 32 bytes, got %d", len(secret))
	}
	if ttl <= 0 || ttl > time.Hour {
		return nil, fmt.Errorf("jwt ttl out of range (0,1h]: %s", ttl)
	}
	return &Service{secret: secret, ttl: ttl, now: time.Now}, nil
}

// Issue mints a signed access token for the given session.
//
// `userID` is the JWT `sub`. The returned string is the compact form ready to
// set as a cookie value.
func (s *Service) Issue(userID uuid.UUID, sess domain.Session) (string, time.Time, error) {
	now := s.now().UTC()
	exp := now.Add(s.ttl)

	claims := Claims{
		OrgID:        sess.OrganizationID,
		TeamMemberID: sess.TeamMemberID,
		Role:         sess.Role,
		IsMaster:     sess.IsMaster,
		UIMode:       sess.UIMode,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    Issuer,
			Subject:   userID.String(),
			IssuedAt:  jwtv5.NewNumericDate(now),
			NotBefore: jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(exp),
			ID:        uuid.NewString(),
		},
	}

	tok := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, exp, nil
}

// ErrInvalidToken is returned for any validation failure. Callers must map it
// to 401 without leaking the underlying reason to the client.
var ErrInvalidToken = errors.New("invalid or expired token")

// Parse validates a signed token and returns its claims.
// Signature, expiry, and issuer are all enforced.
func (s *Service) Parse(raw string) (*Claims, error) {
	parsed, err := jwtv5.ParseWithClaims(raw, &Claims{}, func(t *jwtv5.Token) (any, error) {
		// Reject alg=none and any non-HMAC method. This is how jwt.io exploits
		// start; pin to HS256 explicitly.
		if t.Method.Alg() != jwtv5.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return s.secret, nil
	},
		jwtv5.WithIssuer(Issuer),
		jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}),
		jwtv5.WithExpirationRequired(),
	)
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// Session rehydrates a domain.Session from claims. The caller is expected to
// have already Parse()'d the token.
func (c *Claims) Session() (domain.Session, error) {
	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return domain.Session{}, fmt.Errorf("sub not a uuid: %w", err)
	}
	if !c.Role.IsValid() {
		return domain.Session{}, fmt.Errorf("invalid role in claims: %s", c.Role)
	}
	return domain.Session{
		UserID:         userID,
		OrganizationID: c.OrgID,
		TeamMemberID:   c.TeamMemberID,
		Role:           c.Role,
		IsMaster:       c.IsMaster,
		UIMode:         c.UIMode,
		IssuedAt:       c.IssuedAt.Time,
		ExpiresAt:      c.ExpiresAt.Time,
	}, nil
}
