package common

import (
	"strings"

	"github.com/google/uuid"
)

// ShortID8 generates an 8-character ID from UUID (first 8 chars).
// Suitable for low-volume entities like companies.
func ShortID8() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:8]
}

// ShortID12 generates a 12-character ID from UUID (first 12 chars).
// Suitable for higher-volume entities like sessions.
func ShortID12() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
}