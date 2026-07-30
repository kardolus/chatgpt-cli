package main

import (
	"os"

	"github.com/kardolus/chatgpt-cli/internal"
	"go.uber.org/zap/zapcore"

	"github.com/kardolus/chatgpt-cli/config"
	"github.com/spf13/cobra"
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
	useSpeak        bool
	useDraw         bool
	agentMode       string
	agentEnabled    bool
	promptFile      string
	roleFile        string
	imageFile       string
	audioFile       string
	outputFile      string
	threadName      string
	ServiceURL      string
	shell           string
	mcpEndpoint     string
	mcpTool         string
	mcpHeaders      []string
	modelTarget     string
	paramsList      []string
	paramsJSON      string
	jsonMode        bool
	responseFormat  string
	toolsFlag       bool
	cfg             config.Config
)

type ConfigMetadata struct {
	Key          string
	FlagName     string
	DefaultValue interface{}
	Description  string
}

var configMetadata = []ConfigMetadata{
	{"model", "set-model", "gpt-4o", "Set a new default model by specifying the model name"},
	{"max_tokens", "set-max-tokens", 4096, "Set a new default max token size"},
	{"context_window", "set-context-window", 8192, "Set a new default context window size"},
	{"thread", "set-thread", "default", "Set a new active thread by specifying the thread name"},
	{"api_key", "set-api-key", "", "Set the API key for authentication"},
	{"api_key_file", "set-api-key-file", "", "Load the API key from a file"},
	{"role", "set-role", "You are a helpful assistant.", "Set the role of the AI assistant"},
	{"url", "set-url", "https://api.openai.com", "Set the API base URL"},
	{"completions_path", "set-completions-path", "/v1/chat/completions", "Set the completions API endpoint"},
	{"responses_path", "set-responses-path", "/v1/responses", "Set the responses API endpoint"},
	{"transcriptions_path", "set-transcriptions-path", "/v1/audio/transcriptions", "Set the transcriptions API endpoint"},
	{"speech_path", "set-speech-path", "/v1/audio/speech", "Set the speech API endpoint"},
	{"image_generations_path", "set-image-generations-path", "/v1/images/generations", "Set the image generation API endpoint"},
	{"image_edits_path", "set-image-edits-path", "/v1/images/edits", "Set the image edits API endpoint"},
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
	{"auto_create_new_thread", "set-auto-create-new-thread", false, "Create a new thread for each interactive session"},
	{"auto_shell_title", "set-auto-shell-title", false, "Set the title of the shell to the name of the current thread"},
	{"track_token_usage", "set-track-token-usage", false, "Track token usage"},
	{"skip_tls_verify", "set-skip-tls-verify", false, "Skip TLS certificate verification"},
	{"http_timeout", "set-http-timeout", 60, "Set the HTTP client timeout in seconds (0 for no timeout)"},
	{"multiline", "set-multiline", false, "Enables multiline mode while in interactive mode"},
	{"seed", "set-seed", 0, "Sets the seed for deterministic sampling (Beta)"},
	{"name", "set-name", "openai", "The prefix for environment variable overrides"},
	{"effort", "set-effort", "low", "Set the reasoning effort"},
	{"web", "set-web", false, "Enable web search"},
	{"web_context_size", "set-web-context-size", "low", "Set the context size for web search"},
	{"voice", "set-voice", "nova", "Set the voice used by tts models"},
	{"agent.mode", "set-agent-mode", "react", "Default agent mode (react|plan)"},
	{"agent.max_steps", "set-agent-max-steps", 10, "Max steps (plan mode)"},
	{"agent.max_iterations", "set-agent-max-iterations", 10, "Max iterations (react mode)"},
	{"agent.max_wall_time", "set-agent-max-wall-time", 0, "Max wall time in seconds (0=unlimited)"},
	{"agent.max_shell_calls", "set-agent-max-shell-calls", 0, "Max shell calls (0=unlimited)"},
	{"agent.max_llm_calls", "set-agent-max-llm-calls", 10, "Max LLM calls (0=unlimited)"},
	{"agent.max_file_ops", "set-agent-max-file-ops", 0, "Max file ops (0=unlimited)"},
	{"agent.max_llm_tokens", "set-agent-max-llm-tokens", 0, "Max LLM tokens (0=unlimited)"},
	{"agent.allowed_tools", "set-agent-allowed-tools", []string{"shell", "llm", "files"}, "Allowed tools for agent"},
	{"agent.denied_shell_commands", "set-agent-denied-shell-commands", []string{"rm", "sudo", "dd", "mkfs", "shutdown", "reboot"}, "Denied shell commands"},
	{"agent.allowed_file_ops", "set-agent-allowed-file-ops", []string{"read", "write"}, "Allowed file ops"},
	{"agent.restrict_files_to_work_dir", "set-agent-restrict-files-to-work-dir", true, "Restrict file ops to workdir"},
	{"agent.write_plan_json", "set-agent-write-plan-json", true, "Write plan.json in plan mode"},
	{"agent.plan_json_path", "set-agent-plan-json-path", "", "Override plan.json path"},
	{"agent.work_dir", "set-agent-work-dir", ".", "Agent working directory (default: .)"},
	{"agent.dry_run", "set-agent-dry-run", false, "Agent dry-run (no side effects)"},
	{"user_agent", "set-user-agent", "chatgpt-cli", "Set the User-Agent in request header"},
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

	// Parse flags early so modelTarget gets filled from `--target`
	_ = rootCmd.ParseFlags(os.Args[1:])

	sugar := zap.S()

	var err error
	if cfg, err = initConfig(rootCmd); err != nil {
		sugar.Fatalf("Config initialization failed: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		sugar.Fatalln(err)
	}
}
