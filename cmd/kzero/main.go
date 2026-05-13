package main

import (
	"fmt"
	"os"

	"github.com/hrodrig/kzero/internal/cli"
)

func main() {
	os.Exit(runMain())
}

// runMain runs the CLI and returns an exit code (0 success, 1 error).
func runMain() int {
	if err := cli.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
