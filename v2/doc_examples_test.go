package output

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// docGoBlock is a fenced ```go code block extracted from a Markdown file.
type docGoBlock struct {
	startLine int    // 1-indexed line of the opening fence
	code      string // block contents (without the fences)
}

// TestDocumentationExamplesCompile compiles the complete Go programs embedded in
// the docs/ Markdown files so that documentation drift — examples that reference
// APIs which no longer exist — fails the build.
//
// Only fenced ```go blocks that are complete programs (they contain both
// "package main" and a "func main(") are compiled. Fragments (method signatures,
// partial chains, snippets without a main) are illustrative and skipped.
//
// Each program is written to a temporary directory nested inside this module so
// that imports of the local github.com/ArjenSchwarz/go-output/v2 package resolve
// to the working tree, then built with "go build".
func TestDocumentationExamplesCompile(t *testing.T) {
	docFiles, err := filepath.Glob(filepath.Join("docs", "*.md"))
	if err != nil {
		t.Fatalf("glob docs: %v", err)
	}
	if len(docFiles) == 0 {
		t.Fatal("no documentation files found under docs/")
	}

	// Temp build dirs must live inside this module so local imports resolve to
	// the working tree rather than a downloaded version.
	tmpRoot, err := os.MkdirTemp(".", ".doccheck")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpRoot) })

	checked := 0
	for _, file := range docFiles {
		for i, block := range extractGoBlocks(t, file) {
			if !strings.Contains(block.code, "package main") || !strings.Contains(block.code, "func main(") {
				continue
			}
			checked++
			name := fmt.Sprintf("%s:%d", file, block.startLine)
			t.Run(name, func(t *testing.T) {
				dir := filepath.Join(tmpRoot, fmt.Sprintf("%s_%d", filepath.Base(file), i))
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatalf("mkdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(block.code+"\n"), 0o644); err != nil {
					t.Fatalf("write example: %v", err)
				}
				cmd := exec.Command("go", "build", "-o", os.DevNull, ".")
				cmd.Dir = dir
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Errorf("example does not compile:\n%s", out)
				}
			})
		}
	}
	if checked == 0 {
		t.Fatal("no complete documentation examples were found to compile")
	}
	t.Logf("compiled %d complete documentation example(s)", checked)
}

// extractGoBlocks returns every fenced ```go block in the Markdown file.
func extractGoBlocks(t *testing.T, file string) []docGoBlock {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}

	var blocks []docGoBlock
	var buf []string
	inBlock := false
	startLine := 0
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if trimmed == "```go" || trimmed == "```golang" {
				inBlock = true
				buf = nil
				startLine = i + 1
			}
			continue
		}
		if trimmed == "```" {
			blocks = append(blocks, docGoBlock{startLine: startLine, code: strings.Join(buf, "\n")})
			inBlock = false
			continue
		}
		buf = append(buf, line)
	}
	return blocks
}
