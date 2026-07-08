package inject

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// capWriter stores at most cap bytes but reports every write as fully
// consumed, so a firehose like `cat /dev/zero` can't block the child on a
// full pipe nor grow our memory past cap — the timeout bounds wall-time.
type capWriter struct {
	buf bytes.Buffer
	cap int
}

func (w *capWriter) Write(p []byte) (int, error) {
	n := len(p) // report the FULL length: reslicing p below would otherwise
	// short-write, making io.Copy error out and close the pipe (SIGPIPE to the
	// child) instead of letting the timeout bound wall-time.
	if room := w.cap - w.buf.Len(); room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		w.buf.Write(p)
	}
	return n, nil // discard the rest; never short-write
}

// RunCommand executes args directly (no shell) in dir. The child gets its own
// process group; on timeout the whole group is SIGKILLed, because
// exec.CommandContext alone only kills the direct child and helpers it
// spawned would survive. stdout is capped at maxBytes and stderr at a small
// fixed size so a runaway producer degrades to a bounded read instead of a
// fatal OOM (truncation downstream trims to MaxInject; only a ~1 KiB stderr
// excerpt is ever surfaced).
func RunCommand(args []string, dir string, timeout time.Duration, maxBytes int) (stdout, stderr []byte, err error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	outB := &capWriter{cap: maxBytes}
	errB := &capWriter{cap: 4096}
	cmd.Stdout, cmd.Stderr = outB, errB
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return outB.buf.Bytes(), errB.buf.Bytes(), fmt.Errorf("timeout after %s", timeout)
	case werr := <-done:
		if ee, ok := werr.(*exec.ExitError); ok {
			return outB.buf.Bytes(), errB.buf.Bytes(), fmt.Errorf("exit %d", ee.ExitCode())
		}
		return outB.buf.Bytes(), errB.buf.Bytes(), werr
	}
}

// ReadInjectFile reads path (relative paths resolve against cwd), rejecting
// binary content (a NUL byte in the first 8 KiB). The read is bounded in both
// size and time: at most maxBytes+1 bytes are read (the +1 lets downstream
// truncation detect overflow — an over-cap read is NOT an error), so
// /dev/zero returns promptly instead of looping forever, and a blocking
// special file (e.g. a fifo with no writer) can't hang the hook past timeout.
func ReadInjectFile(path, cwd string, maxBytes int, timeout time.Duration) ([]byte, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	type result struct {
		data []byte
		err  error
	}
	done := make(chan result, 1)
	// A background read of a hung special file (e.g. a writer-less fifo) may
	// linger after we time out; the buffered channel prevents the goroutine
	// from blocking on send, and the leaked read is bounded to one fd — an
	// acceptable trade for keeping the hook responsive.
	go func() {
		f, err := os.Open(path)
		if err != nil {
			done <- result{nil, err}
			return
		}
		defer f.Close()
		data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
		done <- result{data, err}
	}()
	select {
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout after %s", timeout)
	case r := <-done:
		if r.err != nil {
			return nil, r.err
		}
		probe := r.data
		if len(probe) > 8192 {
			probe = probe[:8192]
		}
		if bytes.IndexByte(probe, 0) >= 0 {
			return nil, fmt.Errorf("binary file (NUL byte found)")
		}
		return r.data, nil
	}
}
