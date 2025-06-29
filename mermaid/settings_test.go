package mermaid

import "testing"

func resetScript() { scriptset = false }

func TestMarkdownHeaderFooter(t *testing.T) {
	s := &Settings{AddMarkdown: true}
	if s.MarkdownHeader() != "```mermaid\n" {
		t.Errorf("header mismatch")
	}
	if s.MarkdownFooter() != "\n```" {
		t.Errorf("footer mismatch")
	}
	s.AddMarkdown = false
	if s.MarkdownHeader() != "" || s.MarkdownFooter() != "" {
		t.Errorf("expected empty")
	}
}

func TestHtmlHeaderFooter(t *testing.T) {
	resetScript()
	s := &Settings{AddHTML: true}
	h1 := s.HtmlHeader()
	if h1 == "" || scriptset == false {
		t.Fatalf("expected script header")
	}
	h2 := s.HtmlHeader()
	if h2 != "<div class='mermaid'>\n" {
		t.Errorf("expected div only on second call")
	}
	if s.HtmlFooter() != "</div>\n" {
		t.Errorf("footer mismatch")
	}
}

func TestHeaderFooterDelegation(t *testing.T) {
	resetScript()
	s := &Settings{AddMarkdown: true}
	if s.Header() != "```mermaid\n" || s.Footer() != "\n```" {
		t.Errorf("markdown delegation")
	}
	s.AddMarkdown = false
	s.AddHTML = true
	if s.Header() == "" || s.Footer() != "</div>\n" {
		t.Errorf("html delegation")
	}
}
