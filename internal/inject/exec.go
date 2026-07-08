package inject

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// RunCommand executes args directly (no shell) in dir. The child gets its own
// process group; on timeout the whole group is SIGKILLed, because
// exec.CommandContext alone only kills the direct child and helpers it
// spawned would survive.
func RunCommand(args []string, dir string, timeout time.Duration) (stdout, stderr []byte, err error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	var outB, errB bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outB, &errB
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-time.After(timeout):
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		<-done
		return outB.Bytes(), errB.Bytes(), fmt.Errorf("timeout after %s", timeout)
	case werr := <-done:
		if ee, ok := werr.(*exec.ExitError); ok {
			return outB.Bytes(), errB.Bytes(), fmt.Errorf("exit %d", ee.ExitCode())
		}
		return outB.Bytes(), errB.Bytes(), werr
	}
}

// ReadInjectFile reads path (relative paths resolve against cwd) and rejects
// binary content: a NUL byte in the first 8 KiB.
func ReadInjectFile(path, cwd string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	probe := data
	if len(probe) > 8192 {
		probe = probe[:8192]
	}
	if bytes.IndexByte(probe, 0) >= 0 {
		return nil, fmt.Errorf("binary file (NUL byte found)")
	}
	return data, nil
}
