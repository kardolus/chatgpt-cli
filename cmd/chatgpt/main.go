package main

import (
	"errors"
	"github.com/kardolus/chatgpt-poc/client"
	"github.com/kardolus/chatgpt-poc/http"
	"log"
	"os"
	"strings"
)

const secretEnv = "OPENAI_API_KEY"

func main() {
	exit(run())
}

func exit(err error) {
	if err == nil {
		os.Exit(0)
	}
	log.Printf("Error: %s\n", err)
	os.Exit(1)
}

func run() error {
	if len(os.Args) <= 1 {
		return errors.New("you must specify your query")
	}

	secret := os.Getenv(secretEnv)
	if secret == "" {
		return errors.New("missing environment variable: " + secretEnv)
	}
	client := client.New(http.New().WithSecret(secret))

	if err := client.Stream(strings.Join(os.Args[1:], " ")); err != nil {
		return err
	}

	return nil
}
