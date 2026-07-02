//go:build windows

package cli

import (
	"context"
	"os"
	"os/signal"
)

func pipelineRunContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return signal.NotifyContext(parent, os.Interrupt)
}
