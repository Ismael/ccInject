package inject

import (
	"strings"
	"testing"
)

func TestRenderEscapesAttributes(t *testing.T) {
	b := Block{Source: `cmd:jq '.tasks[] | select(.id == "3")' f.json`, Body: "x"}
	got := b.Render()
	want := `source="cmd:jq '.tasks[] | select(.id == &quot;3&quot;)' f.json"`
	if !strings.Contains(got, want) {
		t.Errorf("got %q, want it to contain %q", got, want)
	}
	if strings.Contains(got, `.id == "3"`) {
		t.Error("raw double quotes leaked into attribute")
	}
}

func TestRenderErrorMarker(t *testing.T) {
	got := Block{Source: "file:missing.md", Err: "open missing.md: no such file"}.Render()
	if !strings.HasSuffix(got, "/>") {
		t.Errorf("bodyless error must be self-closing: %q", got)
	}
	got = Block{Source: "cmd:git log", Err: "exit 128", Body: "fatal: not a git repository"}.Render()
	if strings.HasSuffix(got, "/>") || !strings.Contains(got, "fatal:") {
		t.Errorf("error with stderr body must be a normal block: %q", got)
	}
}

func TestRenderBody(t *testing.T) {
	got := Block{Source: "file:a.md", Body: "content\n"}.Render()
	want := "<injected-context source=\"file:a.md\">\ncontent\n</injected-context>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
