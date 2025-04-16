package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/internal"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	GitCommit       string
	GitVersion      string
	queryMode       bool
	clearHistory    bool
	showHistory     bool
	showVersion     bool
	showDebug       bool
	newThread       bool
	showConfig      bool
	interactiveMode bool
	listModels      bool
	listThreads     bool
	hasPipe         bool
	promptFile      string
	roleFile        string
	imageFile       string
	audioFile       string
	threadName      string
	ServiceURL      string
	shell           string
	cfg             config.Config
)

type ConfigMetadata struct {
	Key          string
	FlagName     string
	DefaultValue interface{}
	Description  string
}

var configMetadata = []ConfigMetadata{
	{"model", "set-model", "gpt-3.5-turbo", "Set a new default model by specifying the model name"},
	{"max_tokens", "set-max-tokens", 4096, "Set a new default max token size"},
	{"context_window", "set-context-window", 8192, "Set a new default context window size"},
	{"thread", "set-thread", "default", "Set a new active thread by specifying the thread name"},
	{"api_key", "set-api-key", "", "Set the API key for authentication"},
	{"role", "set-role", "You are a helpful assistant.", "Set the role of the AI assistant"},
	{"url", "set-url", "https://api.openai.com", "Set the API base URL"},
	{"completions_path", "set-completions-path", "/v1/chat/completions", "Set the completions API endpoint"},
	{"responses_path", "set-responses-path", "/v1/responses", "Set the responses API endpoint"},
	{"models_path", "set-models-path", "/v1/models", "Set the models API endpoint"},
	{"auth_header", "set-auth-header", "Authorization", "Set the authorization header"},
	{"auth_token_prefix", "set-auth-token-prefix", "Bearer ", "Set the authorization token prefix"},
	{"command_prompt", "set-command-prompt", "[%datetime] [Q%counter] [%usage]", "Set the command prompt format for interactive mode"},
	{"command_prompt_color", "set-command-prompt-color", "", "Set the command prompt color"},
	{"output_prompt", "set-output-prompt", "", "Set the output prompt format for interactive mode"},
	{"output_prompt_color", "set-output-prompt-color", "", "Set the output prompt color"},
	{"temperature", "set-temperature", 1.0, "Set the sampling temperature"},
	{"top_p", "set-top-p", 1.0, "Set the top-p value for nucleus sampling"},
	{"frequency_penalty", "set-frequency-penalty", 0.0, "Set the frequency penalty"},
	{"presence_penalty", "set-presence-penalty", 0.0, "Set the presence penalty"},
	{"omit_history", "set-omit-history", false, "Omit history in the conversation"},
	{"auto_create_new_thread", "set-auto-create-new-thread", true, "Create a new thread for each interactive session"},
	{"track_token_usage", "set-track-token-usage", true, "Track token usage"},
	{"skip_tls_verify", "set-skip-tls-verify", false, "Skip TLS certificate verification"},
	{"multiline", "set-multiline", false, "Enables multiline mode while in interactive mode"},
	{"seed", "set-seed", 0, "Sets the seed for deterministic sampling (Beta)"},
	{"name", "set-name", "openai", "The prefix for environment variable overrides"},
	{"effort", "set-effort", "low", "Set the reasoning effort"},
}

func init() {
	internal.SetAllowedLogLevels(zapcore.InfoLevel)
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
	setupFlags(rootCmd)

	sugar := zap.S()

	var err error
	if cfg, err = initConfig(rootCmd); err != nil {
		sugar.Fatalf("Config initialization failed: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		sugar.Fatalln(err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if err := syncFlagsWithViper(cmd); err != nil {
		return err
	}

	cfg = createConfigFromViper()

	if newThread && (cmd.Flag("set-thread").Changed || cmd.Flag("thread").Changed) {
		return errors.New("the --new-thread flag cannot be used with the --set-thread or --thread flags")
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

	sugar := zap.S()

	if showVersion {
		if GitCommit != "homebrew" {
			GitCommit = "commit " + GitCommit
		}
		sugar.Infof("ChatGPT CLI version %s (%s)", GitVersion, GitCommit)
		return nil
	}

	if cmd.Flag("delete-thread").Changed {
		cm := config.NewManager(config.NewStore())

		if err := cm.DeleteThread(threadName); err != nil {
			return err
		}
		sugar.Infof("Successfully deleted thread %s", threadName)
		return nil
	}

	if listThreads {
		cm := config.NewManager(config.NewStore())

		threads, err := cm.ListThreads()
		if err != nil {
			return err
		}
		sugar.Infoln("Available threads:")
		for _, thread := range threads {
			sugar.Infoln(thread)
		}
		return nil
	}

	if clearHistory {
		cm := config.NewManager(config.NewStore())

		if err := cm.DeleteThread(cfg.Thread); err != nil {
			var fileNotFoundError *config.FileNotFoundError
			if errors.As(err, &fileNotFoundError) {
				sugar.Infoln("Thread history does not exist; nothing to clear.")
				return nil
			}
			return err
		}

		sugar.Infoln("History cleared successfully.")
		return nil
	}

	if showHistory {
		var targetThread string
		if len(args) > 0 {
			targetThread = args[0]
		} else {
			targetThread = cfg.Thread
		}

		store, err := history.New()
		if err != nil {
			return err
		}

		h := history.NewHistory(store)

		output, err := h.Print(targetThread)
		if err != nil {
			return err
		}

		sugar.Infoln(output)
		return nil
	}

	if showDebug {
		internal.SetAllowedLogLevels(zapcore.InfoLevel, zapcore.DebugLevel)
	}

	if cmd.Flag("role-file").Changed {
		role, err := utils.FileToString(roleFile)
		if err != nil {
			return err
		}
		cfg.Role = role
		viper.Set("role", role)
	}

	if showConfig {
		allSettings := viper.AllSettings()

		configBytes, err := yaml.Marshal(allSettings)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		sugar.Infoln(string(configBytes))
		return nil
	}

	if viper.GetString("api_key") == "" {
		return errors.New("API key is required. Please set it using the --set-api-key flag, with the runtime flag --api-key or via environment variables")
	}

	ctx := context.Background()

	hs, _ := history.New() // do not error out
	c := client.New(http.RealCallerFactory, hs, &client.RealTime{}, &client.RealFileReader{}, cfg, interactiveMode)

	if ServiceURL != "" {
		c = c.WithServiceURL(ServiceURL)
	}

	if hs != nil && newThread {
		slug := client.GenerateUniqueSlug("cmd_")

		hs.SetThread(slug)

		if err := saveConfig(map[string]interface{}{"thread": slug}); err != nil {
			return fmt.Errorf("failed to save new thread to config: %w", err)
		}
	}

	if cmd.Flag("prompt").Changed {
		prompt, err := utils.FileToString(promptFile)
		if err != nil {
			return err
		}
		c.ProvideContext(prompt)
	}

	if cmd.Flag("image").Changed {
		ctx = context.WithValue(ctx, internal.ImagePathKey, imageFile)
	}

	if cmd.Flag("audio").Changed {
		ctx = context.WithValue(ctx, internal.AudioPathKey, audioFile)
	}

	// Check if there is input from the pipe (stdin)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeContent, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from pipe: %w", err)
		}

		isBinary := utils.IsBinary(pipeContent)
		if isBinary {
			ctx = context.WithValue(ctx, internal.BinaryDataKey, pipeContent)
		}

		chatContext := string(pipeContent)

		if strings.Trim(chatContext, "\n ") != "" {
			hasPipe = true
		}

		if !isBinary {
			c.ProvideContext(chatContext)
		}
	}

	if listModels {
		models, err := c.ListModels()
		if err != nil {
			return err
		}
		sugar.Infoln("Available models:")
		for _, model := range models {
			sugar.Infoln(model)
		}
		return nil
	}

	if tmp := os.Getenv(internal.ConfigHomeEnv); tmp != "" && !fileExists(viper.ConfigFileUsed()) {
		sugar.Warnf("Warning: config.yaml doesn't exist in %s, create it\n", tmp)
	}

	if strings.Contains(c.Config.Model, client.O1ProPattern) {
		// o1-pro models only supports query mode
		queryMode = true
	}

	if interactiveMode {
		sugar.Infof("Entering interactive mode. Using thread '%s'. Type 'clear' to clear the screen, 'exit' to quit, or press Ctrl+C.\n\n", hs.GetThread())
		rl, err := readline.New("")
		if err != nil {
			return err
		}
		defer rl.Close()

		commandPrompt := func(counter, usage int) string {
			return utils.FormatPrompt(c.Config.CommandPrompt, counter, usage, time.Now())
		}

		cmdColor, cmdReset := utils.ColorToAnsi(c.Config.CommandPromptColor)
		outputColor, outPutReset := utils.ColorToAnsi(c.Config.OutputPromptColor)

		qNum, usage := 1, 0
		for {
			rl.SetPrompt(commandPrompt(qNum, usage))

			fmt.Print(cmdColor)
			input, err := readInput(rl, cfg.Multiline)
			fmt.Print(cmdReset)

			if err == io.EOF {
				sugar.Infoln("Bye!")
				return nil
			}

			fmtOutputPrompt := utils.FormatPrompt(c.Config.OutputPrompt, qNum, usage, time.Now())

			if queryMode {
				result, qUsage, err := c.Query(ctx, input)
				if err != nil {
					sugar.Infoln("Error:", err)
				} else {
					sugar.Infof("%s%s%s\n\n", outputColor, fmtOutputPrompt+result, outPutReset)
					usage += qUsage
					qNum++
				}
			} else {
				fmt.Print(outputColor + fmtOutputPrompt)
				if err := c.Stream(ctx, input); err != nil {
					_, _ = fmt.Fprintln(os.Stderr, "Error:", err)
				} else {
					sugar.Infoln()
					qNum++
				}
				fmt.Print(outPutReset)
			}
		}
	} else {
		if len(args) == 0 && !hasPipe {
			return errors.New("you must specify your query or provide input via a pipe")
		}
		if queryMode {
			result, usage, err := c.Query(ctx, strings.Join(args, " "))
			if err != nil {
				return err
			}
			sugar.Infoln(result)

			if c.Config.TrackTokenUsage {
				sugar.Infof("\n[Token Usage: %d]\n", usage)
			}
		} else if err := c.Stream(ctx, strings.Join(args, " ")); err != nil {
			return err
		}
	}
	return nil
}

func initConfig(rootCmd *cobra.Command) (config.Config, error) {
	// Set default name for environment variables if no config is loaded yet.
	viper.SetDefault("name", "openai")

	// Read only the `name` field from the config to determine the environment prefix.
	configHome, err := internal.GetConfigHome()
	if err != nil {
		return config.Config{}, err
	}
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configHome)

	// Attempt to read the configuration file to get the `name` before setting env prefix.
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return config.Config{}, err
		}
	}

	// Retrieve the name from Viper to set the environment prefix.
	envPrefix := viper.GetString("name")
	viper.SetEnvPrefix(envPrefix)
	viper.AutomaticEnv()

	// Now, set up the flags using the fully loaded configuration metadata.
	for _, meta := range configMetadata {
		setupConfigFlags(rootCmd, meta)
	}

	return createConfigFromViper(), nil
}

func readConfigWithComments(configPath string) (*yaml.Node, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return &rootNode, nil
}

func readInput(rl *readline.Instance, multiline bool) (string, error) {
	var lines []string

	sugar := zap.S()
	if multiline {
		sugar.Infoln("Multiline mode enabled. Type 'EOF' on a new line to submit your query.")
	}

	// Custom keybinding to handle backspace in multiline mode
	rl.Config.SetListener(func(line []rune, pos int, key rune) ([]rune, int, bool) {
		// Check if backspace is pressed and if multiline mode is enabled
		if multiline && key == readline.CharBackspace && pos == 0 && len(lines) > 0 {
			fmt.Print("\033[A") // Move cursor up one line

			// Print the last line without clearing
			lastLine := lines[len(lines)-1]
			fmt.Print(lastLine)

			// Remove the last line from the slice
			lines = lines[:len(lines)-1]

			// Set the cursor at the end of the previous line
			return []rune(lastLine), len(lastLine), true
		}
		return line, pos, false // Default behavior for other keys
	})

	for {
		line, err := rl.Readline()
		if errors.Is(err, readline.ErrInterrupt) || err == io.EOF {
			return "", io.EOF
		}

		switch line {
		case "clear":
			fmt.Print("\033[H\033[2J") // ANSI escape code to clear the screen
			continue
		case "exit", "/q":
			return "", io.EOF
		}

		if multiline {
			if line == "EOF" {
				break
			}
			lines = append(lines, line)
		} else {
			return line, nil
		}
	}

	// Join and return all accumulated lines as a single string
	return strings.Join(lines, "\n"), nil
}

func updateConfig(node *yaml.Node, changes map[string]interface{}) error {
	// If the node is not a document or has no content, create an empty mapping node.
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		node.Kind = yaml.DocumentNode
		node.Content = []*yaml.Node{
			{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{},
			},
		}
	}

	// Assume the root is now a mapping node.
	mapNode := node.Content[0]
	if mapNode.Kind != yaml.MappingNode {
		return errors.New("expected a mapping node at the root of the YAML document")
	}

	// Update the values in the mapNode.
	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valueNode := mapNode.Content[i+1]

		key := keyNode.Value
		if newValue, ok := changes[key]; ok {
			newValueStr := fmt.Sprintf("%v", newValue)
			valueNode.Value = newValueStr
		}
	}

	// Add any new keys that don't exist in the current mapNode.
	for key, value := range changes {
		if !keyExistsInNode(mapNode, key) {
			mapNode.Content = append(mapNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: key,
			}, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%v", value),
			})
		}
	}

	return nil
}

func keyExistsInNode(mapNode *yaml.Node, key string) bool {
	for i := 0; i < len(mapNode.Content); i += 2 {
		if mapNode.Content[i].Value == key {
			return true
		}
	}
	return false
}

func saveConfigWithComments(configPath string, node *yaml.Node) error {
	out, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return os.WriteFile(configPath, out, 0644)
}

func saveConfig(changedValues map[string]interface{}) error {
	configFile := viper.ConfigFileUsed()
	configHome, err := internal.GetConfigHome()
	if err != nil {
		return fmt.Errorf("failed to get config home: %w", err)
	}

	// If the config file is not specified, assume it's supposed to be in the default location.
	if configFile == "" {
		configFile = fmt.Sprintf("%s/config.yaml", configHome)
	}

	// Check if the config directory exists.
	if _, err := os.Stat(configHome); os.IsNotExist(err) {
		return fmt.Errorf("config directory does not exist: %s", configHome)
	}

	// Check if the config file itself exists, and create it if it doesn't.
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		file, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		defer file.Close()
	}

	// Read the existing config with comments.
	rootNode, err := readConfigWithComments(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config with comments: %w", err)
	}

	// Update the config with the new values.
	if err := updateConfig(rootNode, changedValues); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Write back the updated config with preserved comments.
	return saveConfigWithComments(configFile, rootNode)
}

func setCustomHelp(rootCmd *cobra.Command) {
	sugar := zap.S()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		sugar.Infoln("ChatGPT CLI - A powerful client for interacting with GPT models.")

		sugar.Infoln("\nUsage:")
		sugar.Infof("  chatgpt [flags]\n")

		sugar.Infoln("General Flags:")
		printFlagWithPadding("-q, --query", "Use query mode instead of stream mode")
		printFlagWithPadding("-i, --interactive", "Use interactive mode")
		printFlagWithPadding("-p, --prompt", "Provide a prompt file for context")
		printFlagWithPadding("-n, --new-thread", "Create a new thread with a random name and target it")
		printFlagWithPadding("-c, --config", "Display the configuration")
		printFlagWithPadding("-v, --version", "Display the version information")
		printFlagWithPadding("-l, --list-models", "List available models")
		printFlagWithPadding("--list-threads", "List available threads")
		printFlagWithPadding("--delete-thread", "Delete the specified thread (supports wildcards)")
		printFlagWithPadding("--clear-history", "Clear the history of the current thread")
		printFlagWithPadding("--show-history [thread]", "Show the human-readable conversation history")
		printFlagWithPadding("--image", "Upload an image from the specified local path or URL")
		printFlagWithPadding("--audio", "Upload an audio file (mp3 or wav)")
		printFlagWithPadding("--role-file", "Set the system role from the specified file")
		printFlagWithPadding("--debug", "Print debug messages")
		printFlagWithPadding("--set-completions", "Generate autocompletion script for your current shell")
		sugar.Infoln()

		sugar.Infoln("Persistent Configuration Setters:")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if strings.HasPrefix(f.Name, "set-") && !isNonConfigSetter(f.Name) {
				printFlagWithPadding("--"+f.Name, f.Usage)
			}
		})

		sugar.Infoln("\nRuntime Value Overrides:")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if isConfigAlias(f.Name) {
				printFlagWithPadding("--"+f.Name, "Override value for "+strings.ReplaceAll(f.Name, "_", "-"))
			}
		})

		sugar.Infoln("\nEnvironment Variables:")
		sugar.Infoln("  You can also use environment variables to set config values. For example:")
		sugar.Infof("  %s_API_KEY=your_api_key chatgpt --query 'Hello'", strings.ToUpper(viper.GetEnvPrefix()))

		configHome, _ := internal.GetConfigHome()

		sugar.Infoln("\nConfiguration File:")
		sugar.Infoln("  All configuration changes made with the setters will be saved in the config.yaml file.")
		sugar.Infof("  The config.yaml file is located in the following path: %s/config.yaml", configHome)
		sugar.Infoln("  You can edit this file manually to change configuration settings as well.")
	})
}

func setupFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().BoolVarP(&interactiveMode, "interactive", "i", false, "Use interactive mode")
	rootCmd.PersistentFlags().BoolVarP(&queryMode, "query", "q", false, "Use query mode instead of stream mode")
	rootCmd.PersistentFlags().BoolVar(&clearHistory, "clear-history", false, "Clear all prior conversation context for the current thread")
	rootCmd.PersistentFlags().BoolVarP(&showConfig, "config", "c", false, "Display the configuration")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "Display the version information")
	rootCmd.PersistentFlags().BoolVarP(&showDebug, "debug", "", false, "Enable debug mode")
	rootCmd.PersistentFlags().BoolVarP(&newThread, "new-thread", "n", false, "Create a new thread with a random name and target it")
	rootCmd.PersistentFlags().BoolVarP(&listModels, "list-models", "l", false, "List available models")
	rootCmd.PersistentFlags().StringVarP(&promptFile, "prompt", "p", "", "Provide a prompt file")
	rootCmd.PersistentFlags().StringVarP(&roleFile, "role-file", "", "", "Provide a role file")
	rootCmd.PersistentFlags().StringVarP(&imageFile, "image", "", "", "Provide an image from a local path or URL")
	rootCmd.PersistentFlags().StringVarP(&audioFile, "audio", "", "", "Provide an audio file from a local path")
	rootCmd.PersistentFlags().BoolVarP(&listThreads, "list-threads", "", false, "List available threads")
	rootCmd.PersistentFlags().StringVar(&threadName, "delete-thread", "", "Delete the specified thread")
	rootCmd.PersistentFlags().BoolVar(&showHistory, "show-history", false, "Show the human-readable conversation history")
	rootCmd.PersistentFlags().StringVar(&shell, "set-completions", "", "Generate autocompletion script for your current shell")
}

func setupConfigFlags(rootCmd *cobra.Command, meta ConfigMetadata) {
	aliasFlagName := strings.ReplaceAll(meta.Key, "_", "-")

	switch meta.DefaultValue.(type) {
	case string:
		rootCmd.PersistentFlags().String(meta.FlagName, viper.GetString(meta.Key), meta.Description)
		rootCmd.PersistentFlags().String(aliasFlagName, viper.GetString(meta.Key), fmt.Sprintf("Alias for setting %s", meta.Key))
	case int:
		rootCmd.PersistentFlags().Int(meta.FlagName, viper.GetInt(meta.Key), meta.Description)
		rootCmd.PersistentFlags().Int(aliasFlagName, viper.GetInt(meta.Key), fmt.Sprintf("Alias for setting %s", meta.Key))
	case bool:
		rootCmd.PersistentFlags().Bool(meta.FlagName, viper.GetBool(meta.Key), meta.Description)
		rootCmd.PersistentFlags().Bool(aliasFlagName, viper.GetBool(meta.Key), fmt.Sprintf("Alias for setting %s", meta.Key))
	case float64:
		rootCmd.PersistentFlags().Float64(meta.FlagName, viper.GetFloat64(meta.Key), meta.Description)
		rootCmd.PersistentFlags().Float64(aliasFlagName, viper.GetFloat64(meta.Key), fmt.Sprintf("Alias for setting %s", meta.Key))
	}

	// Bind the flags directly to Viper keys
	_ = viper.BindPFlag(meta.Key, rootCmd.PersistentFlags().Lookup(meta.FlagName))
	_ = viper.BindPFlag(meta.Key, rootCmd.PersistentFlags().Lookup(aliasFlagName))
	viper.SetDefault(meta.Key, meta.DefaultValue)
}

func isNonConfigSetter(name string) bool {
	return name == "set-completions"
}

func isGeneralFlag(name string) bool {
	switch name {
	case "query", "interactive", "config", "version", "new-thread", "list-models", "list-threads", "clear-history", "delete-thread", "show-history", "prompt", "set-completions", "help", "role-file", "image", "audio":
		return true
	default:
		return false
	}
}

func isConfigAlias(name string) bool {
	return !strings.HasPrefix(name, "set-") && !isGeneralFlag(name)
}

func printFlagWithPadding(name, description string) {
	sugar := zap.S()
	padding := 30
	sugar.Infof("  %-*s %s", padding, name, description)
}

func syncFlagsWithViper(cmd *cobra.Command) error {
	for _, meta := range configMetadata {
		aliasFlagName := strings.ReplaceAll(meta.Key, "_", "-")
		if err := syncFlag(cmd, meta, aliasFlagName); err != nil {
			return err
		}
	}
	return nil
}

func syncFlag(cmd *cobra.Command, meta ConfigMetadata, alias string) error {
	var value interface{}
	var err error

	if cmd.Flag(meta.FlagName).Changed || cmd.Flag(alias).Changed {
		switch meta.DefaultValue.(type) {
		case string:
			value = cmd.Flag(meta.FlagName).Value.String()
			if cmd.Flag(alias).Changed {
				value = cmd.Flag(alias).Value.String()
			}
		case int:
			value, err = cmd.Flags().GetInt(meta.FlagName)
			if cmd.Flag(alias).Changed {
				value, err = cmd.Flags().GetInt(alias)
			}
		case bool:
			value, err = cmd.Flags().GetBool(meta.FlagName)
			if cmd.Flag(alias).Changed {
				value, err = cmd.Flags().GetBool(alias)
			}
		case float64:
			value, err = cmd.Flags().GetFloat64(meta.FlagName)
			if cmd.Flag(alias).Changed {
				value, err = cmd.Flags().GetFloat64(alias)
			}
		default:
			return fmt.Errorf("unsupported type for %s", meta.FlagName)
		}

		if err != nil {
			return fmt.Errorf("failed to parse value for %s: %w", meta.FlagName, err)
		}

		viper.Set(meta.Key, value)
	}
	return nil
}

func createConfigFromViper() config.Config {
	return config.Config{
		Name:                viper.GetString("name"),
		APIKey:              viper.GetString("api_key"),
		Model:               viper.GetString("model"),
		MaxTokens:           viper.GetInt("max_tokens"),
		ContextWindow:       viper.GetInt("context_window"),
		Role:                viper.GetString("role"),
		Temperature:         viper.GetFloat64("temperature"),
		TopP:                viper.GetFloat64("top_p"),
		FrequencyPenalty:    viper.GetFloat64("frequency_penalty"),
		PresencePenalty:     viper.GetFloat64("presence_penalty"),
		Thread:              viper.GetString("thread"),
		OmitHistory:         viper.GetBool("omit_history"),
		URL:                 viper.GetString("url"),
		CompletionsPath:     viper.GetString("completions_path"),
		ResponsesPath:       viper.GetString("responses_path"),
		ModelsPath:          viper.GetString("models_path"),
		AuthHeader:          viper.GetString("auth_header"),
		AuthTokenPrefix:     viper.GetString("auth_token_prefix"),
		CommandPrompt:       viper.GetString("command_prompt"),
		CommandPromptColor:  viper.GetString("command_prompt_color"),
		OutputPrompt:        viper.GetString("output_prompt"),
		OutputPromptColor:   viper.GetString("output_prompt_color"),
		AutoCreateNewThread: viper.GetBool("auto_create_new_thread"),
		TrackTokenUsage:     viper.GetBool("track_token_usage"),
		SkipTLSVerify:       viper.GetBool("skip_tls_verify"),
		Multiline:           viper.GetBool("multiline"),
		Seed:                viper.GetInt("seed"),
		Effort:              viper.GetString("effort"),
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}
