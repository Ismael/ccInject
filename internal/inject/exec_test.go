package inject

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestExecRunCommand(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := RunCommand("wc -l f.txt", dir, 2*time.Second, 32*1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stdout), "3") {
		t.Errorf("wc output %q lacks line count 3", stdout)
	}
}

func TestExecTimeoutKillsGroup(t *testing.T) {
	start := time.Now()
	// sh backgrounds a child sleep; the process-group kill must take both down
	// without waiting for the child's 30s sleep.
	_, _, err := RunCommand("sleep 30 & wait", ".", 200*time.Millisecond, 32*1024)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("want timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("kill was not prompt: took %s", elapsed)
	}
}

func TestExecNonZeroExit(t *testing.T) {
	_, stderr, err := RunCommand("ls /nonexistent-ccinject-test", ".", 2*time.Second, 32*1024)
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

	data, err := ReadInjectFile("ok.md", dir, 32*1024, 2*time.Second) // relative resolves against cwd
	if err != nil || string(data) != "hello" {
		t.Fatalf("got %q, %v", data, err)
	}
	if _, err := ReadInjectFile("bin.dat", dir, 32*1024, 2*time.Second); err == nil || !strings.Contains(err.Error(), "binary") {
		t.Fatalf("want binary-file rejection, got %v", err)
	}
	if _, err := ReadInjectFile("missing.md", dir, 32*1024, 2*time.Second); err == nil {
		t.Fatal("want error for missing file")
	}
}

// TestExecRunCommandBounded proves a firehose (cat /dev/zero) degrades to a
// bounded, timed-out read instead of driving the process to a fatal OOM.
func TestExecRunCommandBounded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/dev/zero is unix-only")
	}
	start := time.Now()
	stdout, _, err := RunCommand("cat /dev/zero", ".", 300*time.Millisecond, 4096)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("want timeout error, got %v", err)
	}
	if len(stdout) > 4096 {
		t.Fatalf("stdout not capped: got %d bytes", len(stdout))
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("not prompt: took %s", elapsed)
	}
}

// TestExecReadInjectFileBounded proves /dev/zero returns bounded bytes or a
// timeout error promptly instead of hanging or OOMing.
func TestExecReadInjectFileBounded(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/dev/zero is unix-only")
	}
	start := time.Now()
	data, err := ReadInjectFile("/dev/zero", "", 4096, 300*time.Millisecond)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("not prompt: took %s", elapsed)
	}
	// Either the size cap kicks in (capped bytes) or the timeout fires; both
	// are bounded outcomes. /dev/zero holds only NUL bytes, so the binary
	// check may also fire — any non-hang, non-OOM result is acceptable.
	if err == nil && len(data) > 4097 {
		t.Fatalf("read not capped: got %d bytes", len(data))
	}
}
