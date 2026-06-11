package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hrodrig/kzero/internal/log"
)

func runTimed(summary io.Writer, command string, colorMode string, format log.Format, fn func() error) error {
	start := time.Now()
	err := fn()
	if summary == nil {
		summary = os.Stderr
	}
	elapsed := time.Since(start)
	if format == log.FormatJSON {
		log.New(summary, format).CommandSummary(command, elapsed, err != nil)
		return err
	}
	writeCommandSummary(summary, command, elapsed, err, colorMode)
	return err
}

func writeCommandSummary(w io.Writer, command string, elapsed time.Duration, err error, colorMode string) {
	elapsed = elapsed.Round(time.Millisecond)
	elapsedStr := elapsed.String()
	failed := err != nil
	elapsedOut := colorizeElapsed(elapsedStr, failed, colorMode)
	if failed {
		_ = log.WriteLine(w, log.LevelError, fmt.Sprintf("kzero %s failed after %s", command, elapsedOut))
		return
	}
	_ = log.WriteLine(w, log.LevelInfo, fmt.Sprintf("kzero %s finished in %s", command, elapsedOut))
}
