package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for bash, zsh, or fish.

To load completions:

Bash:
  $ source <(rootline completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ rootline completion bash > /etc/bash_completion.d/rootline
  # macOS:
  $ rootline completion bash > $(brew --prefix)/etc/bash_completion.d/rootline

Zsh:
  $ source <(rootline completion zsh)
  # To load completions for each session, execute once:
  $ rootline completion zsh > "${fpath[1]}/_rootline"

Fish:
  $ rootline completion fish | source
  # To load completions for each session, execute once:
  $ rootline completion fish > ~/.config/fish/completions/rootline.fish
`,
	ValidArgs: []string{"bash", "zsh", "fish"},
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE:      runCompletion,
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

func runCompletion(cmd *cobra.Command, args []string) error {
	switch args[0] {
	case "bash":
		return rootCmd.GenBashCompletionV2(cmd.OutOrStdout(), true)
	case "zsh":
		return rootCmd.GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
	default:
		return fmt.Errorf("unsupported shell: %s", args[0])
	}
}
