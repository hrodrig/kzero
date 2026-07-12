package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletion_generatesScript(t *testing.T) {
	t.Parallel()

	for _, shell := range completionShells {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr bytes.Buffer
			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"completion", shell})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("completion %s: %v", shell, err)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr must be empty, got %q", stderr.String())
			}
			out := stdout.String()
			if len(out) < 40 {
				t.Fatalf("expected completion script on stdout, got %q", out)
			}
			if !strings.Contains(out, "kzero") {
				t.Fatalf("expected script to mention kzero, got prefix %q", out[:min(80, len(out))])
			}
		})
	}
}

func TestCompletion_rejectsInvalidShell(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"completion", "csh"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unsupported shell")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout must stay empty on error, got %q", stdout.String())
	}
	msg := err.Error()
	if !strings.Contains(msg, "csh") && !strings.Contains(strings.ToLower(msg), "valid") {
		t.Fatalf("expected invalid-shell message, got %v", err)
	}
}

func TestCompletion_requiresExactlyOneArg(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		args []string
	}{
		{name: "none", args: []string{"completion"}},
		{name: "tooMany", args: []string{"completion", "bash", "zsh"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			cmd := newRootCmd()
			cmd.SetOut(&stdout)
			cmd.SetErr(&stdout)
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
