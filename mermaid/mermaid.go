package mermaid

// MermaidChart interface for mermaid chart implementations
type MermaidChart interface {
	RenderString() string
}
