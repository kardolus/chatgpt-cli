package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/cache"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/internal"
	"github.com/spf13/pflag"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

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

func run(cmd *cobra.Command, args []string) error {
	if err := syncFlagsWithViper(cmd); err != nil {
		return err
	}

	cfg = createConfigFromViper()

	changedFlags := make(map[string]bool)
	cmd.Flags().Visit(func(f *pflag.Flag) {
		changedFlags[f.Name] = true
	})

	if err := utils.ValidateFlags(cfg.Model, changedFlags); err != nil {
		return err
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

	if cfg.APIKey == "" {
		if cfg.APIKeyFile == "" {
			return errors.New("API key is required. Provide it via --set-api-key, --set-api-key-file, env var, or config file")
		} else {
			content, err := os.ReadFile(cfg.APIKeyFile)
			if err != nil {
				return err
			}
			cfg.APIKey = strings.TrimSpace(string(content))
		}
	}

	ctx := context.Background()

	hs, _ := history.New() // do not error out

	if hs != nil {
		slug, writeConfig := utils.GenerateThreadName(cfg, interactiveMode, newThread)

		hs.SetThread(slug)

		if writeConfig {
			if err := saveConfig(map[string]interface{}{"thread": slug}); err != nil {
				return fmt.Errorf("failed to save new thread to config: %w", err)
			}
		}

		if cfg.AutoShellTitle {
			if err := setShellTitle(slug); err != nil {
				return err
			}
		}
	}

	c := client.New(http.RealCallerFactory, hs, &client.RealTime{}, fsio.NewRealReader(fsio.DefaultBufferSize), &fsio.RealWriter{}, cfg)

	if ServiceURL != "" {
		c = c.WithServiceURL(ServiceURL)
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

	if cmd.Flag("transcribe").Changed {
		text, err := c.Transcribe(audioFile)
		if err != nil {
			return err
		}
		sugar.Infoln(text)
		return nil
	}

	// Check if there is input from the pipe (stdin)
	var chatContext string
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		pipeContent, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read from pipe: %w", err)
		}

		isBinary := utils.IsBinary(pipeContent)
		if isBinary {
			ctx = context.WithValue(ctx, internal.BinaryDataKey, pipeContent)
		} else {
			chatContext = string(pipeContent)

			if strings.Trim(chatContext, "\n ") != "" {
				hasPipe = true
			}

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

	if !client.GetCapabilities(c.Config.Model).SupportsStreaming {
		queryMode = true
	}

	if cmd.Flag("mcp").Changed {
		if mcpEndpoint == "" {
			return errors.New("--mcp is required")
		}
		if mcpTool == "" {
			return errors.New("--mcp-tool is required when using --mcp")
		}

		headers, err := utils.ParseMCPHeaders(mcpHeaders)
		if err != nil {
			return err
		}

		mcp := api.MCPRequest{
			Endpoint: mcpEndpoint,
			Headers:  headers,
			Tool:     mcpTool,
			Params:   map[string]interface{}{},
		}

		if cmd.Flag("mcp-params").Changed {
			mcp.Params, err = utils.ParseMCPParams([]string{paramsJSON}...)
			if err != nil {
				return err
			}
		}

		if cmd.Flag("mcp-param").Changed {
			newParams, err := utils.ParseMCPParams(paramsList...)
			if err != nil {
				return err
			}
			if len(mcp.Params) > 0 {
				mergeMaps(mcp.Params, newParams)
			} else {
				mcp.Params = newParams
			}
		}

		base, err := client.NewMCPTransport(mcp.Endpoint, c.Caller, mcp.Headers)
		if err != nil {
			return err
		}

		cacheHome, err := internal.GetCacheHome()
		if err != nil {
			return err
		}

		sessionsDir := filepath.Join(cacheHome, "mcp", "sessions")

		store := cache.NewFileStore(sessionsDir)
		sessionStore := cache.New(store)

		transport := client.NewSessionTransport(base, sessionStore)

		c = c.WithTransport(transport)

		if err := c.InjectMCPContext(mcp); err != nil {
			return err
		}

		if len(args) == 0 && !hasPipe && !interactiveMode && !agentEnabled {
			sugar.Infof("[MCP: %s] Context injected. No query submitted.", mcp.Tool)
			return nil
		}
	}

	if agentEnabled {
		mode, err := resolveAgentMode(agentMode, cfg.Agent.Mode)
		if err != nil {
			return err
		}

		goal, err := buildAgentGoal(chatContext, args)
		if err != nil {
			return err
		}

		answer, err := runAgent(ctx, c, cfg, mode, goal)
		if err != nil {
			return err
		}

		// write ONE history interaction (you said you already created the helper)
		if hs != nil && !cfg.OmitHistory {
			if err := appendAgentRunToHistory(hs, cfg.Role, goal, answer); err != nil {
				return err
			}
		}

		return nil
	}

	if interactiveMode {
		sugar.Infof(
			"Entering interactive mode. Using thread '%s'. Multiline mode is %s.\n"+
				"Commands: 'clear' (clear screen), 'multiline' (toggle multiline input), 'exit' or Ctrl+C (quit).\n\n",
			hs.GetThread(),
			boolToOnOff(cfg.Multiline),
		)

		var readlineCfg *readline.Config
		if cfg.OmitHistory || cfg.AutoCreateNewThread || newThread {
			readlineCfg = &readline.Config{
				Prompt: "",
			}
		} else {
			store, err := history.New()
			if err != nil {
				return err
			}

			h := history.NewHistory(store)
			userHistory, err := h.ParseUserHistory(cfg.Thread)
			if err != nil {
				return err
			}

			historyFile, err := utils.CreateHistoryFile(userHistory)
			if err != nil {
				return err
			}
			readlineCfg = &readline.Config{
				Prompt:      "",
				HistoryFile: historyFile,
			}
		}

		rl, err := readline.NewEx(readlineCfg)
		if err != nil {
			return err
		}

		defer rl.Close()

		commandPrompt := func(counter, usage int) string {
			return utils.FormatPrompt(c.Config.CommandPrompt, counter, usage, time.Now())
		}

		cmdColor, cmdReset := utils.ColorToAnsi(c.Config.CommandPromptColor)
		outputColor, outPutReset := utils.ColorToAnsi(c.Config.OutputPromptColor)

		multiline := cfg.Multiline

		qNum, usage := 1, 0
		for {
			rl.SetPrompt(commandPrompt(qNum, usage))

			fmt.Print(cmdColor)
			input, err := readInput(rl, &multiline)
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

		if cmd.Flag("speak").Changed && cmd.Flag("output").Changed {
			return c.SynthesizeSpeech(chatContext+strings.Join(args, " "), outputFile)
		}

		if cmd.Flag("draw").Changed && cmd.Flag("output").Changed {
			if cmd.Flag("image").Changed {
				return c.EditImage(chatContext+strings.Join(args, " "), imageFile, outputFile)
			}
			return c.GenerateImage(chatContext+strings.Join(args, " "), outputFile)
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

func boolToOnOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

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

func readInput(rl *readline.Instance, multiline *bool) (string, error) {
	var lines []string

	sugar := zap.S()
	if *multiline {
		sugar.Infoln("Multiline mode enabled. Type 'EOF' on a new line to submit your query.")
	}

	// Custom keybinding to handle backspace in multiline mode
	rl.Config.SetListener(func(line []rune, pos int, key rune) ([]rune, int, bool) {
		// Check if backspace is pressed and if multiline mode is enabled
		if *multiline && key == readline.CharBackspace && pos == 0 && len(lines) > 0 {
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
		case "multiline":
			if *multiline {
				sugar.Infoln("Multiline mode disabled.")
			} else {
				sugar.Infoln("Multiline mode enabled. Type 'EOF' on a new line to submit your query.")
			}
			*multiline = !*multiline
			continue
		case "exit", "/q":
			return "", io.EOF
		}

		if *multiline {
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

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func setShellTitle(title string) error {
	f := os.Stdout

	if !isTerminal(f) {
		// Not a TTY: silently skip
		return nil
	}

	// ANSI: ESC ] 0 ; <title> BEL
	_, err := fmt.Fprintf(f, "\033]0;%s\007", title)
	if err != nil {
		return fmt.Errorf("failed to write shell title: %w", err)
	}

	return nil
}

func appendAgentRunToHistory(store history.Store, systemRole, goal, answer string) error {
	thread := store.GetThread()

	entries, err := store.ReadThread(thread)
	if err != nil {
		// treat missing history file as empty thread
		if errors.Is(err, os.ErrNotExist) {
			entries = nil
		} else {
			return err
		}
	}

	now := time.Now()

	// Ensure a system message exists at the beginning (optional but matches your UX)
	if len(entries) == 0 || strings.ToLower(strings.TrimSpace(entries[0].Role)) != "system" {
		entries = append(entries, history.History{
			Message: api.Message{
				Role:    "system",
				Content: systemRole,
			},
			Timestamp: now,
		})
	}

	entries = append(entries,
		history.History{
			Message:   api.Message{Role: "user", Content: goal},
			Timestamp: now,
		},
		history.History{
			Message:   api.Message{Role: "assistant", Content: answer},
			Timestamp: now,
		},
	)

	// store already has thread set, but being explicit is fine
	store.SetThread(thread)
	return store.Write(entries)
}

func resolveAgentMode(flagMode, cfgMode string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(flagMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(cfgMode))
	}
	if mode == "" {
		mode = "react"
	}

	// allow some aliases, normalize to "react" or "plan"
	switch mode {
	case "react":
		return "react", nil
	case "plan":
		return "plan", nil
	case "plan_execute", "plan-execute":
		return "plan", nil
	default:
		return "", fmt.Errorf("unknown agent mode %q (expected react|plan)", mode)
	}
}

func buildAgentGoal(chatContext string, args []string) (string, error) {
	var parts []string
	if s := strings.TrimSpace(chatContext); s != "" {
		parts = append(parts, s)
	}
	if len(args) > 0 {
		parts = append(parts, strings.Join(args, " "))
	}

	goal := strings.TrimSpace(strings.Join(parts, "\n"))
	if goal == "" {
		return "", errors.New("missing agent goal (provide args or pipe)")
	}
	return goal, nil
}

func runAgent(
	ctx context.Context,
	c *client.Client,
	cfg config.Config,
	mode string,
	goal string,
) (string, error) {
	clk := agent.NewRealClock()
	llm := agent.NewClientLLM(c)

	tools, err := buildAgentTools(llm)
	if err != nil {
		return "", err
	}

	policy, err := buildAgentPolicy(cfg)
	if err != nil {
		return "", err
	}

	budget := agent.NewDefaultBudget(agent.BudgetLimits{
		MaxSteps:      cfg.Agent.MaxSteps,
		MaxWallTime:   time.Duration(cfg.Agent.MaxWallTime) * time.Second,
		MaxShellCalls: cfg.Agent.MaxShellCalls,
		MaxLLMCalls:   cfg.Agent.MaxLLMCalls,
		MaxFileOps:    cfg.Agent.MaxFileOps,
		MaxLLMTokens:  cfg.Agent.MaxLLMTokens,
		MaxIterations: cfg.Agent.MaxIterations,
	})

	runner := agent.NewDefaultRunner(tools, clk, budget, policy)

	baseOpts := []agent.BaseOption{
		agent.WithWorkDir(cfg.Agent.WorkDir),
		agent.WithDryRun(cfg.Agent.DryRun),
		agent.WithHumanLogger(zap.S()),
	}

	switch mode {
	case "react":
		a, err := agent.New(agent.ModeReAct, agent.Deps{
			Clock:  clk,
			LLM:    llm,
			Runner: runner,
			Budget: budget,
		}, baseOpts...)
		if err != nil {
			return "", err
		}
		return a.RunAgentGoal(ctx, goal)

	case "plan":
		logs, err := agent.NewLogs()
		if err != nil {
			return "", err
		}
		defer logs.Close()

		var planner agent.Planner = agent.NewDefaultPlanner(
			llm,
			budget,
			clk,
			agent.WithPlannerRawSink(func(raw string) {
				if !cfg.Agent.WritePlanJSON {
					return
				}
				planPath := strings.TrimSpace(cfg.Agent.PlanJSONPath)
				if planPath == "" {
					planPath = filepath.Join(logs.Dir, "plan.json")
				}
				_ = os.WriteFile(planPath, []byte(strings.TrimSpace(raw)), 0o644)
			}),
		)

		// wrap it
		planner = agent.NewLoggingPlanner(planner, logs)

		// Plan mode wants debug logger too
		planOpts := append([]agent.BaseOption{}, baseOpts...)
		planOpts = append(planOpts, agent.WithDebugLogger(logs.DebugLogger))

		a, err := agent.New(agent.ModePlanExecute, agent.Deps{
			Clock:   clk,
			Planner: planner,
			Runner:  runner,
			LLM:     llm,
			Budget:  budget,
		}, planOpts...)
		if err != nil {
			return "", err
		}
		return a.RunAgentGoal(ctx, goal)

	default:
		return "", fmt.Errorf("internal error: unsupported mode %q", mode)
	}
}

func buildAgentTools(llm agent.LLM) (agent.Tools, error) {
	sh := agent.NewExecShellRunner()
	r := fsio.NewRealReader(fsio.DefaultBufferSize)
	w := &fsio.RealWriter{}
	files := agent.NewFSIOFileOps(r, w)

	return agent.Tools{
		Shell: sh,
		LLM:   llm,
		Files: files,
	}, nil
}

func buildAgentPolicy(cfg config.Config) (agent.Policy, error) {
	allowedTools, err := parseToolKinds(cfg.Agent.AllowedTools)
	if err != nil {
		return nil, err
	}

	return agent.NewDefaultPolicy(agent.PolicyLimits{
		AllowedTools:           allowedTools,
		DeniedShellCommands:    cfg.Agent.DeniedShellCommands,
		AllowedFileOps:         cfg.Agent.AllowedFileOps,
		RestrictFilesToWorkDir: cfg.Agent.RestrictFilesToWorkDir,
	}), nil
}

func toAliasFlagName(viperKey string) string {
	s := strings.ReplaceAll(viperKey, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
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
	aliasFlagName := toAliasFlagName(meta.Key)

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

func isNonConfigSetter(name string) bool {
	return name == "set-completions"
}

func isGeneralFlag(name string) bool {
	var generalFlags = map[string]bool{
		"query":           true,
		"interactive":     true,
		"config":          true,
		"version":         true,
		"new-thread":      true,
		"list-models":     true,
		"list-threads":    true,
		"clear-history":   true,
		"delete-thread":   true,
		"show-history":    true,
		"prompt":          true,
		"agent":           true,
		"set-completions": true,
		"help":            true,
		"role-file":       true,
		"image":           true,
		"audio":           true,
		"speak":           true,
		"draw":            true,
		"output":          true,
		"transcribe":      true,
		"mcp":             true,
		"mcp-header":      true,
		"mcp-param":       true,
		"mcp-params":      true,
		"mcp-tool":        true,
		"target":          true,
	}

	return generalFlags[name]
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
		aliasFlagName := toAliasFlagName(meta.Key)
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

func parseToolKinds(in []string) ([]agent.ToolKind, error) {
	out := make([]agent.ToolKind, 0, len(in))
	seen := map[agent.ToolKind]bool{}

	for _, raw := range in {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" {
			continue
		}

		var k agent.ToolKind
		switch s {
		case "shell":
			k = agent.ToolShell
		case "llm":
			k = agent.ToolLLM
		case "files", "file":
			k = agent.ToolFiles
		default:
			return nil, fmt.Errorf("unknown agent.allowed_tools entry %q (expected shell|llm|files)", raw)
		}

		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}

	// If config is empty, decide your behavior. I’d rather error than silently allow all.
	if len(out) == 0 {
		return nil, errors.New("agent.allowed_tools is empty (expected at least one of shell|llm|files)")
	}

	return out, nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

func mergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}
