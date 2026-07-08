package main

import (
	"io"
	"os"

	"ccinject/internal/inject"
)

// Fail-open contract: exit 0 no matter what. A hook crash must never block
// an Agent dispatch; printing nothing leaves the dispatch untouched.
func main() {
	defer func() { _ = recover() }()
	if os.Getenv("CCINJECT_DISABLE") == "1" {
		return
	}
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		return
	}
	if out := inject.Run(stdin, inject.ConfigFromEnv()); out != nil {
		os.Stdout.Write(out)
		os.Stdout.Write([]byte("\n"))
	}
}
