package output

import (
	"testing"
)

// TestContentType_String tests the string representation of ContentType
func TestContentType_String(t *testing.T) {
	tests := []struct {
		name     string
		ct       ContentType
		expected string
	}{
		{"Table", ContentTypeTable, FormatTable},
		{"Text", ContentTypeText, FormatText},
		{"Raw", ContentTypeRaw, "raw"},
		{"Section", ContentTypeSection, "section"},
		{"Unknown", ContentType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ct.String(); got != tt.expected {
				t.Errorf("ContentType.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestGenerateID tests the ID generation function
func TestGenerateID(t *testing.T) {
	// Test that IDs are generated
	id1 := GenerateID()
	if id1 == "" {
		t.Error("GenerateID() returned empty string")
	}

	// Test that IDs are unique
	id2 := GenerateID()
	if id1 == id2 {
		t.Error("GenerateID() returned duplicate IDs")
	}

	// Test that IDs have the expected prefix
	if len(id1) < 8 || id1[:8] != "content-" {
		t.Errorf("GenerateID() = %v, expected to start with 'content-'", id1)
	}
}

// TestGenerateID_Uniqueness tests that multiple ID generations are unique
func TestGenerateID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	const numIDs = 1000

	for i := 0; i < numIDs; i++ {
		id := GenerateID()
		if ids[id] {
			t.Errorf("GenerateID() generated duplicate ID: %v", id)
		}
		ids[id] = true
	}

	if len(ids) != numIDs {
		t.Errorf("Expected %d unique IDs, got %d", numIDs, len(ids))
	}
}

// mockContent is a test implementation of the Content interface
type mockContent struct {
	id    string
	ctype ContentType
}

func (m *mockContent) Type() ContentType {
	return m.ctype
}

func (m *mockContent) ID() string {
	return m.id
}

func (m *mockContent) AppendText(b []byte) ([]byte, error) {
	return append(b, "mock text content"...), nil
}

func (m *mockContent) AppendBinary(b []byte) ([]byte, error) {
	return append(b, []byte("mock binary content")...), nil
}

// TestContent_Interface tests the Content interface implementation
func TestContent_Interface(t *testing.T) {
	mock := &mockContent{
		id:    "test-id",
		ctype: ContentTypeText,
	}

	// Test Type()
	if got := mock.Type(); got != ContentTypeText {
		t.Errorf("Content.Type() = %v, want %v", got, ContentTypeText)
	}

	// Test ID()
	if got := mock.ID(); got != "test-id" {
		t.Errorf("Content.ID() = %v, want %v", got, "test-id")
	}

	// Test AppendText()
	result, err := mock.AppendText([]byte("prefix-"))
	if err != nil {
		t.Errorf("Content.AppendText() error = %v", err)
	}
	expected := "prefix-mock text content"
	if string(result) != expected {
		t.Errorf("Content.AppendText() = %v, want %v", string(result), expected)
	}

	// Test AppendBinary()
	result, err = mock.AppendBinary([]byte("prefix-"))
	if err != nil {
		t.Errorf("Content.AppendBinary() error = %v", err)
	}
	expected = "prefix-mock binary content"
	if string(result) != expected {
		t.Errorf("Content.AppendBinary() = %v, want %v", string(result), expected)
	}
}

// TestContentType_EnumValues tests all ContentType enum values
func TestContentType_EnumValues(t *testing.T) {
	expectedTypes := []ContentType{
		ContentTypeTable,
		ContentTypeText,
		ContentTypeRaw,
		ContentTypeSection,
	}

	// Verify enum values are sequential starting from 0
	for i, ct := range expectedTypes {
		if int(ct) != i {
			t.Errorf("ContentType enum value mismatch: %v should be %d", ct, i)
		}
	}

	// Verify they all have valid string representations
	for _, ct := range expectedTypes {
		str := ct.String()
		if str == "unknown" {
			t.Errorf("ContentType %v should not return 'unknown'", ct)
		}
	}
}
