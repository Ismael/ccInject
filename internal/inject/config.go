package inject

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	CmdTimeout time.Duration
	Budget     time.Duration // total wall budget for all directives
	// MaxInject caps a single injection. Content over this size is NOT
	// truncated — it is rejected whole with an "is X MB, can't add fully"
	// marker, so the subagent fetches it itself instead of reading a silently
	// clipped fragment. It doubles as the memory guard on reads/exec: a
	// firehose (cat /dev/zero, a multi-GB file) is bounded to ~this many bytes.
	MaxInject     int
	MaxDirectives int
	// NoCmd rejects @inject-cmd directives (with an error marker) so only
	// @inject-file works. @inject-cmd runs arbitrary shell in prompts, and
	// some environments want that surface closed.
	NoCmd bool
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}

func ConfigFromEnv() Config {
	return Config{
		CmdTimeout: time.Duration(envInt("CCINJECT_CMD_TIMEOUT_MS", 2000)) * time.Millisecond,
		Budget:     time.Duration(envInt("CCINJECT_BUDGET_MS", 5000)) * time.Millisecond,
		// ~100k tokens at a rough 4 bytes/token ≈ 0.4 MB — the point past which
		// a single injection is too big to be worth inlining.
		MaxInject:     envInt("CCINJECT_MAX_INJECT_BYTES", 400*1024),
		MaxDirectives: envInt("CCINJECT_MAX_DIRECTIVES", 16),
		NoCmd:         os.Getenv("CCINJECT_DISABLE_CMD") == "1",
	}
}
