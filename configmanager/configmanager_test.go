package configmanager_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/types"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitConfigManager(t *testing.T) {
	spec.Run(t, "Config Manager", testConfig, spec.Report(report.Terminal{}))
}

func testConfig(t *testing.T, when spec.G, it spec.S) {
	const (
		defaultMaxTokens        = 10
		defaultName             = "default-name"
		defaultURL              = "default-url"
		defaultModel            = "default-model"
		defaultRole             = "default-role"
		defaultApiKey           = "default-api-key"
		defaultThread           = "default-thread"
		defaultCompletionsPath  = "default-completions-path"
		defaultModelsPath       = "default-models-path"
		defaultAuthHeader       = "default-auth-header"
		defaultAuthTokenPrefix  = "default-auth-token-prefix"
		defaultOmitHistory      = false
		defaultTemperature      = 1.1
		defaultTopP             = 2.2
		defaultFrequencyPenalty = 3.3
		defaultPresencePenalty  = 4.4
	)

	var (
		mockCtrl        *gomock.Controller
		mockConfigStore *MockConfigStore
		defaultConfig   types.Config
		envPrefix       string
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockConfigStore = NewMockConfigStore(mockCtrl)

		defaultConfig = types.Config{
			Name:             defaultName,
			APIKey:           defaultApiKey,
			Model:            defaultModel,
			MaxTokens:        defaultMaxTokens,
			URL:              defaultURL,
			CompletionsPath:  defaultCompletionsPath,
			ModelsPath:       defaultModelsPath,
			AuthHeader:       defaultAuthHeader,
			AuthTokenPrefix:  defaultAuthTokenPrefix,
			OmitHistory:      defaultOmitHistory,
			Role:             defaultRole,
			Thread:           defaultThread,
			Temperature:      defaultTemperature,
			TopP:             defaultTopP,
			FrequencyPenalty: defaultFrequencyPenalty,
			PresencePenalty:  defaultPresencePenalty,
		}

		envPrefix = strings.ToUpper(defaultConfig.Name) + "_"

		Expect(os.Unsetenv(envPrefix + "API_KEY")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "MODEL")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "MAX_TOKENS")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "URL")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "COMPLETIONS_PATH")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "MODELS_PATH")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "AUTH_HEADER")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "AUTH_TOKEN_PREFIX")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "OMIT_HISTORY")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "ROLE")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "THREAD")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "TEMPERATURE")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "TOP_P")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "FREQUENCY_PENALTY")).To(Succeed())
		Expect(os.Unsetenv(envPrefix + "PRESENCE_PENALTY")).To(Succeed())
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	it("should use default values when no environment variables or user config are provided", func() {
		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
		mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("no user config")).Times(1)

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
		Expect(subject.Config.Name).To(Equal(defaultName))
		Expect(subject.Config.Model).To(Equal(defaultModel))
		Expect(subject.Config.URL).To(Equal(defaultURL))
		Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
		Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
		Expect(subject.Config.AuthHeader).To(Equal(defaultAuthHeader))
		Expect(subject.Config.AuthTokenPrefix).To(Equal(defaultAuthTokenPrefix))
		Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
		Expect(subject.Config.Role).To(Equal(defaultRole))
		Expect(subject.Config.Thread).To(Equal(defaultThread))
		Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
		Expect(subject.Config.TopP).To(Equal(defaultTopP))
		Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
		Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
	})

	it("should prioritize user-provided config over defaults", func() {
		userConfig := types.Config{
			Model:            "user-model",
			MaxTokens:        20,
			URL:              "user-url",
			CompletionsPath:  "user-completions-path",
			ModelsPath:       "user-models-path",
			AuthHeader:       "user-auth-header",
			AuthTokenPrefix:  "user-auth-token-prefix",
			OmitHistory:      true,
			Role:             "user-role",
			Thread:           "user-thread",
			Temperature:      2.5,
			TopP:             3.5,
			FrequencyPenalty: 4.5,
			PresencePenalty:  5.5,
		}

		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).AnyTimes()
		mockConfigStore.EXPECT().Read().Return(userConfig, nil).AnyTimes()

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.Model).To(Equal("user-model"))
		Expect(subject.Config.MaxTokens).To(Equal(20))
		Expect(subject.Config.URL).To(Equal("user-url"))
		Expect(subject.Config.CompletionsPath).To(Equal("user-completions-path"))
		Expect(subject.Config.ModelsPath).To(Equal("user-models-path"))
		Expect(subject.Config.AuthHeader).To(Equal("user-auth-header"))
		Expect(subject.Config.AuthTokenPrefix).To(Equal("user-auth-token-prefix"))
		Expect(subject.Config.OmitHistory).To(BeTrue())
		Expect(subject.Config.Role).To(Equal("user-role"))
		Expect(subject.Config.Thread).To(Equal("user-thread"))
		Expect(subject.Config.Temperature).To(Equal(2.5))
		Expect(subject.Config.TopP).To(Equal(3.5))
		Expect(subject.Config.FrequencyPenalty).To(Equal(4.5))
		Expect(subject.Config.PresencePenalty).To(Equal(5.5))
	})

	it("should prioritize environment variables over default config", func() {
		os.Setenv(envPrefix+"API_KEY", "env-api-key")
		os.Setenv(envPrefix+"MODEL", "env-model")
		os.Setenv(envPrefix+"MAX_TOKENS", "15")
		os.Setenv(envPrefix+"URL", "env-url")
		os.Setenv(envPrefix+"COMPLETIONS_PATH", "env-completions-path")
		os.Setenv(envPrefix+"MODELS_PATH", "env-models-path")
		os.Setenv(envPrefix+"AUTH_HEADER", "env-auth-header")
		os.Setenv(envPrefix+"AUTH_TOKEN_PREFIX", "env-auth-token-prefix")
		os.Setenv(envPrefix+"OMIT_HISTORY", "true")
		os.Setenv(envPrefix+"ROLE", "env-role")
		os.Setenv(envPrefix+"THREAD", "env-thread")
		os.Setenv(envPrefix+"TEMPERATURE", "2.2")
		os.Setenv(envPrefix+"TOP_P", "3.3")
		os.Setenv(envPrefix+"FREQUENCY_PENALTY", "4.4")
		os.Setenv(envPrefix+"PRESENCE_PENALTY", "5.5")

		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).AnyTimes()
		mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("config error")).AnyTimes()

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.APIKey).To(Equal("env-api-key"))
		Expect(subject.Config.Model).To(Equal("env-model"))
		Expect(subject.Config.MaxTokens).To(Equal(15))
		Expect(subject.Config.URL).To(Equal("env-url"))
		Expect(subject.Config.CompletionsPath).To(Equal("env-completions-path"))
		Expect(subject.Config.ModelsPath).To(Equal("env-models-path"))
		Expect(subject.Config.AuthHeader).To(Equal("env-auth-header"))
		Expect(subject.Config.AuthTokenPrefix).To(Equal("env-auth-token-prefix"))
		Expect(subject.Config.OmitHistory).To(BeTrue())
		Expect(subject.Config.Role).To(Equal("env-role"))
		Expect(subject.Config.Thread).To(Equal("env-thread"))
		Expect(subject.Config.Temperature).To(Equal(2.2))
		Expect(subject.Config.TopP).To(Equal(3.3))
		Expect(subject.Config.FrequencyPenalty).To(Equal(4.4))
		Expect(subject.Config.PresencePenalty).To(Equal(5.5))
	})

	it("should prioritize environment variables over user-provided config", func() {
		os.Setenv(envPrefix+"API_KEY", "env-api-key")
		os.Setenv(envPrefix+"MODEL", "env-model")
		os.Setenv(envPrefix+"MAX_TOKENS", "15")
		os.Setenv(envPrefix+"URL", "env-url")
		os.Setenv(envPrefix+"COMPLETIONS_PATH", "env-completions-path")
		os.Setenv(envPrefix+"MODELS_PATH", "env-models-path")
		os.Setenv(envPrefix+"AUTH_HEADER", "env-auth-header")
		os.Setenv(envPrefix+"AUTH_TOKEN_PREFIX", "env-auth-token-prefix")
		os.Setenv(envPrefix+"OMIT_HISTORY", "true")
		os.Setenv(envPrefix+"ROLE", "env-role")
		os.Setenv(envPrefix+"THREAD", "env-thread")
		os.Setenv(envPrefix+"TEMPERATURE", "2.2")
		os.Setenv(envPrefix+"TOP_P", "3.3")
		os.Setenv(envPrefix+"FREQUENCY_PENALTY", "4.4")
		os.Setenv(envPrefix+"PRESENCE_PENALTY", "5.5")

		userConfig := types.Config{
			APIKey:           "user-api-key",
			Model:            "user-model",
			MaxTokens:        20,
			URL:              "user-url",
			CompletionsPath:  "user-completions-path",
			ModelsPath:       "user-models-path",
			AuthHeader:       "user-auth-header",
			AuthTokenPrefix:  "user-auth-token-prefix",
			OmitHistory:      false,
			Role:             "user-role",
			Thread:           "user-thread",
			Temperature:      1.5,
			TopP:             2.5,
			FrequencyPenalty: 3.5,
			PresencePenalty:  4.5,
		}

		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).AnyTimes()
		mockConfigStore.EXPECT().Read().Return(userConfig, nil).AnyTimes()

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.APIKey).To(Equal("env-api-key"))
		Expect(subject.Config.Model).To(Equal("env-model"))
		Expect(subject.Config.MaxTokens).To(Equal(15))
		Expect(subject.Config.URL).To(Equal("env-url"))
		Expect(subject.Config.CompletionsPath).To(Equal("env-completions-path"))
		Expect(subject.Config.ModelsPath).To(Equal("env-models-path"))
		Expect(subject.Config.AuthHeader).To(Equal("env-auth-header"))
		Expect(subject.Config.AuthTokenPrefix).To(Equal("env-auth-token-prefix"))
		Expect(subject.Config.OmitHistory).To(BeTrue())
		Expect(subject.Config.Role).To(Equal("env-role"))
		Expect(subject.Config.Thread).To(Equal("env-thread"))
		Expect(subject.Config.Temperature).To(Equal(2.2))
		Expect(subject.Config.TopP).To(Equal(3.3))
		Expect(subject.Config.FrequencyPenalty).To(Equal(4.4))
		Expect(subject.Config.PresencePenalty).To(Equal(5.5))
	})
}
