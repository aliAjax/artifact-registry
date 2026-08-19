// Package domain defines storage-independent registry concepts and invariants.
package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRepository      = errors.New("invalid repository name")
	ErrInvalidDigest          = errors.New("invalid content digest")
	ErrInvalidMediaType       = errors.New("unsupported media type")
	ErrBlobNotFound           = errors.New("blob not found")
	ErrManifestNotFound       = errors.New("manifest not found")
	ErrTagNotFound            = errors.New("tag not found")
	ErrTagImmutable           = errors.New("tag is immutable")
	ErrUploadNotFound         = errors.New("upload session not found")
	ErrUploadExpired          = errors.New("upload session expired")
	ErrUploadConflict         = errors.New("upload range conflicts with recorded data")
	ErrUploadOutOfOrder       = errors.New("upload chunk is outside contiguous upload window")
	ErrDigestMismatch         = errors.New("provided digest does not match content")
	ErrQuotaExceeded          = errors.New("repository quota exceeded")
	ErrManifestBlocked        = errors.New("artifact is blocked by policy")
	ErrPreconditionFailed     = errors.New("precondition failed")
	ErrInvalidRange           = errors.New("invalid byte range")
	ErrNotImplemented         = errors.New("capability not implemented")
	ErrInvalidStateTransition = errors.New("invalid lifecycle transition")
	ErrLeaseActive            = errors.New("replication lease is active")
	ErrIdempotencyConflict    = errors.New("idempotency key conflicts with prior request")
)

type ValidationError struct{ Field, Reason string }

func (e ValidationError) Error() string { return fmt.Sprintf("invalid %s: %s", e.Field, e.Reason) }

type ConflictError struct{ Resource, Detail string }

func (e ConflictError) Error() string {
	return fmt.Sprintf("conflict for %s: %s", e.Resource, e.Detail)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrBlobNotFound) || errors.Is(err, ErrManifestNotFound) || errors.Is(err, ErrTagNotFound) || errors.Is(err, ErrUploadNotFound)
}
