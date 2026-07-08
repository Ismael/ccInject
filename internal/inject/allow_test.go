package inject

import "testing"

func TestAllowlist(t *testing.T) {
	ok := [][]string{
		{"git", "show", "--stat", "HEAD"},
		{"git", "diff", "main...HEAD"},
		{"git", "log", "--oneline", "-5"},
		{"jq", `.tasks[] | select(.id == "3")`, "plan.md.tasks.json"},
		{"sed", "-n", "10,20p", "f.txt"},
		{"sed", "-n", "/start/,/end/p", "f.txt"},
		{"cat", "f"}, {"wc", "-l", "f"}, {"head", "-n", "5", "f"}, {"tail", "-n", "3", "f"},
	}
	for _, args := range ok {
		if err := CheckAllowed(args, nil); err != nil {
			t.Errorf("%v: want allowed, got %v", args, err)
		}
	}
	bad := [][]string{
		{"git", "push"},
		{"git"},
		{"git", "diff", "--output=/tmp/x"},
		{"git", "diff", "-o", "/tmp/x"},
		{"git", "log", "--ext-diff"},
		{"sed", "s/a/b/", "f"},              // no -n
		{"sed", "-n", "s/a/b/w out", "f"},   // not print-only
		{"sed", "-i", "-n", "1p", "f"},      // -n not first flag
		{"rm", "-rf", "x"},
		{"curl", "https://x"},
		{"./git", "show"},                     // path-qualified
		{"/tmp/evil/git", "show"},             // absolute path
		{"jq", "-f", "filter.jq", "data.json"}, // jq file-read flag
	}
	for _, args := range bad {
		if err := CheckAllowed(args, nil); err == nil {
			t.Errorf("%v: want rejection, got nil", args)
		}
	}
}

func TestAllowlistExtra(t *testing.T) {
	if err := CheckAllowed([]string{"rg", "foo", "src/"}, []string{"rg", "ls"}); err != nil {
		t.Errorf("extra-allowed rg rejected: %v", err)
	}
	if err := CheckAllowed([]string{"rg", "foo"}, nil); err == nil {
		t.Error("rg without extra-allow: want rejection")
	}
}

func TestAllowlistRejectsPathAndJqFileFlags(t *testing.T) {
	bad := [][]string{
		{"./git", "show"},
		{"/tmp/evil/git", "show"},
		{"tools/jq", "."},
		{"jq", "-f", "filter.jq", "data.json"},
		{"jq", "--rawfile", "x", "/etc/passwd", "."},
		{"jq", "--slurpfile", "x", "secrets.json", "."},
		{"jq", "--from-file", "f.jq", "data.json"},
	}
	for _, args := range bad {
		if err := CheckAllowed(args, nil); err == nil {
			t.Errorf("%v: want rejection, got nil", args)
		}
	}
	// A path-qualified command must be rejected even if it appears in extra.
	if err := CheckAllowed([]string{"./rg", "x"}, []string{"rg"}); err == nil {
		t.Error("path-qualified extra command: want rejection, got nil")
	}
	// Bare positional filename for jq is still allowed.
	if err := CheckAllowed([]string{"jq", ".foo", "data.json"}, nil); err != nil {
		t.Errorf("jq with positional file: want allowed, got %v", err)
	}
	// Empty args must not panic.
	if err := CheckAllowed(nil, nil); err == nil {
		t.Error("empty args: want error, got nil")
	}
}
