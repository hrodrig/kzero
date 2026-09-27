package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hrodrig/kzero/internal/cluster"
	"github.com/hrodrig/kzero/internal/config"
	"github.com/hrodrig/kzero/internal/log"
	"github.com/spf13/cobra"
)

// runLogClock is overridden in tests for deterministic filenames.
var runLogClock = time.Now

// resolveRunLogPath returns the opt-in log file path, or "" when logging to disk is unset.
// Precedence: --log-file → --log-dir → run.log_file → run.log_dir.
func resolveRunLogPath(cfg *config.Config, command string) (string, error) {
	file := strings.TrimSpace(logFileFlag)
	if file == "" && cfg != nil {
		file = strings.TrimSpace(cfg.Run.LogFile)
	}
	if file != "" {
		return file, nil
	}

	dir := strings.TrimSpace(logDirFlag)
	if dir == "" && cfg != nil {
		dir = strings.TrimSpace(cfg.Run.LogDir)
	}
	if dir == "" {
		return "", nil
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("run log_dir: mkdir %s: %w", dir, err)
	}

	cmdSlug := cluster.SanitizeForFilename(command)
	slug := "unknown"
	if cfg != nil {
		if tgt, err := cluster.ResolveFromConfig(cfg); err == nil {
			slug = cluster.SanitizeForFilename(tgt.ClusterName)
		}
	}
	ts := runLogClock().Format("20060102-150405")
	name := fmt.Sprintf("kzero-%s-%s-%s.log", cmdSlug, slug, ts)
	return filepath.Join(dir, name), nil
}

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// beginRunLog optionally tees stdout/stderr to an append-only file. end must be deferred.
func beginRunLog(cmd *cobra.Command, cfg *config.Config, command string) (end func(), err error) {
	end = func() {}
	path, err := resolveRunLogPath(cfg, command)
	if err != nil {
		return end, err
	}
	if path == "" {
		return end, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return end, fmt.Errorf("run log_file: mkdir parent: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return end, fmt.Errorf("run log_file: open %s: %w", path, err)
	}

	fileW := &lockedWriter{w: f}
	prevOut := cmd.OutOrStdout()
	prevErr := cmd.ErrOrStderr()
	cmd.SetOut(io.MultiWriter(prevOut, fileW))
	cmd.SetErr(io.MultiWriter(prevErr, fileW))

	_ = log.WriteLine(cmd.OutOrStdout(), log.LevelInfo, "log_file: "+path)

	end = func() {
		_ = f.Close()
		cmd.SetOut(prevOut)
		cmd.SetErr(prevErr)
	}
	return end, nil
}
