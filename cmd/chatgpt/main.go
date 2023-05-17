package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strings"
)

const secretEnv = "OPENAI_API_KEY"

var (
	queryMode       bool
	clearHistory    bool
	showVersion     bool
	interactiveMode bool
	modelName       string
	GitCommit       string
	GitVersion      string
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
	rootCmd.PersistentFlags().BoolVarP(&clearHistory, "clear-history", "c", false, "Clear the history of ChatGPT CLI")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display the version information")
	rootCmd.PersistentFlags().StringVarP(&modelName, "model", "m", "", "Use a custom GPT model by specifying the model name")

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	secret := viper.GetString(secretEnv)
	if secret == "" {
		return errors.New("missing environment variable: " + secretEnv)
	}
	client := client.New(http.New().WithSecret(secret), history.New())

	if modelName != "" {
		client = client.WithModel(modelName)
	}

	// Check if there is input from the pipe (stdin)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeContent, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from pipe: %w", err)
		}
		client.ProvideContext(string(pipeContent))
	}

	if showVersion {
		fmt.Printf("ChatGPT CLI version %s (commit %s)\n", GitVersion, GitCommit)
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
