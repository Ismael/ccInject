package inject

import (
	"fmt"
	"strings"
)

var attrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func escapeAttr(s string) string { return attrEscaper.Replace(s) }

type Block struct {
	Source    string // normalized directive source (unescaped; Render escapes)
	Body      string // verbatim content; for failed commands, the stderr excerpt
	Err       string // failure reason; empty on success
	Truncated bool
}

func (b Block) Render() string {
	var sb strings.Builder
	sb.WriteString(`<injected-context source="` + escapeAttr(b.Source) + `"`)
	if b.Err != "" {
		sb.WriteString(` error="` + escapeAttr(b.Err) + `"`)
	}
	if b.Truncated {
		sb.WriteString(` truncated="true"`)
	}
	if b.Body == "" && b.Err != "" {
		sb.WriteString("/>")
		return sb.String()
	}
	sb.WriteString(">\n")
	sb.WriteString(strings.TrimRight(b.Body, "\n"))
	sb.WriteString("\n</injected-context>")
	return sb.String()
}

func truncate(body string, maxBytes int) (string, bool) {
	if len(body) <= maxBytes {
		return body, false
	}
	return body[:maxBytes] + "\n[ccinject: truncated at " + sizeLabel(maxBytes) + "]", true
}

// sizeLabel avoids "0 KiB" for small or unaligned caps (env overrides, tests).
func sizeLabel(n int) string {
	if n >= 1024 && n%1024 == 0 {
		return fmt.Sprintf("%d KiB", n/1024)
	}
	return fmt.Sprintf("%d bytes", n)
}
