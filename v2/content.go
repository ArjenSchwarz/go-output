package output

import (
	"crypto/rand"
	"encoding"
	"fmt"
)

// ContentType identifies the type of content
type ContentType int

const (
	ContentTypeTable ContentType = iota
	ContentTypeText
	ContentTypeRaw
	ContentTypeSection
)

// String returns the string representation of the ContentType
func (ct ContentType) String() string {
	switch ct {
	case ContentTypeTable:
		return "table"
	case ContentTypeText:
		return "text"
	case ContentTypeRaw:
		return "raw"
	case ContentTypeSection:
		return "section"
	default:
		return "unknown"
	}
}

// Content is the core interface all content must implement
type Content interface {
	// Type returns the content type
	Type() ContentType

	// ID returns a unique identifier for this content
	ID() string

	// Encoding interfaces for efficient serialization
	encoding.TextAppender
	encoding.BinaryAppender
}

// generateID creates a unique identifier for content
func generateID() string {
	// Generate 8 random bytes
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple counter if crypto/rand fails
		return fmt.Sprintf("content-%d", len(bytes))
	}
	return fmt.Sprintf("content-%x", bytes)
}
