package engine

import (
	"fmt"

	"github.com/hrodrig/kzero/internal/config"
)

func (r *LiveRunner) logLive(_ *config.Config, format string, args ...interface{}) {
	if r.Out == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(r.Out, "[live] %s\n", msg)
}
