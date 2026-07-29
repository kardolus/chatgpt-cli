package main

import (
	"fmt"
	"strings"

	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

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
		printFlagWithPadding("--transcribe", "Transcribe an audio file")
		printFlagWithPadding("--speak", "Use text-to-speech")
		printFlagWithPadding("--draw", "Draw an image")
		printFlagWithPadding("--output", "The output audio file for text-to-speech")
		printFlagWithPadding("--role-file", "Set the system role from the specified file")
		printFlagWithPadding("--debug", "Print debug messages")
		printFlagWithPadding("--agent", "Enable agent mode")
		printFlagWithPadding("--target", "Load configuration from config.<target>.yaml")
		printFlagWithPadding("--mcp", "MCP endpoint URL (e.g. http://localhost:3333)")
		printFlagWithPadding("--mcp-tool", "Tool name to call on the MCP server")
		printFlagWithPadding("--mcp-header", "HTTP header for MCP call (repeatable, 'Key: Value')")
		printFlagWithPadding("--mcp-param", "Key-value pair as key=value. Can be specified multiple times")
		printFlagWithPadding("--mcp-params", "Provide parameters as a raw JSON string")
		printFlagWithPadding("--set-completions", "Generate autocompletion script for your current shell")
		sugar.Infoln()

		sugar.Infoln("Persistent Configuration Setters:")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if strings.HasPrefix(f.Name, "set-") && !utils.IsNonConfigSetter(f.Name) {
				printFlagWithPadding("--"+f.Name, f.Usage)
			}
		})

		sugar.Infoln("\nRuntime Value Overrides:")
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if utils.IsConfigAlias(f.Name) {
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
	rootCmd.PersistentFlags().BoolVarP(&useSpeak, "speak", "", false, "Use text-to-speak")
	rootCmd.PersistentFlags().BoolVarP(&useDraw, "draw", "", false, "Draw an image")
	rootCmd.PersistentFlags().StringVarP(&promptFile, "prompt", "p", "", "Provide a prompt file")
	rootCmd.PersistentFlags().StringVarP(&roleFile, "role-file", "", "", "Provide a role file")
	rootCmd.PersistentFlags().StringVarP(&imageFile, "image", "", "", "Provide an image from a local path or URL")
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "", "", "Provide an output file for text-to-speech")
	rootCmd.PersistentFlags().StringVarP(&audioFile, "audio", "", "", "Provide an audio file from a local path")
	rootCmd.PersistentFlags().StringVarP(&audioFile, "transcribe", "", "", "Provide an audio file from a local path")
	rootCmd.PersistentFlags().BoolVarP(&listThreads, "list-threads", "", false, "List available threads")
	rootCmd.PersistentFlags().StringVar(&threadName, "delete-thread", "", "Delete the specified thread")
	rootCmd.PersistentFlags().BoolVar(&showHistory, "show-history", false, "Show the human-readable conversation history")
	rootCmd.PersistentFlags().StringVar(&shell, "set-completions", "", "Generate autocompletion script for your current shell")
	rootCmd.PersistentFlags().StringVar(&modelTarget, "target", "", "Specify the model to target")
	rootCmd.PersistentFlags().StringVar(&mcpEndpoint, "mcp", "", "MCP endpoint URL (e.g. http://localhost:3333)")
	rootCmd.PersistentFlags().StringVar(&mcpTool, "mcp-tool", "", "MCP tool name to call")
	rootCmd.PersistentFlags().StringArrayVar(&mcpHeaders, "mcp-header", []string{}, "MCP header in the form 'Key: Value' (repeatable)")
	rootCmd.PersistentFlags().StringArrayVar(&paramsList, "mcp-param", []string{}, "Key-value pair as key=value. Can be specified multiple times")
	rootCmd.PersistentFlags().StringVar(&paramsJSON, "mcp-params", "", "Provide parameters as a raw JSON string")
	rootCmd.PersistentFlags().BoolVar(&agentEnabled, "agent", false, "Run agent (experimental)")
}

func setupConfigFlags(rootCmd *cobra.Command, meta ConfigMetadata) {
	aliasFlagName := utils.ToAliasFlagName(meta.Key)

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
	case []string:
		rootCmd.PersistentFlags().StringSlice(meta.FlagName, viper.GetStringSlice(meta.Key), meta.Description)
		rootCmd.PersistentFlags().StringSlice(aliasFlagName, viper.GetStringSlice(meta.Key), fmt.Sprintf("Alias for setting %s", meta.Key))
	}

	// Bind the flags directly to Viper keys
	_ = viper.BindPFlag(meta.Key, rootCmd.PersistentFlags().Lookup(meta.FlagName))
	_ = viper.BindPFlag(meta.Key, rootCmd.PersistentFlags().Lookup(aliasFlagName))
	viper.SetDefault(meta.Key, meta.DefaultValue)
}

func printFlagWithPadding(name, description string) {
	sugar := zap.S()
	padding := 30
	sugar.Infof("  %-*s %s", padding, name, description)
}
