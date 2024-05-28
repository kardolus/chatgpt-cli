package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	queryMode       bool
	clearHistory    bool
	showVersion     bool
	showConfig      bool
	interactiveMode bool
	listModels      bool
	listThreads     bool
	modelName       string
	threadName      string
	maxTokens       int
	contextWindow   int
	GitCommit       string
	GitVersion      string
	ServiceURL      string
	shell           string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chatgpt",
		Short: "ChatGPT CLI Tool",
		Long: "A powerful ChatGPT client that enables seamless interactions with the GPT model. " +
			"Provides multiple modes and context management features, including the ability to " +
			"pipe custom context into the conversation.",
		RunE:          run,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Use interactive mode")
	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")
	rootCmd.PersistentFlags().BoolVar(&clearHistory, "clear-history", false, "Clear all prior conversation context for the current thread")
	rootCmd.PersistentFlags().BoolVarP(&showConfig, "config", "c", false, "Display the configuration")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display the version information")
	rootCmd.PersistentFlags().BoolVarP(&listModels, "list-models", "l", false, "List available models")
	rootCmd.PersistentFlags().BoolVarP(&listThreads, "list-threads", "", false, "List available threads")
	rootCmd.PersistentFlags().StringVar(&modelName, "set-model", "", "Set a new default GPT model by specifying the model name")
	rootCmd.PersistentFlags().StringVar(&threadName, "set-thread", "", "Set a new active thread by specifying the thread name")
	rootCmd.PersistentFlags().StringVar(&threadName, "delete-thread", "", "Delete the specified thread")
	rootCmd.PersistentFlags().StringVar(&shell, "set-completions", "", "Generate autocompletion script for your current shell")
	rootCmd.PersistentFlags().IntVar(&maxTokens, "set-max-tokens", 0, "Set a new default max token size by specifying the max tokens")
	rootCmd.PersistentFlags().IntVar(&contextWindow, "set-context-window", 0, "Set a new default context window size")

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Flags that do not require an API key
	if showVersion {
		fmt.Printf("ChatGPT CLI version %s (commit %s)\n", GitVersion, GitCommit)
		return nil
	}

	if cmd.Flag("set-completions").Changed {
		return config.GenCompletions(cmd, shell)
	}

	if cmd.Flag("set-model").Changed {
		cm := configmanager.New(config.New())

		if err := cm.WriteModel(modelName); err != nil {
			return err
		}
		fmt.Println("Model successfully updated to", modelName)
		return nil
	}

	if cmd.Flag("set-max-tokens").Changed {
		cm := configmanager.New(config.New())

		if err := cm.WriteMaxTokens(maxTokens); err != nil {
			return err
		}
		fmt.Println("Max tokens successfully updated to", maxTokens)
		return nil
	}

	if cmd.Flag("set-context-window").Changed {
		cm := configmanager.New(config.New())

		if err := cm.WriteContextWindow(contextWindow); err != nil {
			return err
		}
		fmt.Println("Context window successfully updated to", contextWindow)
		return nil
	}

	if cmd.Flag("set-thread").Changed {
		cm := configmanager.New(config.New())

		if err := cm.WriteThread(threadName); err != nil {
			return err
		}
		fmt.Println("Thread successfully updated to", threadName)
		return nil
	}

	if cmd.Flag("delete-thread").Changed {
		cm := configmanager.New(config.New())

		if err := cm.DeleteThread(threadName); err != nil {
			return err
		}
		fmt.Printf("Successfully deleted thead %s\n", threadName)
		return nil
	}

	if listThreads {
		cm := configmanager.New(config.New())

		threads, err := cm.ListThreads()
		if err != nil {
			return err
		}
		fmt.Println("Available threads:")
		for _, thread := range threads {
			fmt.Println(thread)
		}
		return nil
	}

	if clearHistory {
		cm := configmanager.New(config.New())

		if err := cm.DeleteThread(cm.Config.Thread); err != nil {
			return err
		}

		fmt.Println("History successfully cleared.")
		return nil
	}

	if showConfig {
		cm := configmanager.New(config.New()).WithEnvironment()

		if c, err := cm.ShowConfig(); err != nil {
			return err
		} else {
			fmt.Println(c)
		}
		return nil
	}

	// Flags that require an API key
	hs, _ := history.New() // do not error out
	client, err := client.New(http.RealCallerFactory, config.New(), hs)
	if err != nil {
		return err
	}

	if ServiceURL != "" {
		client = client.WithServiceURL(ServiceURL)
	}

	// Check if there is input from the pipe (stdin)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeContent, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from pipe: %w", err)
		}
		client.ProvideContext(string(pipeContent))
	}

	if listModels {
		models, err := client.ListModels()
		if err != nil {
			return err
		}
		fmt.Println("Available models:")
		for _, model := range models {
			fmt.Println(model)
		}
		return nil
	}

	if interactiveMode {
		fmt.Printf("Entering interactive mode. Type 'exit' and press Enter or press Ctrl+C to quit.\n\n")
		rl, err := readline.New("")
		if err != nil {
			return err
		}
		defer rl.Close()

		prompt := func(counter string) string {
			cm := configmanager.New(config.New())
			if len(cm.Config.CommandPrompt) != 0 {
				return cm.Config.CommandPrompt
			} else {
				return fmt.Sprintf("[%s] [%s]: ", time.Now().Format("2006-01-02 15:04:05"), counter)
			}
		}

		qNum, usage := 1, 0
		for {
			if queryMode {
				rl.SetPrompt(prompt(strconv.Itoa(usage)))
			} else {
				rl.SetPrompt(prompt(fmt.Sprintf("Q%d", qNum)))
			}

			line, err := rl.Readline()
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("Bye!")
				break
			}

			if line == "exit" || line == "/q" {
				fmt.Println("Bye!")
				if queryMode {
					fmt.Printf("Total tokens used: %d\n", usage)
				}
				break
			}

			if queryMode {
				result, qUsage, err := client.Query(line)
				if err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Printf("%s\n\n", result)
					usage += qUsage
				}
			} else {
				if err := client.Stream(line); err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
				} else {
					fmt.Println()
					qNum++
				}
			}
		}
	} else {
		if len(args) == 0 {
			return errors.New("you must specify your query")
		}
		if queryMode {
			result, _, err := client.Query(strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Println(result)
		} else {
			if err := client.Stream(strings.Join(args, " ")); err != nil {
				return err
			}
		}
	}
	return nil
}
