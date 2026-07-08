package inject

import "strings"

var attrEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func escapeAttr(s string) string { return attrEscaper.Replace(s) }

type Block struct {
	Source string // normalized directive source (unescaped; Render escapes)
	Body   string // verbatim content; for failed commands, the stderr excerpt
	Err    string // failure reason; empty on success
}

func (b Block) Render() string {
	var sb strings.Builder
	sb.WriteString(`<injected-context source="` + escapeAttr(b.Source) + `"`)
	if b.Err != "" {
		sb.WriteString(` error="` + escapeAttr(b.Err) + `"`)
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
