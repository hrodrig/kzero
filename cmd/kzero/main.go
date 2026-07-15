package main

import (
	"fmt"
	"os"

	"github.com/hrodrig/kzero/internal/cli"
	"github.com/hrodrig/kzero/internal/exitcode"
)

func main() {
	os.Exit(runMain())
}

// runMain runs the CLI and returns a process exit code (see exitcode.Of / #42).
func runMain() int {
	if err := cli.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		return exitcode.Of(err)
	}
	return exitcode.Success
}
