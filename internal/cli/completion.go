package cli

import (
	"fmt"

	"github.com/hrodrig/kzero/internal/exitcode"
	"github.com/spf13/cobra"
)

var completionShells = []string{"bash", "zsh", "fish", "powershell"}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate a completion script for bash, zsh, fish, or powershell.

Write the script to a file or pipe it into your shell's completion loader.
Invalid or missing shell names exit non-zero.

Examples:
  # Bash (session)
  source <(kzero completion bash)

  # Zsh (session)
  source <(kzero completion zsh)

  # Fish (persist)
  kzero completion fish > ~/.config/fish/completions/kzero.fish

  # PowerShell (session)
  kzero completion powershell | Out-String | Invoke-Expression
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             completionShells,
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			root := cmd.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			case "powershell":
				return root.GenPowerShellCompletionWithDesc(out)
			default:
				// OnlyValidArgs should reject this; keep a clear fallback.
				return exitcode.New(exitcode.ConfigError, fmt.Errorf("unsupported shell %q (want bash, zsh, fish, or powershell)", args[0]))
			}
		},
	}
}
