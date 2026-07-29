package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func initConfig(rootCmd *cobra.Command) (config.Config, error) {
	// Set default name for environment variables if no config is loaded yet.
	viper.SetDefault("name", "openai")

	// Read only the `name` field from the config to determine the environment prefix.
	configHome, err := internal.GetConfigHome()
	if err != nil {
		return config.Config{}, err
	}

	configName := "config"
	if modelTarget != "" {
		configName += "." + modelTarget
	}

	viper.SetConfigName(configName)
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
	rootNode, err := utils.ReadConfigWithComments(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config with comments: %w", err)
	}

	// Update the config with the new values.
	if err := utils.UpdateConfigNode(rootNode, changedValues); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	// Write back the updated config with preserved comments.
	return utils.SaveConfigWithComments(configFile, rootNode)
}

func syncFlagsWithViper(cmd *cobra.Command) error {
	for _, meta := range configMetadata {
		aliasFlagName := utils.ToAliasFlagName(meta.Key)
		if err := syncFlag(cmd, meta, aliasFlagName); err != nil {
			return err
		}
	}
	return nil
}

func syncFlag(cmd *cobra.Command, meta ConfigMetadata, alias string) error {
	mainFlag := cmd.Flag(meta.FlagName)
	aliasFlag := cmd.Flag(alias)

	// If either doesn't exist, just treat it as "not changed"
	mainChanged := mainFlag != nil && mainFlag.Changed
	aliasChanged := aliasFlag != nil && aliasFlag.Changed

	if !mainChanged && !aliasChanged {
		return nil
	}

	var (
		value interface{}
		err   error
	)

	switch meta.DefaultValue.(type) {
	case string:
		if aliasChanged {
			value = aliasFlag.Value.String()
		} else {
			value = mainFlag.Value.String()
		}

	case int:
		if aliasChanged {
			value, err = cmd.Flags().GetInt(alias)
		} else {
			value, err = cmd.Flags().GetInt(meta.FlagName)
		}

	case bool:
		if aliasChanged {
			value, err = cmd.Flags().GetBool(alias)
		} else {
			value, err = cmd.Flags().GetBool(meta.FlagName)
		}

	case float64:
		if aliasChanged {
			value, err = cmd.Flags().GetFloat64(alias)
		} else {
			value, err = cmd.Flags().GetFloat64(meta.FlagName)
		}

	case []string:
		if aliasChanged {
			value, err = cmd.Flags().GetStringSlice(alias)
		} else {
			value, err = cmd.Flags().GetStringSlice(meta.FlagName)
		}

	default:
		return fmt.Errorf("unsupported type for %s", meta.FlagName)
	}

	if err != nil {
		return fmt.Errorf("failed to parse value for %s: %w", meta.FlagName, err)
	}

	viper.Set(meta.Key, value)
	return nil
}

func createConfigFromViper() config.Config {
	return config.Config{
		Name:                 viper.GetString("name"),
		APIKey:               viper.GetString("api_key"),
		APIKeyFile:           viper.GetString("api_key_file"),
		Model:                viper.GetString("model"),
		MaxTokens:            viper.GetInt("max_tokens"),
		ContextWindow:        viper.GetInt("context_window"),
		Role:                 viper.GetString("role"),
		Temperature:          viper.GetFloat64("temperature"),
		TopP:                 viper.GetFloat64("top_p"),
		FrequencyPenalty:     viper.GetFloat64("frequency_penalty"),
		PresencePenalty:      viper.GetFloat64("presence_penalty"),
		Thread:               viper.GetString("thread"),
		OmitHistory:          viper.GetBool("omit_history"),
		URL:                  viper.GetString("url"),
		CompletionsPath:      viper.GetString("completions_path"),
		ResponsesPath:        viper.GetString("responses_path"),
		TranscriptionsPath:   viper.GetString("transcriptions_path"),
		SpeechPath:           viper.GetString("speech_path"),
		ImageGenerationsPath: viper.GetString("image_generations_path"),
		ImageEditsPath:       viper.GetString("image_edits_path"),
		ModelsPath:           viper.GetString("models_path"),
		AuthHeader:           viper.GetString("auth_header"),
		AuthTokenPrefix:      viper.GetString("auth_token_prefix"),
		CommandPrompt:        viper.GetString("command_prompt"),
		CommandPromptColor:   viper.GetString("command_prompt_color"),
		OutputPrompt:         viper.GetString("output_prompt"),
		OutputPromptColor:    viper.GetString("output_prompt_color"),
		AutoCreateNewThread:  viper.GetBool("auto_create_new_thread"),
		AutoShellTitle:       viper.GetBool("auto_shell_title"),
		TrackTokenUsage:      viper.GetBool("track_token_usage"),
		SkipTLSVerify:        viper.GetBool("skip_tls_verify"),
		HTTPTimeout:          viper.GetInt("http_timeout"),
		MaxRetries:           viper.GetInt("max_retries"),
		RetryBaseDelayMs:     viper.GetInt("retry_base_delay_ms"),
		Multiline:            viper.GetBool("multiline"),
		Seed:                 viper.GetInt("seed"),
		Effort:               viper.GetString("effort"),
		Web:                  viper.GetBool("web"),
		WebContextSize:       viper.GetString("web_context_size"),
		Voice:                viper.GetString("voice"),
		UserAgent:            viper.GetString("user_agent"),
		CustomHeaders:        viper.GetStringMapString("custom_headers"),
		Agent: config.AgentConfig{
			Mode:          viper.GetString("agent.mode"),
			WorkDir:       viper.GetString("agent.work_dir"),
			DryRun:        viper.GetBool("agent.dry_run"),
			MaxSteps:      viper.GetInt("agent.max_steps"),
			MaxIterations: viper.GetInt("agent.max_iterations"),
			MaxWallTime:   viper.GetInt("agent.max_wall_time"),
			MaxShellCalls: viper.GetInt("agent.max_shell_calls"),
			MaxLLMCalls:   viper.GetInt("agent.max_llm_calls"),
			MaxFileOps:    viper.GetInt("agent.max_file_ops"),
			MaxLLMTokens:  viper.GetInt("agent.max_llm_tokens"),

			AllowedTools:           viper.GetStringSlice("agent.allowed_tools"),
			DeniedShellCommands:    viper.GetStringSlice("agent.denied_shell_commands"),
			AllowedFileOps:         viper.GetStringSlice("agent.allowed_file_ops"),
			RestrictFilesToWorkDir: viper.GetBool("agent.restrict_files_to_work_dir"),

			WritePlanJSON: viper.GetBool("agent.write_plan_json"),
			PlanJSONPath:  viper.GetString("agent.plan_json_path"),
		},
	}
}
