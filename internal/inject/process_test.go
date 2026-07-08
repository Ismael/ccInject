package inject

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testCfg() Config {
	return Config{
		CmdTimeout:    2 * time.Second,
		Budget:        5 * time.Second,
		MaxInject:     32 * 1024,
		MaxDirectives: 16,
	}
}

func TestProcessHappyPath(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ctx.md"), []byte("the context body"), 0o644)
	out := Process("task\n@inject-file:ctx.md\n@inject-cmd:`wc -c ctx.md`\n", dir, testCfg())
	if out.Injected != 2 || out.Failed != 0 || out.Skipped != 0 {
		t.Fatalf("got %+v", out)
	}
	if !strings.Contains(out.Prompt, `<injected-context source="file:ctx.md">`) ||
		!strings.Contains(out.Prompt, "the context body") ||
		!strings.Contains(out.Prompt, `<injected-context source="cmd:wc -c ctx.md">`) {
		t.Errorf("prompt missing blocks:\n%s", out.Prompt)
	}
	if !strings.HasPrefix(out.Prompt, "task\n@inject-file:ctx.md\n") {
		t.Error("authored prompt must stay intact at the top")
	}
}

func TestProcessDuplicateDirective(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("body"), 0o644)
	out := Process("@inject-file:a.md\n@inject-file:a.md\n", dir, testCfg())
	if out.Injected != 1 || out.Skipped != 1 {
		t.Fatalf("want 1 injected 1 skipped, got %+v", out)
	}
}

func TestProcessMarkerSkipWithoutFetching(t *testing.T) {
	// File does not exist: if the marker check failed to short-circuit,
	// this would be a Failed, not a Skipped.
	prompt := "@inject-file:ghost.md\n<injected-context source=\"file:ghost.md\">\nold\n</injected-context>\n"
	out := Process(prompt, t.TempDir(), testCfg())
	if out.Skipped != 1 || out.Failed != 0 || out.Injected != 0 {
		t.Fatalf("want pure skip, got %+v", out)
	}
}

func TestProcessContentDedup(t *testing.T) {
	dir := t.TempDir()
	long := strings.Repeat("hand-pasted content line\n", 4) // ≥64 bytes
	os.WriteFile(filepath.Join(dir, "big.md"), []byte(long), 0o644)
	os.WriteFile(filepath.Join(dir, "tiny.md"), []byte("ok"), 0o644)

	out := Process("context:\n"+long+"\n@inject-file:big.md\n", dir, testCfg())
	if out.Skipped != 1 || out.Injected != 0 {
		t.Fatalf("long hand-pasted content: want skip, got %+v", out)
	}
	// "ok" appears in the prompt but is <64 bytes: must NOT be skipped.
	out = Process("this is ok\n@inject-file:tiny.md\n", dir, testCfg())
	if out.Injected != 1 || out.Skipped != 0 {
		t.Fatalf("short content must not false-skip, got %+v", out)
	}
}

func TestProcessLimitsAndFailures(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("aaa"), 0o644)

	cfg := testCfg()
	cfg.MaxDirectives = 1
	out := Process("@inject-file:a.md\n@inject-cmd:`wc -c a.md`\n", dir, cfg)
	if out.Injected != 1 || out.Failed != 1 {
		t.Fatalf("directive limit: got %+v", out)
	}
	if !strings.Contains(out.Prompt, `error="over directive limit`) {
		t.Error("want error marker for over-limit directive")
	}

	cfg = testCfg()
	cfg.Budget = -time.Second // already exhausted
	out = Process("@inject-file:a.md\n", dir, cfg)
	if out.Failed != 1 || !strings.Contains(out.Prompt, "budget exhausted") {
		t.Fatalf("budget: got %+v", out)
	}

	// Over MaxInject: rejected whole with a size marker, never truncated.
	os.WriteFile(filepath.Join(dir, "big.md"), []byte(strings.Repeat("x", 5000)), 0o644)
	cfg = testCfg()
	cfg.MaxInject = 100
	out = Process("@inject-file:big.md\n", dir, cfg)
	if out.Failed != 1 || out.Injected != 0 || !strings.Contains(out.Prompt, "can't add fully") {
		t.Fatalf("oversized file: got %+v", out)
	}
	if strings.Contains(out.Prompt, "xxxxx") {
		t.Error("oversized content must not be partially injected")
	}
}

func TestProcessMessage(t *testing.T) {
	o := Outcome{Injected: 2, InjectedBytes: 12700, Skipped: 1, Failed: 1,
		FailNotes: []string{"timeout after 2s: git log --all"}}
	got := o.Message()
	for _, want := range []string{"ccinject:", "2 injected", "12.4 KiB", "1 skipped (already present)", "1 failed", "timeout"} {
		if !strings.Contains(got, want) {
			t.Errorf("message %q missing %q", got, want)
		}
	}
}
