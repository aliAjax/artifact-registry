package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"strings"
)

const SHA256Prefix = "sha256:"

type Digest string

func ParseDigest(v string) (Digest, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if !strings.HasPrefix(v, SHA256Prefix) || len(v) != len(SHA256Prefix)+64 {
		return "", ErrInvalidDigest
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(v, SHA256Prefix)); err != nil {
		return "", ErrInvalidDigest
	}
	return Digest(v), nil
}
func MustDigest(v string) Digest {
	d, err := ParseDigest(v)
	if err != nil {
		panic(err)
	}
	return d
}
func (d Digest) String() string    { return string(d) }
func (d Digest) Algorithm() string { return "sha256" }
func (d Digest) Hex() string       { return strings.TrimPrefix(string(d), SHA256Prefix) }
func (d Digest) Validate() error   { _, err := ParseDigest(string(d)); return err }
func DigestBytes(data []byte) Digest {
	sum := sha256.Sum256(data)
	return Digest(SHA256Prefix + hex.EncodeToString(sum[:]))
}
func DigestReader(r io.Reader) (Digest, int64, error) {
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", n, err
	}
	return digestHash(h), n, nil
}
func NewDigester() hash.Hash        { return sha256.New() }
func DigestHash(h hash.Hash) Digest { return digestHash(h) }
func digestHash(h hash.Hash) Digest { return Digest(SHA256Prefix + hex.EncodeToString(h.Sum(nil))) }
func VerifyDigest(r io.Reader, expected Digest) (int64, error) {
	actual, n, err := DigestReader(r)
	if err != nil {
		return n, err
	}
	if actual != expected {
		return n, fmt.Errorf("%w: expected %s got %s", ErrDigestMismatch, expected, actual)
	}
	return n, nil
}
