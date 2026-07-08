package inject

import (
	"fmt"
	"strings"
	"time"
)

// Content-substring idempotence only applies above this floor: the empty
// string is a substring of everything, and trivially common output would
// false-skip.
const minContentDedupLen = 64

type Outcome struct {
	Prompt        string // augmented prompt; equals the input when nothing was appended
	Injected      int
	Skipped       int
	Failed        int
	InjectedBytes int
	FailNotes     []string // short per-failure reasons for the systemMessage
}

func (o Outcome) Message() string {
	var parts []string
	if o.Injected > 0 {
		parts = append(parts, fmt.Sprintf("%d injected (%.1f KiB)", o.Injected, float64(o.InjectedBytes)/1024))
	}
	if o.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped (already present)", o.Skipped))
	}
	if o.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed (%s)", o.Failed, strings.Join(o.FailNotes, "; ")))
	}
	return "ccinject: " + strings.Join(parts, ", ")
}

// Process expands every directive in prompt, applying idempotence checks in
// spec order: per-run seen-set, pre-execution marker, post-execution content.
func Process(prompt, cwd string, cfg Config) Outcome {
	out := Outcome{Prompt: prompt}
	directives := ParseDirectives(prompt)
	if len(directives) == 0 {
		return out
	}
	deadline := time.Now().Add(cfg.Budget)
	seen := make(map[string]bool)
	var blocks []string
	total := 0

	fail := func(d Directive, why, stderrExcerpt string) {
		out.Failed++
		out.FailNotes = append(out.FailNotes, why+": "+clip(d.Arg, 60))
		blocks = append(blocks, Block{Source: d.Source, Err: why, Body: stderrExcerpt}.Render())
	}

	attempted := 0
	for _, d := range directives {
		if seen[d.Source] {
			out.Skipped++
			continue
		}
		seen[d.Source] = true
		if strings.Contains(prompt, `source="`+escapeAttr(d.Source)+`"`) {
			out.Skipped++
			continue
		}
		attempted++
		if attempted > cfg.MaxDirectives {
			fail(d, fmt.Sprintf("over directive limit (%d)", cfg.MaxDirectives), "")
			continue
		}
		if !time.Now().Before(deadline) {
			fail(d, "budget exhausted", "")
			continue
		}
		body, errText, stderrExcerpt := fetch(d, cwd, cfg, deadline)
		if errText != "" {
			fail(d, errText, stderrExcerpt)
			continue
		}
		if trimmed := strings.TrimSpace(body); len(trimmed) >= minContentDedupLen && strings.Contains(prompt, trimmed) {
			out.Skipped++
			continue
		}
		// Oversized content is rejected upstream in fetch (with an "is X MB"
		// marker), never truncated — so whatever reaches here is injected whole.
		total += len(body)
		blocks = append(blocks, Block{Source: d.Source, Body: body}.Render())
		out.Injected++
	}
	out.InjectedBytes = total
	if len(blocks) > 0 {
		out.Prompt = prompt + "\n\n" + strings.Join(blocks, "\n\n")
	}
	return out
}

func fetch(d Directive, cwd string, cfg Config, deadline time.Time) (body, errText, stderrExcerpt string) {
	switch d.Kind {
	case "file":
		timeout := cfg.CmdTimeout
		if remain := time.Until(deadline); remain < timeout {
			timeout = remain
		}
		data, err := ReadInjectFile(d.Arg, cwd, cfg.MaxInject, timeout)
		if err != nil {
			return "", err.Error(), ""
		}
		return string(data), "", ""
	default: // "cmd"
		timeout := cfg.CmdTimeout
		if remain := time.Until(deadline); remain < timeout {
			timeout = remain
		}
		stdout, stderr, err := RunCommand(d.Arg, cwd, timeout, cfg.MaxInject)
		if err != nil {
			return "", err.Error(), clip(string(stderr), 1024)
		}
		return string(stdout), "", ""
	}
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
