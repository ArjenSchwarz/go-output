package output

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// Regression tests for T-1543: built documents expose mutable section contents.
//
// Document.GetContents copies only the top-level slice, so callers could
// type-assert a returned content value to *SectionContent and call AddContent
// after Builder.Build(), mutating the finalized document. This changed
// rendered output post-build and raced with renderers reading
// section.Contents(). Sections are now frozen at Build(): post-build
// AddContent calls are silently ignored.

// TestSectionContent_PostBuildMutation_DoesNotChangeRenderedOutput verifies
// that mutating a section obtained from a built document cannot change what
// renders.
//
// Bug: AddContent on a section from GetContents() appended to the built
// document, so the injected content appeared in subsequent renders.
// Expected: rendering a built document yields identical output before and
// after an attempted post-build mutation.
func TestSectionContent_PostBuildMutation_DoesNotChangeRenderedOutput(t *testing.T) {
	doc := New().
		Section("Report", func(b *Builder) {
			b.Text("original content")
		}).
		Build()

	renderer := &markdownRenderer{}
	before, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("unexpected render error before mutation: %v", err)
	}

	section, ok := doc.GetContents()[0].(*SectionContent)
	if !ok {
		t.Fatalf("expected *SectionContent, got %T", doc.GetContents()[0])
	}
	section.AddContent(NewTextContent("injected after build"))

	after, err := renderer.Render(context.Background(), doc)
	if err != nil {
		t.Fatalf("unexpected render error after mutation: %v", err)
	}

	if got, want := string(after), string(before); got != want {
		t.Errorf("rendered output changed after post-build mutation:\nbefore: %q\nafter:  %q", want, got)
	}
	if strings.Contains(string(after), "injected after build") {
		t.Error("post-build injected content appeared in rendered output")
	}
	if got, want := len(section.Contents()), 1; got != want {
		t.Errorf("section contents mutated after build: got %d entries, want %d", got, want)
	}
}

// TestSectionContent_PostBuildMutation_NestedSectionAlsoFrozen verifies that
// sections nested inside other sections of a built document are frozen too,
// since Contents() exposes the same nested pointers.
func TestSectionContent_PostBuildMutation_NestedSectionAlsoFrozen(t *testing.T) {
	doc := New().
		Section("Outer", func(b *Builder) {
			b.Section("Inner", func(nb *Builder) {
				nb.Text("inner content")
			}, WithLevel(1))
		}).
		Build()

	outer, ok := doc.GetContents()[0].(*SectionContent)
	if !ok {
		t.Fatalf("expected *SectionContent, got %T", doc.GetContents()[0])
	}
	inner, ok := outer.Contents()[0].(*SectionContent)
	if !ok {
		t.Fatalf("expected nested *SectionContent, got %T", outer.Contents()[0])
	}

	inner.AddContent(NewTextContent("injected into nested section"))

	if got, want := len(inner.Contents()), 1; got != want {
		t.Errorf("nested section contents mutated after build: got %d entries, want %d", got, want)
	}
}

// TestSectionContent_PostBuildMutation_SectionInCollapsibleFrozen verifies
// that a section reachable through a collapsible section of a built document
// is frozen, since DefaultCollapsibleSection.Content() also shares pointers.
func TestSectionContent_PostBuildMutation_SectionInCollapsibleFrozen(t *testing.T) {
	section := NewSectionContent("Wrapped Section")
	section.AddContent(NewTextContent("wrapped content"))

	doc := New().
		AddCollapsibleSection("Collapsible", []Content{section}).
		Build()

	collapsible, ok := doc.GetContents()[0].(*DefaultCollapsibleSection)
	if !ok {
		t.Fatalf("expected *DefaultCollapsibleSection, got %T", doc.GetContents()[0])
	}
	wrapped, ok := collapsible.Content()[0].(*SectionContent)
	if !ok {
		t.Fatalf("expected wrapped *SectionContent, got %T", collapsible.Content()[0])
	}

	wrapped.AddContent(NewTextContent("injected into wrapped section"))

	if got, want := len(wrapped.Contents()), 1; got != want {
		t.Errorf("wrapped section contents mutated after build: got %d entries, want %d", got, want)
	}
}

// TestSectionContent_PreBuildAddContentStillWorks verifies that freezing at
// Build() does not affect the legitimate building phase: sections constructed
// directly can still be populated before the document is built.
func TestSectionContent_PreBuildAddContentStillWorks(t *testing.T) {
	section := NewSectionContent("Manual Section")
	section.AddContent(NewTextContent("first"))

	builder := New().AddContent(section)
	section.AddContent(NewTextContent("second"))

	doc := builder.Build()

	built, ok := doc.GetContents()[0].(*SectionContent)
	if !ok {
		t.Fatalf("expected *SectionContent, got %T", doc.GetContents()[0])
	}
	if got, want := len(built.Contents()), 2; got != want {
		t.Errorf("pre-build AddContent lost content: got %d entries, want %d", got, want)
	}
}

// TestSectionContent_CloneOfBuiltSectionIsMutable verifies the documented
// escape hatch: Clone() of a frozen section returns a caller-owned copy that
// can be mutated without affecting the built document.
func TestSectionContent_CloneOfBuiltSectionIsMutable(t *testing.T) {
	doc := New().
		Section("Report", func(b *Builder) {
			b.Text("original content")
		}).
		Build()

	section := doc.GetContents()[0].(*SectionContent)
	clone := section.Clone().(*SectionContent)
	clone.AddContent(NewTextContent("added to clone"))

	if got, want := len(clone.Contents()), 2; got != want {
		t.Errorf("clone of built section not mutable: got %d entries, want %d", got, want)
	}
	if got, want := len(section.Contents()), 1; got != want {
		t.Errorf("mutating clone affected built section: got %d entries, want %d", got, want)
	}
}

// TestSectionContent_ConcurrentRenderAndPostBuildMutation exercises the race
// between renderers recursively reading section.Contents() and a caller
// attempting SectionContent.AddContent on a built document. Run with -race.
//
// Bug: AddContent appended to s.contents with no synchronization while
// renders read the same slice, a data race.
// Expected: post-build AddContent is a no-op, so concurrent render plus
// attempted mutation is race-free and output is stable.
func TestSectionContent_ConcurrentRenderAndPostBuildMutation(t *testing.T) {
	doc := New().
		Section("Concurrent Section", func(b *Builder) {
			b.Text("stable content")
		}).
		Build()

	section, ok := doc.GetContents()[0].(*SectionContent)
	if !ok {
		t.Fatalf("expected *SectionContent, got %T", doc.GetContents()[0])
	}

	renderer := &markdownRenderer{}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)
		go func() {
			defer wg.Done()
			for range 25 {
				if _, err := renderer.Render(context.Background(), doc); err != nil {
					t.Errorf("unexpected render error during concurrent mutation: %v", err)
				}
			}
		}()
		go func() {
			defer wg.Done()
			for range 25 {
				section.AddContent(NewTextContent("post-build mutation attempt"))
			}
		}()
	}
	wg.Wait()

	if got, want := len(section.Contents()), 1; got != want {
		t.Errorf("section contents mutated after build: got %d entries, want %d", got, want)
	}
}
