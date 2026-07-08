package tests

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"ccinject/internal/inject"
)

// End-to-end: a scratch git repo, a real prompt with all three directive
// outcomes (inject, skip, fail), through inject.Run exactly as main calls it.
func TestE2EGitRepo(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	run("git", "init", "-q")
	if err := os.WriteFile(filepath.Join(dir, "kit.md"), []byte(strings.Repeat("ui verify kit instructions\n", 8)), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "kit.md")
	run("git", "commit", "-q", "-m", "add kit")

	prompt := "Verify the UI.\n" +
		"@inject-file:kit.md\n" +
		"@inject-cmd:`git log --oneline -1 | cat`\n" + // pipe now works: real shell
		"@inject-cmd:`git push no-such-remote main`\n" // must fail: no such remote
	stdin, _ := json.Marshal(map[string]any{
		"tool_name": "Agent", "cwd": dir,
		"tool_input": map[string]any{"prompt": prompt, "subagent_type": "general-purpose"},
	})
	out := inject.Run(stdin, inject.ConfigFromEnv())
	if out == nil {
		t.Fatal("want output")
	}
	s := string(out)
	for _, want := range []string{"ui verify kit instructions", "add kit", "2 injected", "1 failed"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
}
