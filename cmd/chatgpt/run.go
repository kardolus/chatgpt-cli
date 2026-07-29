package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/cache"
	"github.com/kardolus/chatgpt-cli/cmd/chatgpt/utils"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"
)

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
		}

		key, err := config.ReadAPIKeyFile(cfg.APIKeyFile)
		if err != nil {
			return err
		}
		cfg.APIKey = key
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

	if tmp := os.Getenv(internal.ConfigHomeEnv); tmp != "" && !utils.FileExists(viper.ConfigFileUsed()) {
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
				utils.MergeMaps(mcp.Params, newParams)
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
		mode, err := utils.ResolveAgentMode(agentMode, cfg.Agent.Mode)
		if err != nil {
			return err
		}

		goal, err := utils.BuildAgentGoal(chatContext, args)
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
			utils.BoolToOnOff(cfg.Multiline),
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
