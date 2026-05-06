package engine

import (
	"context"
	"os/exec"
)

func defaultExecCombined(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, argv0, args...)
	cmd.Dir = dir
	cmd.Env = env
	return cmd.CombinedOutput()
}
