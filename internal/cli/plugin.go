package cli

import (
	"os"
	"path/filepath"
	"strings"
)

// Plugin name and binary conventions for kubectl-kzero (ROADMAP #52).
//
// kubectl follows the plugin spec:
// https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/
// A plugin is any executable on $PATH whose name starts with kubectl-.
// When the user runs kubectl <name> <sub-args> and kubectl-<name> exists,
// kubectl invokes that binary with argv[0]=kubectl-<name> and the remaining
// args after kubectl <name>.
//
// Both binaries (kzero and kubectl-kzero) share ./cmd/kzero; command parsing
// is identical. The basename kubectl-kzero triggers plugin discovery.
const (
	pluginBinaryBasename = "kubectl-kzero"
	// PluginRootCommand is the kubectl subcommand name (kubectl kzero …).
	PluginRootCommand = "kzero"
)

// IsPluginInvocation reports whether the current binary was launched under
// the kubectl-kzero basename (kubectl plugin discovery or a direct PATH hit).
//
// Detection rules, in order:
//  1. If KZERO_FORCE_KUBECTL_PLUGIN=1/true/yes, always treat as a plugin.
//  2. Otherwise, only the literal basename kubectl-kzero (optional .exe).
func IsPluginInvocation() bool {
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("KZERO_FORCE_KUBECTL_PLUGIN"))); v == "1" || v == "true" || v == "yes" {
		return true
	}
	base := filepath.Base(os.Args[0])
	base = strings.TrimSuffix(base, ".exe")
	return base == pluginBinaryBasename
}

// InvocationLabel returns a short tag for how the binary was invoked
// (kzero, kubectl-kzero, or the basename when launched via a symlink).
// Used by version output so operators can tell which entry point fired.
func InvocationLabel() string {
	if IsPluginInvocation() {
		return "kubectl-" + PluginRootCommand
	}
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".exe")
	if name == "" || name == "." {
		return PluginRootCommand
	}
	return name
}
