package main

import (
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-poc/client"
	"github.com/kardolus/chatgpt-poc/http"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const secretEnv = "OPENAI_API_KEY"

var queryMode bool

func main() {
	var rootCmd = &cobra.Command{
		Use:   "chatgpt",
		Short: "ChatGPT Proof of Concept",
		Long:  "A Proof of Concept for building ChatGPT clients.",
		RunE:  run,
	}

	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")

	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.New("you must specify your query")
	}

	secret := viper.GetString(secretEnv)
	if secret == "" {
		return errors.New("missing environment variable: " + secretEnv)
	}
	client := client.New(http.New().WithSecret(secret))

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
