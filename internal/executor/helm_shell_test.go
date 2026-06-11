package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hrodrig/kzero/internal/config"
)

func TestShellHelm_Uninstall(t *testing.T) {
	t.Parallel()

	var got []string
	h := NewShellHelm(HelmDeps{
		Cfg: &config.Config{
			Command: config.CommandConfig{Helm: "/bin/helm"},
		},
		Run: func(ctx context.Context, argv0 string, args, env []string, dir string) ([]byte, error) {
			got = append(got, argv0)
			got = append(got, args...)
			return nil, nil
		},
	})
	step := config.PipelineStep{Name: "prom", Namespace: "mon", Ref: "release.mon/prom", Type: "release"}
	if err := h.Uninstall(context.Background(), step); err != nil {
		t.Fatal(err)
	}
	if len(got) < 3 || got[0] != "/bin/helm" || got[1] != "uninstall" || got[2] != "prom" {
		t.Fatalf("unexpected argv: %v", got)
	}
	if !h.UsesSDK() {
		// shell backend
	} else {
		t.Fatal("shell helm reported sdk")
	}
}

func TestShellHelm_UpgradeInstall(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	script := filepath.Join(dir, "prom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var argv0 string
	h := NewShellHelm(HelmDeps{
		Cfg: &config.Config{Helm: config.HelmConfig{Workspace: dir}},
		Run: func(ctx context.Context, a0 string, args, env []string, d string) ([]byte, error) {
			argv0 = a0
			if a0 != "/bin/sh" || args[0] != script {
				t.Fatalf("unexpected exec: %s %v", a0, args)
			}
			return nil, nil
		},
	})
	step := config.PipelineStep{Name: "prom", Namespace: "mon", Ref: "release.mon/prom", Type: "release"}
	if err := h.UpgradeInstall(context.Background(), step); err != nil {
		t.Fatal(err)
	}
	if argv0 != "/bin/sh" {
		t.Fatalf("got %q", argv0)
	}
}

func TestShellHelm_UpgradeInstall_scriptOverride(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sub := filepath.Join(dir, "monitoring")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(sub, "prom.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var args []string
	h := NewShellHelm(HelmDeps{
		Cfg: &config.Config{Helm: config.HelmConfig{Workspace: dir}},
		Run: func(ctx context.Context, a0 string, a, env []string, d string) ([]byte, error) {
			args = a
			return nil, nil
		},
	})
	step := config.PipelineStep{
		Name: "prom", Namespace: "mon", Ref: "release.mon/prom", Type: "release",
		Script: "monitoring/prom.sh",
	}
	if err := h.UpgradeInstall(context.Background(), step); err != nil {
		t.Fatal(err)
	}
	if len(args) == 0 || args[0] != script {
		t.Fatalf("args=%v want script %s", args, script)
	}
}

func TestShellHelm_UpgradeInstall_missingScript(t *testing.T) {
	t.Parallel()

	h := NewShellHelm(HelmDeps{
		Cfg: &config.Config{Helm: config.HelmConfig{Workspace: t.TempDir()}},
		Run: func(context.Context, string, []string, []string, string) ([]byte, error) {
			return nil, nil
		},
	})
	step := config.PipelineStep{Name: "prom", Namespace: "mon", Ref: "release.mon/prom", Type: "release"}
	if err := h.UpgradeInstall(context.Background(), step); err == nil {
		t.Fatal("expected missing script error")
	}
}

func TestShellHelm_UpgradeInstall_emptyWorkspace(t *testing.T) {
	t.Parallel()

	h := NewShellHelm(HelmDeps{Cfg: &config.Config{}})
	step := config.PipelineStep{Name: "prom", Ref: "release.mon/prom", Type: "release"}
	if err := h.UpgradeInstall(context.Background(), step); err == nil || !strings.Contains(err.Error(), "helm.workspace") {
		t.Fatalf("got %v", err)
	}
}

func TestNewHelmReleases_routing(t *testing.T) {
	t.Parallel()

	deps := HelmDeps{Cfg: &config.Config{Run: config.RunConfig{Execution: "shell"}}}
	shell, err := NewHelmReleases(deps.Cfg, deps)
	if err != nil || shell.UsesSDK() {
		t.Fatalf("shell path: sdk=%v err=%v", shell.UsesSDK(), err)
	}

	nativeCfg := &config.Config{Run: config.RunConfig{Execution: "native"}}
	sdk, err := NewHelmReleases(nativeCfg, deps)
	if err != nil || !sdk.UsesSDK() {
		t.Fatalf("native path: sdk=%v err=%v", sdk.UsesSDK(), err)
	}
}

func TestHelmPath_default(t *testing.T) {
	t.Parallel()
	if HelmPath(nil) != "helm" {
		t.Fatal("expected default helm")
	}
	if HelmPath(&config.Config{Command: config.CommandConfig{Helm: "/opt/helm"}}) != "/opt/helm" {
		t.Fatal("expected custom helm")
	}
}
