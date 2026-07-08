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
// full pipe nor grow our memory past cap — the timeout bounds wall-time. total
// counts every byte seen (including discarded ones), so a caller can tell that
// output overflowed cap and how big it actually was.
type capWriter struct {
	buf   bytes.Buffer
	cap   int
	total int64
}

func (w *capWriter) Write(p []byte) (int, error) {
	n := len(p) // report the FULL length: reslicing p below would otherwise
	// short-write, making io.Copy error out and close the pipe (SIGPIPE to the
	// child) instead of letting the timeout bound wall-time.
	w.total += int64(n)
	if room := w.cap - w.buf.Len(); room > 0 {
		if len(p) > room {
			p = p[:room]
		}
		w.buf.Write(p)
	}
	return n, nil // discard the rest; never short-write
}

// humanMB renders a byte count as "X.Y MB" for the oversized-injection marker.
func humanMB(n int64) string {
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

// RunCommand runs command through `sh -c` in dir — full shell, so pipes,
// redirects, and substitutions all work; vetting the command is the caller's
// (the agent's) job, not ours. The child gets its own process group; on timeout
// the whole group is SIGKILLed, because exec.CommandContext alone only kills the
// direct child and helpers it spawned would survive. stdout is capped at
// maxBytes and stderr at a small fixed size so a runaway producer degrades to a
// bounded, timed-out read instead of a fatal OOM; output past maxBytes is
// rejected whole (only a ~1 KiB stderr excerpt is ever surfaced).
func RunCommand(command, dir string, timeout time.Duration, maxBytes int) (stdout, stderr []byte, err error) {
	cmd := exec.Command("sh", "-c", command)
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
		if werr == nil && outB.total > int64(maxBytes) {
			return nil, errB.buf.Bytes(), fmt.Errorf("output is %s, can't add fully", humanMB(outB.total))
		}
		return outB.buf.Bytes(), errB.buf.Bytes(), werr
	}
}

// ReadInjectFile reads path (relative paths resolve against cwd), rejecting
// binary content (a NUL byte in the first 8 KiB). The read is bounded in both
// size and time: at most maxBytes+1 bytes are read (the +1 lets us detect
// overflow), so /dev/zero returns promptly instead of looping forever, and a
// blocking special file (e.g. a fifo with no writer) can't hang the hook past
// timeout. A file larger than maxBytes is rejected whole with an "is X MB,
// can't add fully" error rather than truncated.
func ReadInjectFile(path, cwd string, maxBytes int, timeout time.Duration) ([]byte, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	type result struct {
		data []byte
		size int64 // stat size when known (0 for streams); for the size marker
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
			done <- result{nil, 0, err}
			return
		}
		defer f.Close()
		var size int64
		if fi, e := f.Stat(); e == nil {
			size = fi.Size()
		}
		data, err := io.ReadAll(io.LimitReader(f, int64(maxBytes)+1))
		done <- result{data, size, err}
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
		if len(r.data) > maxBytes {
			// Prefer the stat size for the message; a stream reports 0, so fall
			// back to the (over-cap) bytes actually read.
			size := r.size
			if size < int64(len(r.data)) {
				size = int64(len(r.data))
			}
			return nil, fmt.Errorf("file is %s, can't add fully", humanMB(size))
		}
		return r.data, nil
	}
}
