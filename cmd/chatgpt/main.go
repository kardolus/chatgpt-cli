package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io"
	"os"
	"strings"
)

var (
	queryMode       bool
	clearHistory    bool
	showVersion     bool
	showConfig      bool
	interactiveMode bool
	listModels      bool
	modelName       string
	maxTokens       int
	GitCommit       string
	GitVersion      string
	ServiceURL      string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chatgpt",
		Short: "ChatGPT CLI Tool",
		Long: "A powerful ChatGPT client that enables seamless interactions with the GPT model. " +
			"Provides multiple modes and context management features, including the ability to " +
			"pipe custom context into the conversation.",
		RunE: run,
	}

	rootCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Use interactive mode")
	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")
	rootCmd.PersistentFlags().BoolVar(&clearHistory, "clear-history", false, "Clear the history of ChatGPT CLI")
	rootCmd.PersistentFlags().BoolVarP(&showConfig, "config", "c", false, "Display the configuration")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display the version information")
	rootCmd.PersistentFlags().BoolVarP(&listModels, "list-models", "l", false, "List available models")
	rootCmd.PersistentFlags().StringVar(&modelName, "set-model", "", "Set a new default GPT model by specifying the model name")
	rootCmd.PersistentFlags().IntVar(&maxTokens, "set-max-tokens", 0, "Set a new default max token size by specifying the max tokens")

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Flags that do not require an API key
	if showVersion {
		fmt.Printf("ChatGPT CLI version %s (commit %s)\n", GitVersion, GitCommit)
		return nil
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

	if clearHistory {
		historyHandler := history.New()
		err := historyHandler.Delete()
		if err != nil {
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
			fmt.Printf(c)
		}
		return nil
	}

	client, err := client.New(http.New(), config.New(), history.New())
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
		scanner := bufio.NewScanner(os.Stdin)
		qNum := 1
		for {
			fmt.Printf("Q%d: ", qNum)
			scanned := scanner.Scan()
			if !scanned {
				if err := scanner.Err(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				// Exit the loop if no more input (e.g., Ctrl+D)
				break
			}
			line := scanner.Text()
			if line == "exit" {
				break
			}
			if err := client.Stream(line); err != nil {
				fmt.Println("Error:", err)
			} else {
				// Handle the streamed response here, which currently does nothing
				qNum++
			}
		}
	} else {
		if len(args) == 0 {
			return errors.New("you must specify your query")
		}
		if queryMode {
			result, err := client.Query(strings.Join(args, " "))
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
