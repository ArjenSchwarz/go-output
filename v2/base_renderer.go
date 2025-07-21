// Package output provides a flexible output generation library for Go applications.
// It supports multiple output formats (JSON, YAML, CSV, HTML, Markdown, etc.)
// with a document-builder pattern that eliminates global state and provides
// thread-safe operations.
package output

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// baseRenderer provides common functionality shared by all renderer implementations
type baseRenderer struct {
	mu sync.RWMutex
}

// renderDocument handles the core document rendering logic with context cancellation
func (b *baseRenderer) renderDocument(ctx context.Context, doc *Document, renderFunc func(Content) ([]byte, error)) ([]byte, error) {
	if doc == nil {
		return nil, fmt.Errorf("document cannot be nil")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var result bytes.Buffer
	contents := doc.GetContents()

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		contentBytes, err := renderFunc(content)
		if err != nil {
			return nil, fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}

		if i > 0 && len(contentBytes) > 0 {
			result.WriteByte('\n')
		}

		result.Write(contentBytes)
	}

	return result.Bytes(), nil
}

// renderDocumentTo handles streaming document rendering with context cancellation
func (b *baseRenderer) renderDocumentTo(ctx context.Context, doc *Document, w io.Writer, renderFunc func(Content, io.Writer) error) error {
	if doc == nil {
		return fmt.Errorf("document cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	contents := doc.GetContents()

	for i, content := range contents {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if i > 0 {
			if _, err := w.Write([]byte{'\n'}); err != nil {
				return fmt.Errorf("failed to write separator: %w", err)
			}
		}

		if err := renderFunc(content, w); err != nil {
			return fmt.Errorf("failed to render content %s: %w", content.ID(), err)
		}
	}

	return nil
}

// renderContent provides a default content rendering implementation
func (b *baseRenderer) renderContent(content Content) ([]byte, error) {
	if content == nil {
		return nil, fmt.Errorf("content cannot be nil")
	}

	// Use the content's own AppendText method for basic rendering
	return content.AppendText(nil)
}

// renderContentTo provides a default streaming content rendering implementation
func (b *baseRenderer) renderContentTo(content Content, w io.Writer) error {
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	contentBytes, err := content.AppendText(nil)
	if err != nil {
		return err
	}

	_, err = w.Write(contentBytes)
	return err
}
