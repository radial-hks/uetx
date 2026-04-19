package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	mrand "math/rand"
	"strings"
)

// GUIDFunc generates a 32-character uppercase hex GUID.
type GUIDFunc func() string

// NewGUID generates a cryptographically random GUID.
func NewGUID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return strings.ToUpper(hex.EncodeToString(b))
}

// NewSeededGUIDFunc returns a deterministic GUIDFunc for testing.
func NewSeededGUIDFunc(seed int64) GUIDFunc {
	r := mrand.New(mrand.NewSource(seed))
	return func() string {
		b := make([]byte, 16)
		r.Read(b)
		return strings.ToUpper(hex.EncodeToString(b))
	}
}
