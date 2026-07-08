package inject

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecRunCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := RunCommand([]string{"wc", "-l", "f.txt"}, dir, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stdout), "3") {
		t.Errorf("wc output %q lacks line count 3", stdout)
	}
}

func TestExecTimeoutKillsGroup(t *testing.T) {
	start := time.Now()
	// bash spawns a grandchild; the process-group kill must take both down
	// without waiting for the grandchild's 30s sleep.
	_, _, err := RunCommand([]string{"bash", "-c", "sleep 30 & wait"}, ".", 200*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("want timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("kill was not prompt: took %s", elapsed)
	}
}

func TestExecNonZeroExit(t *testing.T) {
	_, stderr, err := RunCommand([]string{"ls", "/nonexistent-ccinject-test"}, ".", 2*time.Second)
	if err == nil || !strings.HasPrefix(err.Error(), "exit ") {
		t.Fatalf("want 'exit N' error, got %v", err)
	}
	if len(stderr) == 0 {
		t.Error("want stderr captured on failure")
	}
}

func TestExecReadInjectFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "ok.md"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "bin.dat"), []byte{1, 2, 0, 4}, 0o644)

	data, err := ReadInjectFile("ok.md", dir) // relative resolves against cwd
	if err != nil || string(data) != "hello" {
		t.Fatalf("got %q, %v", data, err)
	}
	if _, err := ReadInjectFile("bin.dat", dir); err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("want binary-file rejection, got %v", err)
	}
	if _, err := ReadInjectFile("missing.md", dir); err == nil {
		t.Fatal("want error for missing file")
	}
}
