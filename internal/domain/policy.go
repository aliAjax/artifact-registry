package domain

import (
	"fmt"
	"strings"
	"time"
)

type RetentionPolicy struct {
	ID               string        `json:"id"`
	RepositoryPrefix string        `json:"repositoryPrefix"`
	KeepLast         int           `json:"keepLast"`
	MaxAge           time.Duration `json:"maxAge"`
	KeepTagged       bool          `json:"keepTagged"`
	CreatedAt        time.Time     `json:"createdAt"`
	Enabled          bool          `json:"enabled"`
}

func (p RetentionPolicy) Validate() error {
	if p.ID == "" {
		return ValidationError{"policy.id", "required"}
	}
	if p.KeepLast < 0 {
		return ValidationError{"policy.keepLast", "negative"}
	}
	if p.MaxAge < 0 {
		return ValidationError{"policy.maxAge", "negative"}
	}
	return nil
}
func (p RetentionPolicy) Applies(repo RepositoryName) bool {
	return p.Enabled && strings.HasPrefix(repo.String(), p.RepositoryPrefix)
}
func (p RetentionPolicy) Explain() string {
	return fmt.Sprintf("keep last %d, max age %s, keep tagged %t", p.KeepLast, p.MaxAge, p.KeepTagged)
}

type Quota struct {
	Tenant    string    `json:"tenant"`
	Limit     int64     `json:"limit"`
	Used      int64     `json:"used"`
	Reserved  int64     `json:"reserved"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   int64     `json:"version"`
}

func (q Quota) Remaining() int64 { return q.Limit - q.Used - q.Reserved }
func (q *Quota) Reserve(n int64, now time.Time) error {
	if n < 0 {
		return ValidationError{"reservation", "negative"}
	}
	if n > q.Remaining() {
		return ErrQuotaExceeded
	}
	q.Reserved += n
	q.UpdatedAt = now
	q.Version++
	return nil
}
func (q *Quota) Commit(reserved, actual int64, now time.Time) error {
	if reserved < 0 || actual < 0 || reserved > q.Reserved {
		return ErrQuotaExceeded
	}
	q.Reserved -= reserved
	q.Used += actual
	q.UpdatedAt = now
	q.Version++
	return nil
}
func (q *Quota) Release(n int64, now time.Time) error {
	if n < 0 || n > q.Reserved {
		return ErrQuotaExceeded
	}
	q.Reserved -= n
	q.UpdatedAt = now
	q.Version++
	return nil
}

type AuditEntry struct {
	ID           string            `json:"id"`
	At           time.Time         `json:"at"`
	Actor        string            `json:"actor"`
	Action       string            `json:"action"`
	Resource     string            `json:"resource"`
	Detail       map[string]string `json:"detail,omitempty"`
	PreviousHash string            `json:"previousHash"`
	Hash         string            `json:"hash"`
}
