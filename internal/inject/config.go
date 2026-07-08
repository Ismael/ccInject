package inject

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	CmdTimeout    time.Duration
	Budget        time.Duration // total wall budget for all directives
	MaxInject     int           // bytes per injection
	MaxTotal      int           // bytes across all injections
	MaxDirectives int
	ExtraAllow    []string // CCINJECT_ALLOW additions
}

func envInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil && v > 0 {
		return v
	}
	return def
}

func ConfigFromEnv() Config {
	var extra []string
	for _, name := range strings.Split(os.Getenv("CCINJECT_ALLOW"), ",") {
		if name = strings.TrimSpace(name); name != "" {
			extra = append(extra, name)
		}
	}
	return Config{
		CmdTimeout:    time.Duration(envInt("CCINJECT_CMD_TIMEOUT_MS", 2000)) * time.Millisecond,
		Budget:        time.Duration(envInt("CCINJECT_BUDGET_MS", 5000)) * time.Millisecond,
		MaxInject:     envInt("CCINJECT_MAX_INJECT_BYTES", 32*1024),
		MaxTotal:      envInt("CCINJECT_MAX_TOTAL_BYTES", 128*1024),
		MaxDirectives: envInt("CCINJECT_MAX_DIRECTIVES", 16),
		ExtraAllow:    extra,
	}
}
