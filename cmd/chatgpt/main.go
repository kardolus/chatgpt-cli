package main

import (
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
	queryMode    bool
	clearHistory bool
	showVersion  bool
	GitCommit    string
	GitVersion   string
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

	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")
	rootCmd.PersistentFlags().BoolVarP(&clearHistory, "clear-history", "c", false, "Clear the history of ChatGPT CLI")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display the version information")

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if showVersion {
		fmt.Printf("ChatGPT CLI version %s (commit %s)\n", GitVersion, GitCommit)
		return nil
	}

	if clearHistory {
		historyHandler := history.NewDefault()
		err := historyHandler.Delete()
		if err != nil {
			return err
		}
		fmt.Println("History successfully cleared.")
	}

	if len(args) == 0 {
		if clearHistory {
			return nil
		} else {
			return errors.New("you must specify your query")
		}
	}

	secret := viper.GetString(secretEnv)
	if secret == "" {
		return errors.New("missing environment variable: " + secretEnv)
	}
	client := client.NewDefault(http.New().WithSecret(secret), history.NewDefault())

	// Check if there is input from the pipe (stdin)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeContent, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from pipe: %w", err)
		}
		client.ProvideContext(string(pipeContent))
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
	return nil
}
