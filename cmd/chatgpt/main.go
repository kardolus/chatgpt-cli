package main

import (
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const secretEnv = "OPENAI_API_KEY"

var queryMode bool
var clearHistory bool

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chatgpt",
		Short: "ChatGPT Proof of Concept",
		Long:  "A Proof of Concept for building ChatGPT clients.",
		RunE:  run,
	}

	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")
	rootCmd.PersistentFlags().BoolVarP(&clearHistory, "clear-history", "c", false, "Clear the history of ChatGPT CLI")

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if clearHistory {
		historyHandler := history.New()
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
	client := client.New(http.New().WithSecret(secret), history.New())

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
