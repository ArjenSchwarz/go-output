package output

import (
	"sync"
	"testing"
)

// testContent is a test implementation of the Content interface for document tests
type testContent struct {
	id          string
	contentType ContentType
}

func (m *testContent) Type() ContentType {
	return m.contentType
}

func (m *testContent) ID() string {
	return m.id
}

func (m *testContent) AppendText(b []byte) ([]byte, error) {
	return append(b, []byte(m.id)...), nil
}

func (m *testContent) AppendBinary(b []byte) ([]byte, error) {
	return append(b, []byte(m.id)...), nil
}

func TestNew(t *testing.T) {
	builder := New()
	if builder == nil {
		t.Fatal("New() returned nil")
	}
	if builder.doc == nil {
		t.Fatal("New() created builder with nil document")
	}
	if builder.doc.metadata == nil {
		t.Fatal("New() created document with nil metadata")
	}
	if len(builder.doc.contents) != 0 {
		t.Errorf("New() created document with non-empty contents: got %d, want 0", len(builder.doc.contents))
	}
}

func TestBuilder_Build(t *testing.T) {
	builder := New()

	// Add some metadata
	builder.SetMetadata("key1", "value1")
	builder.SetMetadata("key2", 42)

	// Build the document
	doc := builder.Build()

	if doc == nil {
		t.Fatal("Build() returned nil")
	}

	// Verify metadata
	metadata := doc.GetMetadata()
	if len(metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(metadata))
	}
	if metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1='value1', got %v", metadata["key1"])
	}
	if metadata["key2"] != 42 {
		t.Errorf("Expected metadata key2=42, got %v", metadata["key2"])
	}

	// Verify that the builder is cleared after Build()
	if builder.doc != nil {
		t.Error("Builder should clear document reference after Build()")
	}

	// Attempting to use builder after Build should not panic
	builder.SetMetadata("key3", "value3") // Should not panic
}

func TestBuilder_SetMetadata(t *testing.T) {
	builder := New()

	// Test fluent API
	result := builder.SetMetadata("key1", "value1").SetMetadata("key2", 123)
	if result != builder {
		t.Error("SetMetadata should return the same builder instance (fluent API)")
	}

	doc := builder.Build()
	metadata := doc.GetMetadata()

	if metadata["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got %v", metadata["key1"])
	}
	if metadata["key2"] != 123 {
		t.Errorf("Expected key2=123, got %v", metadata["key2"])
	}
}

func TestBuilder_addContent(t *testing.T) {
	builder := New()

	content1 := &testContent{id: "test1", contentType: ContentTypeText}
	content2 := &testContent{id: "test2", contentType: ContentTypeTable}

	// Test fluent API with addContent
	result := builder.addContent(content1).addContent(content2)
	if result != builder {
		t.Error("addContent should return the same builder instance (fluent API)")
	}

	doc := builder.Build()
	contents := doc.GetContents()

	if len(contents) != 2 {
		t.Fatalf("Expected 2 contents, got %d", len(contents))
	}

	if contents[0].ID() != "test1" {
		t.Errorf("Expected first content ID='test1', got %s", contents[0].ID())
	}
	if contents[1].ID() != "test2" {
		t.Errorf("Expected second content ID='test2', got %s", contents[1].ID())
	}
}

func TestDocument_GetContents(t *testing.T) {
	builder := New()
	content := &testContent{id: "test1", contentType: ContentTypeText}
	builder.addContent(content)

	doc := builder.Build()

	// Get contents
	contents1 := doc.GetContents()
	contents2 := doc.GetContents()

	// Verify we get the same data
	if len(contents1) != 1 || len(contents2) != 1 {
		t.Error("GetContents returned inconsistent results")
	}

	// Verify we get copies (different slices)
	if &contents1[0] == &contents2[0] {
		t.Error("GetContents should return a copy of the slice")
	}

	// Modifying the returned slice should not affect the document
	contents1[0] = &testContent{id: "modified", contentType: ContentTypeRaw}
	contents3 := doc.GetContents()
	if contents3[0].ID() == "modified" {
		t.Error("Modifying returned contents affected the document")
	}
}

func TestDocument_GetMetadata(t *testing.T) {
	builder := New()
	builder.SetMetadata("key1", "value1")
	builder.SetMetadata("key2", 42)

	doc := builder.Build()

	// Get metadata
	metadata1 := doc.GetMetadata()
	metadata2 := doc.GetMetadata()

	// Verify we get the same data
	if len(metadata1) != 2 || len(metadata2) != 2 {
		t.Error("GetMetadata returned inconsistent results")
	}

	// Modifying the returned map should not affect the document
	metadata1["key1"] = "modified"
	metadata1["key3"] = "new"

	metadata3 := doc.GetMetadata()
	if metadata3["key1"] == "modified" {
		t.Error("Modifying returned metadata affected the document")
	}
	if _, exists := metadata3["key3"]; exists {
		t.Error("Adding to returned metadata affected the document")
	}
}

func TestBuilder_ThreadSafety(t *testing.T) {
	builder := New()
	var wg sync.WaitGroup

	// Number of concurrent operations
	const numGoroutines = 100

	wg.Add(numGoroutines * 2)

	// Concurrent metadata operations
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			builder.SetMetadata(string(rune('a'+index%26)), index)
		}(i)
	}

	// Concurrent content operations
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			content := &testContent{
				id:          generateID(),
				contentType: ContentType(index % 4),
			}
			builder.addContent(content)
		}(i)
	}

	wg.Wait()

	doc := builder.Build()
	contents := doc.GetContents()
	metadata := doc.GetMetadata()

	// Verify all operations succeeded
	if len(contents) != numGoroutines {
		t.Errorf("Expected %d contents, got %d", numGoroutines, len(contents))
	}

	// Metadata might have fewer entries due to key collisions (using modulo 26)
	if len(metadata) == 0 {
		t.Error("Expected some metadata entries, got none")
	}
}

func TestDocument_ThreadSafety(t *testing.T) {
	builder := New()

	// Add some initial content
	for i := 0; i < 10; i++ {
		builder.addContent(&testContent{
			id:          generateID(),
			contentType: ContentType(i % 4),
		})
	}
	builder.SetMetadata("initial", "value")

	doc := builder.Build()
	var wg sync.WaitGroup

	// Number of concurrent readers
	const numReaders = 100

	wg.Add(numReaders * 2)

	// Concurrent content reads
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			contents := doc.GetContents()
			if len(contents) != 10 {
				t.Errorf("Expected 10 contents, got %d", len(contents))
			}
		}()
	}

	// Concurrent metadata reads
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			metadata := doc.GetMetadata()
			if metadata["initial"] != "value" {
				t.Errorf("Expected initial='value', got %v", metadata["initial"])
			}
		}()
	}

	wg.Wait()
}

func TestBuilder_NilSafety(t *testing.T) {
	builder := New()
	doc := builder.Build()

	// These operations should not panic even though builder.doc is nil
	builder.SetMetadata("key", "value")
	builder.addContent(&testContent{id: "test", contentType: ContentTypeText})

	// The document should remain unchanged
	if len(doc.GetContents()) != 0 {
		t.Error("Document was modified after Build()")
	}
	if len(doc.GetMetadata()) != 0 {
		t.Error("Document metadata was modified after Build()")
	}
}
