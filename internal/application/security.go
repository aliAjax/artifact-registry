package application

import (
	"artifact-registry/internal/domain"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type SecurityService struct {
	Secret []byte
	Clock  Clock
	Audits AuditStore
}

func (s *SecurityService) Sign(subject string, payload []byte) string {
	h := hmac.New(sha256.New, s.Secret)
	h.Write([]byte(subject))
	h.Write([]byte{0})
	h.Write(payload)
	return "hmac-sha256=" + hex.EncodeToString(h.Sum(nil))
}
func (s *SecurityService) Verify(subject string, payload []byte, signature string) bool {
	expected := s.Sign(subject, payload)
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}
func (s *SecurityService) AuditSignature(ctx context.Context, actor, subject string, payload []byte, signature string) error {
	ok := s.Verify(subject, payload, signature)
	action := "signature.rejected"
	if ok {
		action = "signature.accepted"
	}
	if s.Audits != nil {
		_ = s.Audits.Append(ctx, domain.AuditEntry{ID: hex.EncodeToString(sha256.New().Sum([]byte(subject))), At: s.now(), Actor: actor, Action: action, Resource: subject, Detail: map[string]string{"valid": boolText(ok)}})
	}
	if !ok {
		return domain.ErrDigestMismatch
	}
	return nil
}
func boolText(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
func (s *SecurityService) now() time.Time {
	if s.Clock != nil {
		return s.Clock.Now().UTC()
	}
	return time.Now().UTC()
}

type AccessDecision struct {
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason"`
	Subject   string    `json:"subject"`
	Action    string    `json:"action"`
	Tenant    string    `json:"tenant"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (d AccessDecision) Error() string {
	if d.Allowed {
		return ""
	}
	return d.Reason
}
func DecideRead(tenant, subject string, now, timeLimit time.Time) AccessDecision {
	if tenant == "" {
		return AccessDecision{Reason: "tenant context required", Subject: subject}
	}
	if !timeLimit.IsZero() && now.After(timeLimit) {
		return AccessDecision{Reason: "authorization expired", Tenant: tenant, Subject: subject}
	}
	return AccessDecision{Allowed: true, Reason: "tenant scope matched", Tenant: tenant, Subject: subject, Action: "read", ExpiresAt: timeLimit}
}
func DecideWrite(tenant, subject string, immutable bool, hasExisting bool) AccessDecision {
	if tenant == "" {
		return AccessDecision{Reason: "tenant context required", Subject: subject}
	}
	if immutable && hasExisting {
		return AccessDecision{Reason: "immutable resource", Tenant: tenant, Subject: subject}
	}
	return AccessDecision{Allowed: true, Reason: "write accepted", Tenant: tenant, Subject: subject, Action: "write"}
}
