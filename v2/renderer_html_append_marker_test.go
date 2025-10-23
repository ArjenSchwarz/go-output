package output

import (
	"context"
	"strings"
	"testing"
)

func TestHTMLRenderer_AppendMarkerInFullHTML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		useTemplate   bool
		wantMarker    bool
		markerContent string
	}{
		"Full HTML page includes marker": {
			useTemplate:   true,
			wantMarker:    true,
			markerContent: HTMLAppendMarker,
		},
		"HTML fragment does not include marker": {
			useTemplate:   false,
			wantMarker:    false,
			markerContent: HTMLAppendMarker,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Build a simple document
			doc := New().
				Text("Test content").
				Build()

			// Create renderer with or without template
			renderer := &htmlRenderer{
				baseRenderer: baseRenderer{},
				useTemplate:  tc.useTemplate,
				template:     nil, // Uses DefaultHTMLTemplate
			}

			ctx := context.Background()
			result, err := renderer.Render(ctx, doc)
			if err != nil {
				t.Fatalf("Failed to render HTML: %v", err)
			}

			resultStr := string(result)

			if tc.wantMarker {
				// Should contain the marker
				if !strings.Contains(resultStr, tc.markerContent) {
					t.Errorf("Expected marker %q in full HTML output, but it was not found", tc.markerContent)
				}

				// Marker should be before closing body tag
				bodyCloseIdx := strings.LastIndex(resultStr, "</body>")
				markerIdx := strings.LastIndex(resultStr, tc.markerContent)

				if bodyCloseIdx == -1 {
					t.Error("Expected </body> tag in full HTML output")
				}
				if markerIdx == -1 {
					t.Error("Expected marker in full HTML output")
				}

				if markerIdx != -1 && bodyCloseIdx != -1 && markerIdx >= bodyCloseIdx {
					t.Errorf("Marker should be before </body> tag: marker at %d, </body> at %d", markerIdx, bodyCloseIdx)
				}
			} else {
				// Should not contain the marker
				if strings.Contains(resultStr, tc.markerContent) {
					t.Errorf("Expected no marker in HTML fragment output, but found %q", tc.markerContent)
				}
			}
		})
	}
}

func TestHTMLRenderer_MarkerPlacement(t *testing.T) {
	t.Parallel()

	doc := New().
		Text("Content").
		Build()

	renderer := &htmlRenderer{
		baseRenderer: baseRenderer{},
		useTemplate:  true,
		template:     nil,
	}

	ctx := context.Background()
	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Find positions of important elements
	htmlCloseIdx := strings.LastIndex(resultStr, "</html>")
	bodyCloseIdx := strings.LastIndex(resultStr, "</body>")
	markerIdx := strings.LastIndex(resultStr, HTMLAppendMarker)

	if htmlCloseIdx == -1 {
		t.Fatal("Expected </html> tag in output")
	}
	if bodyCloseIdx == -1 {
		t.Fatal("Expected </body> tag in output")
	}
	if markerIdx == -1 {
		t.Fatal("Expected marker in output")
	}

	// Verify order: marker should be before </body>, and </body> before </html>
	if markerIdx >= bodyCloseIdx {
		t.Errorf("Marker should be before </body>: marker at %d, </body> at %d", markerIdx, bodyCloseIdx)
	}
	if bodyCloseIdx >= htmlCloseIdx {
		t.Errorf("</body> should be before </html>: </body> at %d, </html> at %d", bodyCloseIdx, htmlCloseIdx)
	}
}

func TestHTMLRenderer_MarkerFormat(t *testing.T) {
	t.Parallel()

	doc := New().
		Text("Test").
		Build()

	renderer := &htmlRenderer{
		baseRenderer: baseRenderer{},
		useTemplate:  true,
		template:     nil,
	}

	ctx := context.Background()
	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Verify exact marker format
	if !strings.Contains(resultStr, HTMLAppendMarker) {
		t.Errorf("Marker format mismatch: expected %q, not found in output", HTMLAppendMarker)
	}
}

func TestHTMLRenderer_MultipleMarkers(t *testing.T) {
	t.Parallel()

	doc := New().
		Text("Test").
		Build()

	renderer := &htmlRenderer{
		baseRenderer: baseRenderer{},
		useTemplate:  true,
		template:     nil,
	}

	ctx := context.Background()
	result, err := renderer.Render(ctx, doc)
	if err != nil {
		t.Fatalf("Failed to render HTML: %v", err)
	}

	resultStr := string(result)

	// Count occurrences of marker
	count := strings.Count(resultStr, HTMLAppendMarker)

	if count != 1 {
		t.Errorf("Expected exactly 1 marker in output, found %d", count)
	}
}
