// Package inject implements the ccinject PreToolUse hook: it expands
// @inject-file / @inject-cmd directives found in Agent-dispatch prompts into
// <injected-context> blocks appended to the prompt.
package inject

import (
	"regexp"
	"strings"
)

type Directive struct {
	Kind   string // "file" or "cmd"
	Arg    string // trimmed path, or command after backtick stripping
	Source string // normalized identity: "file:<path>" or "cmd:<command>"
}

// Line-anchored so prose that merely mentions a directive never triggers.
var directiveRe = regexp.MustCompile(`^[ \t]*@inject-(file|cmd):(.+)$`)

func ParseDirectives(prompt string) []Directive {
	var out []Directive
	for _, line := range strings.Split(prompt, "\n") {
		m := directiveRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		arg := strings.TrimSpace(m[2])
		if m[1] == "cmd" {
			arg = stripBackticks(arg)
		}
		if arg == "" {
			continue
		}
		out = append(out, Directive{Kind: m[1], Arg: arg, Source: m[1] + ":" + arg})
	}
	return out
}

func stripBackticks(s string) string {
	if len(s) >= 2 && strings.HasPrefix(s, "`") && strings.HasSuffix(s, "`") {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}
