package main

import (
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/utils"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
	"io"
	"os"
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
	hasPipe         bool
	threadName      string
	ServiceURL      string
	shell           string
)

type ConfigMetadata struct {
	Key          string
	FlagName     string
	DefaultValue interface{}
	Description  string
}

var configMetadata = []ConfigMetadata{
	{"model", "set-model", "gpt-4", "Set a new default GPT model by specifying the model name"},
	{"max_tokens", "set-max-tokens", 4096, "Set a new default max token size"},
	{"context_window", "set-context-window", 8192, "Set a new default context window size"},
	{"thread", "set-thread", "default", "Set a new active thread by specifying the thread name"},
	{"api_key", "set-api-key", "", "Set the API key for authentication"},
	{"url", "set-url", "https://api.openai.com", "Set the API base URL"},
	{"completions_path", "set-completions-path", "/v1/chat/completions", "Set the completions API endpoint"},
	{"models_path", "set-models-path", "/v1/models", "Set the models API endpoint"},
	{"auth_header", "set-auth-header", "Authorization", "Set the authorization header"},
	{"auth_token_prefix", "set-auth-token-prefix", "Bearer", "Set the authorization token prefix"},
	{"command_prompt", "set-command-prompt", "[%datetime] [Q%counter] [%usage]", "Set the command prompt format"},
	{"output_prompt", "set-output-prompt", "", "Set the output prompt format"},
	{"temperature", "set-temperature", 1.0, "Set the sampling temperature"},
	{"top_p", "set-top-p", 1.0, "Set the top-p value for nucleus sampling"},
	{"frequency_penalty", "set-frequency-penalty", 0.0, "Set the frequency penalty"},
	{"presence_penalty", "set-presence-penalty", 0.0, "Set the presence penalty"},
	{"omit_history", "set-omit-history", false, "Omit history in the conversation"},
	{"auto_create_new_thread", "set-auto-create-new-thread", true, "Automatically create a new thread for each session"},
	{"track_token_usage", "set-track-token-usage", true, "Track token usage"},
	{"skip_tls_verify", "set-skip-tls-verify", false, "Skip TLS certificate verification"},
	{"debug", "set-debug", false, "Enable debug mode"},
}

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

	setCustomHelp(rootCmd)

	rootCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Use interactive mode")
	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")
	rootCmd.PersistentFlags().BoolVar(&clearHistory, "clear-history", false, "Clear all prior conversation context for the current thread")
	rootCmd.PersistentFlags().BoolVarP(&showConfig, "config", "c", false, "Display the configuration")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display the version information")
	rootCmd.PersistentFlags().BoolVarP(&listModels, "list-models", "l", false, "List available models")
	rootCmd.PersistentFlags().BoolVarP(&listThreads, "list-threads", "", false, "List available threads")
	rootCmd.PersistentFlags().StringVar(&threadName, "delete-thread", "", "Delete the specified thread")
	rootCmd.PersistentFlags().StringVar(&shell, "set-completions", "", "Generate autocompletion script for your current shell")

	if err := initConfig(rootCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Config initialization failed: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Sync flag values to Viper manually for the keys in configMetadata, keeping the correct type
	for _, meta := range configMetadata {
		aliasFlagName := strings.ReplaceAll(meta.Key, "_", "-")
		var value interface{}
		var err error

		// Check if either the set-* flag or its alias is changed
		if cmd.Flag(meta.FlagName).Changed || cmd.Flag(aliasFlagName).Changed {
			switch meta.DefaultValue.(type) {
			case string:
				value = cmd.Flag(meta.FlagName).Value.String()
				if cmd.Flag(aliasFlagName).Changed {
					value = cmd.Flag(aliasFlagName).Value.String()
				}
			case int:
				value, err = cmd.Flags().GetInt(meta.FlagName)
				if cmd.Flag(aliasFlagName).Changed {
					value, err = cmd.Flags().GetInt(aliasFlagName)
				}
			case bool:
				value, err = cmd.Flags().GetBool(meta.FlagName)
				if cmd.Flag(aliasFlagName).Changed {
					value, err = cmd.Flags().GetBool(aliasFlagName)
				}
			case float64:
				value, err = cmd.Flags().GetFloat64(meta.FlagName)
				if cmd.Flag(aliasFlagName).Changed {
					value, err = cmd.Flags().GetFloat64(aliasFlagName)
				}
			default:
				return fmt.Errorf("unsupported type for %s", meta.FlagName)
			}

			if err != nil {
				return fmt.Errorf("failed to parse value for %s: %w", meta.FlagName, err)
			}

			// Set the value in Viper for runtime use
			viper.Set(meta.Key, value)
		}
	}

	changedValues := map[string]interface{}{}
	for _, meta := range configMetadata {
		if cmd.Flag(meta.FlagName).Changed {
			changedValues[meta.Key] = viper.Get(meta.Key)
		}
	}

	if len(changedValues) > 0 {
		return saveConfig(changedValues)
	}

	if cmd.Flag("set-completions").Changed {
		return config.GenCompletions(cmd, shell)
	}

	if cmd.Flag("delete-thread").Changed {
		cm := configmanager.New(config.New())

		if err := cm.DeleteThread(threadName); err != nil {
			return err
		}
		fmt.Printf("Successfully deleted thread %s\n", threadName)
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
		allSettings := viper.AllSettings()

		configBytes, err := yaml.Marshal(allSettings)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		fmt.Println(string(configBytes))
		return nil
	}

	// Flags that require an API key
	hs, _ := history.New() // do not error out
	client, err := client.New(http.RealCallerFactory, config.New(), hs, interactiveMode)
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

		context := string(pipeContent)

		if strings.Trim(context, "\n ") != "" {
			hasPipe = true
		}
		client.ProvideContext(context)
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
		fmt.Printf("Entering interactive mode. Using thread ‘%s’. Type ‘clear’ to clear the screen, ‘exit’ to quit, or press Ctrl+C.\n\n", hs.GetThread())
		rl, err := readline.New("")
		if err != nil {
			return err
		}
		defer rl.Close()

		commandPrompt := func(counter, usage int) string {
			return config.FormatPrompt(client.Config.CommandPrompt, counter, usage, time.Now())
		}

		qNum, usage := 1, 0
		for {
			rl.SetPrompt(commandPrompt(qNum, usage))

			line, err := rl.Readline()
			if errors.Is(err, readline.ErrInterrupt) || err == io.EOF {
				fmt.Println("Bye!")
				break
			}

			if line == "clear" {
				ansiClearScreenCode := "\033[H\033[2J"
				fmt.Print(ansiClearScreenCode)
				continue
			}

			if line == "exit" || line == "/q" {
				fmt.Println("Bye!")
				if queryMode {
					fmt.Printf("Total tokens used: %d\n", usage)
				}
				break
			}

			fmtOutputPrompt := config.FormatPrompt(client.Config.OutputPrompt, qNum, usage, time.Now())

			if queryMode {
				result, qUsage, err := client.Query(line)
				if err != nil {
					fmt.Println("Error:", err)
				} else {
					fmt.Printf("%s\n\n", fmtOutputPrompt+result)
					usage += qUsage
					qNum++
				}
			} else {
				fmt.Print(fmtOutputPrompt)
				if err := client.Stream(line); err != nil {
					fmt.Fprintln(os.Stderr, "Error:", err)
				} else {
					fmt.Println()
					qNum++
				}
			}
		}
	} else {
		if len(args) == 0 && !hasPipe {
			return errors.New("you must specify your query or provide input via a pipe")
		}
		if queryMode {
			result, usage, err := client.Query(strings.Join(args, " "))
			if err != nil {
				return err
			}
			fmt.Println(result)

			if client.Config.TrackTokenUsage {
				fmt.Printf("\n[Token Usage: %d]\n", usage)
			}
		} else {
			if err := client.Stream(strings.Join(args, " ")); err != nil {
				return err
			}
		}
	}
	return nil
}

func initConfig(rootCmd *cobra.Command) error {
	cm := configmanager.New(config.New()).WithEnvironment()

	viper.SetEnvPrefix(cm.Config.Name)
	viper.AutomaticEnv()

	for _, meta := range configMetadata {
		// Convert the key from snake_case to kebab-case for the alias flag
		aliasFlagName := strings.ReplaceAll(meta.Key, "_", "-")

		// Add both regular and "set-" flags based on its type
		switch v := meta.DefaultValue.(type) {
		case string:
			rootCmd.PersistentFlags().String(meta.FlagName, v, meta.Description)
			rootCmd.PersistentFlags().String(aliasFlagName, v, fmt.Sprintf("Alias for setting %s", meta.Key))
		case int:
			rootCmd.PersistentFlags().Int(meta.FlagName, v, meta.Description)
			rootCmd.PersistentFlags().Int(aliasFlagName, v, fmt.Sprintf("Alias for setting %s", meta.Key))
		case bool:
			rootCmd.PersistentFlags().Bool(meta.FlagName, v, meta.Description)
			rootCmd.PersistentFlags().Bool(aliasFlagName, v, fmt.Sprintf("Alias for setting %s", meta.Key))
		case float64:
			rootCmd.PersistentFlags().Float64(meta.FlagName, v, meta.Description)
			rootCmd.PersistentFlags().Float64(aliasFlagName, v, fmt.Sprintf("Alias for setting %s", meta.Key))
		}

		// Bind each "set-" flag and regular alias flag to the corresponding viper key
		viper.BindPFlag(meta.Key, rootCmd.PersistentFlags().Lookup(meta.FlagName))
		viper.BindPFlag(meta.Key, rootCmd.PersistentFlags().Lookup(aliasFlagName))
		viper.SetDefault(meta.Key, meta.DefaultValue)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	configHome, err := utils.GetConfigHome()
	if err != nil {
		return err
	}
	viper.AddConfigPath(configHome)

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return nil
		} else {
			return err
		}
	}

	return nil
}

func saveConfig(changedValues map[string]interface{}) error {
	if len(changedValues) == 0 {
		return nil
	}

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		configHome, err := utils.GetConfigHome()
		if err != nil {
			return err
		}
		configFile = fmt.Sprintf("%s/config.yaml", configHome)
	}

	// Load the existing config to merge changes
	existingConfig := map[string]interface{}{}
	fileData, err := os.ReadFile(configFile)
	if err == nil {
		_ = yaml.Unmarshal(fileData, &existingConfig)
	}

	// Merge the changed values
	for key, value := range changedValues {
		key = strings.ReplaceAll(key, "-", "_") // Ensure keys are in the correct format
		existingConfig[key] = value
	}

	// Write the updated config back to the file
	yamlData, err := yaml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	return os.WriteFile(configFile, yamlData, 0644)
}

func setCustomHelp(rootCmd *cobra.Command) {
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		fmt.Println("ChatGPT CLI - A powerful client for interacting with GPT models.")

		fmt.Println("\nUsage:")
		fmt.Println("  chatgpt [flags]\n")

		fmt.Println("General Flags:")
		printFlagWithPadding("-q, --query", "Use query mode instead of stream mode")
		printFlagWithPadding("-i, --interactive", "Use interactive mode")
		printFlagWithPadding("-c, --config", "Display the configuration")
		printFlagWithPadding("-v, --version", "Display the version information")
		printFlagWithPadding("-l, --list-models", "List available models")
		printFlagWithPadding("--list-threads", "List available threads")
		printFlagWithPadding("--clear-history", "Clear the history of the current thread")
		printFlagWithPadding("--delete-thread", "Delete the specified thread")
		printFlagWithPadding("--set-completions", "Generate autocompletion script for your current shell")
		fmt.Println()

		fmt.Println("Configuration Setters:")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if strings.HasPrefix(f.Name, "set-") && !isNonConfigSetter(f.Name) {
				printFlagWithPadding("--"+f.Name, f.Usage)
			}
		})

		fmt.Println("\nRuntime Value Overrides:")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if isConfigAlias(f.Name) {
				printFlagWithPadding("--"+f.Name, "Override value for "+strings.ReplaceAll(f.Name, "_", "-"))
			}
		})

		fmt.Println("\nEnvironment Variables:")
		fmt.Println("  You can also use environment variables to set config values. For example:")
		fmt.Printf("  %s_API_KEY=your_api_key chatgpt --query 'Hello'\n", strings.ToUpper(viper.GetEnvPrefix()))

		configHome, _ := utils.GetConfigHome()

		fmt.Println("\nConfiguration File:")
		fmt.Println("  All configuration changes made with the setters will be saved in the config.yaml file.")
		fmt.Printf("  The config.yaml file is located in the following path:")
		fmt.Printf(" %s/config.yaml\n", configHome)
		fmt.Println("  You can edit this file manually to change configuration settings as well.")
	})
}

// Helper function to check if a flag is a non-config-specific setter
func isNonConfigSetter(name string) bool {
	return name == "set-completions"
}

// Helper function to check if a flag is a general flag
func isGeneralFlag(name string) bool {
	switch name {
	case "query", "interactive", "config", "version", "list-models", "list-threads", "clear-history", "delete-thread", "set-completions", "help":
		return true
	default:
		return false
	}
}

// Helper function to check if a flag is a config alias
func isConfigAlias(name string) bool {
	return !strings.HasPrefix(name, "set-") && !isGeneralFlag(name)
}

func printFlagWithPadding(name, description string) {
	padding := 30
	fmt.Printf("  %-*s %s\n", padding, name, description)
}

// TODO populate the config struct and pass it to the client
// TODO get rid of your own config parsing logic
// TODO update README
