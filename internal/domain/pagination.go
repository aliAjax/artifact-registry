package domain

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

type Cursor struct {
	Value     string `json:"value"`
	Version   int64  `json:"version"`
	Direction string `json:"direction"`
}

func (c Cursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.Value + "|" + strconv.FormatInt(c.Version, 10) + "|" + c.Direction))
}
func DecodeCursor(v string) (Cursor, error) {
	if strings.TrimSpace(v) == "" {
		return Cursor{}, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(v)
	if err != nil {
		return Cursor{}, ValidationError{"cursor", "invalid base64"}
	}
	parts := strings.Split(string(raw), "|")
	if len(parts) != 3 {
		return Cursor{}, ValidationError{"cursor", "invalid shape"}
	}
	ver, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return Cursor{}, ValidationError{"cursor", "invalid version"}
	}
	return Cursor{Value: parts[0], Version: ver, Direction: parts[2]}, nil
}

type PageInfo struct {
	HasNext     bool   `json:"hasNext"`
	HasPrevious bool   `json:"hasPrevious"`
	StartCursor string `json:"startCursor,omitempty"`
	EndCursor   string `json:"endCursor,omitempty"`
	Total       int    `json:"total,omitempty"`
}

func NewPageInfo[T any](items []T, next, previous string, total int) PageInfo {
	return PageInfo{HasNext: next != "", HasPrevious: previous != "", StartCursor: previous, EndCursor: next, Total: total}
}

type SortField struct {
	Name       string
	Descending bool
}

func ParseSort(v string) []SortField {
	out := []SortField{}
	for _, part := range strings.Split(v, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		desc := strings.HasPrefix(part, "-")
		if desc {
			part = strings.TrimPrefix(part, "-")
		}
		out = append(out, SortField{Name: part, Descending: desc})
	}
	return out
}
func ValidateSort(fields []SortField, allowed map[string]bool) error {
	for _, f := range fields {
		if !allowed[f.Name] {
			return fmt.Errorf("unsupported sort field %s", f.Name)
		}
	}
	return nil
}
