package mermaid

// MermaidChart defines the interface for all Mermaid chart types
type MermaidChart interface {
	RenderString() string
}
