package config

type Config struct {
	Name                 string            `yaml:"name"`
	APIKey               string            `yaml:"api_key"`
	APIKeyFile           string            `yaml:"api_key_file"`
	Model                string            `yaml:"model"`
	MaxTokens            int               `yaml:"max_tokens"`
	ContextWindow        int               `yaml:"context_window"`
	Role                 string            `yaml:"role"`
	Temperature          float64           `yaml:"temperature"`
	TopP                 float64           `yaml:"top_p"`
	FrequencyPenalty     float64           `yaml:"frequency_penalty"`
	PresencePenalty      float64           `yaml:"presence_penalty"`
	Thread               string            `yaml:"thread"`
	OmitHistory          bool              `yaml:"omit_history"`
	URL                  string            `yaml:"url"`
	CompletionsPath      string            `yaml:"completions_path"`
	ModelsPath           string            `yaml:"models_path"`
	ResponsesPath        string            `yaml:"responses_path"`
	SpeechPath           string            `yaml:"speech_path"`
	ImageGenerationsPath string            `yaml:"image_generations_path"`
	ImageEditsPath       string            `yaml:"image_edits_path"`
	TranscriptionsPath   string            `yaml:"transcriptions_path"`
	AuthHeader           string            `yaml:"auth_header"`
	AuthTokenPrefix      string            `yaml:"auth_token_prefix"`
	CommandPrompt        string            `yaml:"command_prompt"`
	CommandPromptColor   string            `yaml:"command_prompt_color"`
	OutputPrompt         string            `yaml:"output_prompt"`
	OutputPromptColor    string            `yaml:"output_prompt_color"`
	AutoCreateNewThread  bool              `yaml:"auto_create_new_thread"`
	AutoShellTitle       bool              `yaml:"auto_shell_title"`
	TrackTokenUsage      bool              `yaml:"track_token_usage"`
	SkipTLSVerify        bool              `yaml:"skip_tls_verify"`
	Multiline            bool              `yaml:"multiline"`
	Web                  bool              `yaml:"web"`
	WebContextSize       string            `yaml:"web_context_size"`
	Seed                 int               `yaml:"seed"`
	Effort               string            `yaml:"effort"`
	Voice                string            `yaml:"voice"`
	UserAgent            string            `yaml:"user_agent"`
	CustomHeaders        map[string]string `yaml:"custom_headers"`
	Agent                AgentConfig       `yaml:"agent"`
}

type AgentConfig struct {
	Mode string `yaml:"mode"`

	// Budgets / guardrails (0 = unlimited)
	MaxSteps      int `yaml:"max_steps"`
	MaxIterations int `yaml:"max_iterations"`
	MaxWallTime   int `yaml:"max_wall_time"`
	MaxShellCalls int `yaml:"max_shell_calls"`
	MaxLLMCalls   int `yaml:"max_llm_calls"`
	MaxFileOps    int `yaml:"max_file_ops"`
	MaxLLMTokens  int `yaml:"max_llm_tokens"`

	// Safety/policy
	AllowedTools           []string `yaml:"allowed_tools"`
	DeniedShellCommands    []string `yaml:"denied_shell_commands"`
	AllowedFileOps         []string `yaml:"allowed_file_ops"`
	RestrictFilesToWorkDir bool     `yaml:"restrict_files_to_work_dir"`

	// Logging / artifacts
	WritePlanJSON bool   `yaml:"write_plan_json"`
	PlanJSONPath  string `yaml:"plan_json_path"`
}
