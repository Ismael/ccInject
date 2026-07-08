package inject

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Print-only sed scripts: an optional numeric/$ or /regex/ address (possibly
// a range) followed by p or P. Anything fancier (s///, w, e, r …) is rejected.
var sedPrintOnly = regexp.MustCompile(
	`^(?:[0-9$]+(?:,[0-9$]+)?|/(?:[^/\\]|\\.)*/(?:,/(?:[^/\\]|\\.)*/)?)?[pP]$`)

// CheckAllowed enforces the read-only allowlist on a tokenized command.
// extra holds user additions from CCINJECT_ALLOW: first-token match only —
// read-only-ness of extras is the user's responsibility.
func CheckAllowed(args []string, extra []string) error {
	if len(args) == 0 {
		return fmt.Errorf("empty command")
	}
	// The executor runs args[0] verbatim, so the barrier must reject any
	// path-qualified command — otherwise a repo-committed ./tools/jq (basename
	// "jq") would pass the allowlist and run as arbitrary code. Bare names
	// resolve via PATH to the real tools.
	if filepath.Base(args[0]) != args[0] {
		return fmt.Errorf("command must be a bare name, not a path: %q", args[0])
	}
	name := args[0]
	switch name {
	case "git":
		if len(args) < 2 {
			return fmt.Errorf("git needs a subcommand (show, diff, or log)")
		}
		switch args[1] {
		case "show", "diff", "log":
		default:
			return fmt.Errorf("git %s not allowed (only show, diff, log)", args[1])
		}
		for _, a := range args[2:] {
			if a == "-o" || strings.HasPrefix(a, "--output") || a == "--ext-diff" {
				return fmt.Errorf("git flag %s not allowed (writes files or runs external commands)", a)
			}
		}
	case "sed":
		// Exactly: sed -n '<print-only script>' <file...>
		if len(args) < 3 || args[1] != "-n" {
			return fmt.Errorf("sed allowed only as: sed -n '<addresses>p' <file...>")
		}
		if !sedPrintOnly.MatchString(args[2]) {
			return fmt.Errorf("sed script %q is not print-only", args[2])
		}
	case "jq":
		// jq selects from a JSON file given as a positional arg. Deny flags
		// that read arbitrary files beyond that. jq bundles short options, so
		// -nf parses as -n -f — reject any short bundle containing f, not just
		// the exact -f token. (jq's env/$ENV can still disclose environment
		// variables — inherent to jq, accepted here.)
		for _, a := range args[1:] {
			if a == "--from-file" || a == "--rawfile" || a == "--slurpfile" {
				return fmt.Errorf("jq flag %s not allowed (reads arbitrary files)", a)
			}
			if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && strings.ContainsRune(a, 'f') {
				return fmt.Errorf("jq flag %s not allowed (reads arbitrary files)", a)
			}
		}
	case "cat", "wc", "head", "tail":
	default:
		for _, e := range extra {
			if name == e {
				return nil
			}
		}
		return fmt.Errorf("command %q not in allowlist (built-ins: git show/diff/log, cat, jq, sed -n, wc, head, tail; extend via CCINJECT_ALLOW)", name)
	}
	return nil
}
