package api

type Speech struct {
	Model          string `json:"model"`
	Input          string `json:"input"`
	Voice          string `json:"voice"`
	ResponseFormat string `json:"response_format"`
}

type MiniMaxSpeech struct {
	Model        string              `json:"model"`
	Text         string              `json:"text"`
	Stream       bool                `json:"stream"`
	OutputFormat string              `json:"output_format"`
	VoiceSetting MiniMaxVoiceSetting `json:"voice_setting"`
	AudioSetting MiniMaxAudioSetting `json:"audio_setting"`
}

type MiniMaxVoiceSetting struct {
	VoiceID string `json:"voice_id"`
}

type MiniMaxAudioSetting struct {
	Format string `json:"format"`
}
