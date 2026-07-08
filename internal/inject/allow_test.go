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
