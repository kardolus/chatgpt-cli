package config

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func GenCompletions(command *cobra.Command, shell string) error {

	var completionCmd = &cobra.Command{
		Use:   "chatgpt --set-completions [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `To load completions:

Bash:

  $ source <(chatgpt --set-completions bash)

  # To load completions for each session, execute once:
  # Linux:
  $ chatgpt --set-completions bash > /etc/bash_completion.d/chatgpt
  # macOS:
  $ chatgpt --set-completions bash > /usr/local/etc/bash_completion.d/chatgpt

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ chatgpt --set-completions zsh > "${fpath[1]}/_chatgpt"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ chatgpt --set-completions fish | source

  # To load completions for each session, execute once:
  $ chatgpt --set-completions fish > ~/.config/fish/completions/chatgpt.fish

PowerShell:

  PS> chatgpt --set-completions powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> chatgpt --set-completions powershell > chatgpt.ps1
  # and source this file from your PowerShell profile.
`,
		Args:                  cobra.ExactArgs(1),
		DisableFlagsInUseLine: true,
		DisableFlagParsing:    true,
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				command.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				command.Root().GenZshCompletion(os.Stdout)
			case "fish":
				command.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				command.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			case "-h", "--help":
				cmd.Help()
			default:
				fmt.Printf(`
Usage:
  chatgpt --set-completions [bash|zsh|fish|powershell]

Flags:
  -h, --help   help for --set-completions

Invalid Arg: %s
`, args[0])
			}
		},
	}
	completionCmd.SetArgs([]string{shell})
	return completionCmd.Execute()
}
