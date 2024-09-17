package configmanager_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/types"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=configmocks_test.go -package=configmanager_test github.com/kardolus/chatgpt-cli/config ConfigStore

func TestUnitConfigManager(t *testing.T) {
	spec.Run(t, "Config Manager", testConfig, spec.Report(report.Terminal{}))
}

func testConfig(t *testing.T, when spec.G, it spec.S) {
	const (
		defaultMaxTokens           = 10
		defaultContextWindow       = 20
		defaultName                = "default-name"
		defaultURL                 = "default-url"
		defaultModel               = "default-model"
		defaultRole                = "default-role"
		defaultApiKey              = "default-api-key"
		defaultThread              = "default-thread"
		defaultCompletionsPath     = "default-completions-path"
		defaultModelsPath          = "default-models-path"
		defaultAuthHeader          = "default-auth-header"
		defaultAuthTokenPrefix     = "default-auth-token-prefix"
		defaultOmitHistory         = false
		defaultAutoCreateNewThread = false
		defaultTrackTokenUsage     = false
		defaultDebug               = false
		defaultSkipTLSVerify       = false
		defaultTemperature         = 1.1
		defaultTopP                = 2.2
		defaultFrequencyPenalty    = 3.3
		defaultPresencePenalty     = 4.4
		defaultCommandPrompt       = "default-command-prompt"
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
			Name:                defaultName,
			APIKey:              defaultApiKey,
			Model:               defaultModel,
			MaxTokens:           defaultMaxTokens,
			ContextWindow:       defaultContextWindow,
			URL:                 defaultURL,
			CompletionsPath:     defaultCompletionsPath,
			ModelsPath:          defaultModelsPath,
			AuthHeader:          defaultAuthHeader,
			AuthTokenPrefix:     defaultAuthTokenPrefix,
			OmitHistory:         defaultOmitHistory,
			Role:                defaultRole,
			Thread:              defaultThread,
			Temperature:         defaultTemperature,
			TopP:                defaultTopP,
			FrequencyPenalty:    defaultFrequencyPenalty,
			PresencePenalty:     defaultPresencePenalty,
			CommandPrompt:       defaultCommandPrompt,
			AutoCreateNewThread: defaultAutoCreateNewThread,
			TrackTokenUsage:     defaultTrackTokenUsage,
			Debug:               defaultDebug,
			SkipTLSVerify:       defaultSkipTLSVerify,
		}

		envPrefix = strings.ToUpper(defaultConfig.Name) + "_"

		unsetEnvironmentVariables(envPrefix)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	it("should use default values when no environment variables or user config are provided", func() {
		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
		mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("no user config")).Times(1)

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
		Expect(subject.Config.ContextWindow).To(Equal(defaultContextWindow))
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
		Expect(subject.Config.CommandPrompt).To(Equal(defaultCommandPrompt))
		Expect(subject.Config.AutoCreateNewThread).To(Equal(defaultAutoCreateNewThread))
		Expect(subject.Config.TrackTokenUsage).To(Equal(defaultTrackTokenUsage))
		Expect(subject.Config.Debug).To(Equal(defaultDebug))
		Expect(subject.Config.SkipTLSVerify).To(Equal(defaultSkipTLSVerify))
	})

	it("should prioritize user-provided config over defaults", func() {
		userConfig := types.Config{
			Model:               "user-model",
			MaxTokens:           20,
			ContextWindow:       30,
			URL:                 "user-url",
			CompletionsPath:     "user-completions-path",
			ModelsPath:          "user-models-path",
			AuthHeader:          "user-auth-header",
			AuthTokenPrefix:     "user-auth-token-prefix",
			OmitHistory:         true,
			Role:                "user-role",
			Thread:              "user-thread",
			Temperature:         2.5,
			TopP:                3.5,
			FrequencyPenalty:    4.5,
			PresencePenalty:     5.5,
			CommandPrompt:       "user-command-prompt",
			AutoCreateNewThread: true,
			TrackTokenUsage:     true,
			Debug:               true,
			SkipTLSVerify:       true,
		}

		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
		mockConfigStore.EXPECT().Read().Return(userConfig, nil).Times(1)

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.Model).To(Equal("user-model"))
		Expect(subject.Config.MaxTokens).To(Equal(20))
		Expect(subject.Config.ContextWindow).To(Equal(30))
		Expect(subject.Config.URL).To(Equal("user-url"))
		Expect(subject.Config.CompletionsPath).To(Equal("user-completions-path"))
		Expect(subject.Config.ModelsPath).To(Equal("user-models-path"))
		Expect(subject.Config.AuthHeader).To(Equal("user-auth-header"))
		Expect(subject.Config.AuthTokenPrefix).To(Equal("user-auth-token-prefix"))
		Expect(subject.Config.OmitHistory).To(BeTrue())
		Expect(subject.Config.AutoCreateNewThread).To(BeTrue())
		Expect(subject.Config.TrackTokenUsage).To(BeTrue())
		Expect(subject.Config.Debug).To(BeTrue())
		Expect(subject.Config.SkipTLSVerify).To(BeTrue())
		Expect(subject.Config.Role).To(Equal("user-role"))
		Expect(subject.Config.Thread).To(Equal("user-thread"))
		Expect(subject.Config.Temperature).To(Equal(2.5))
		Expect(subject.Config.TopP).To(Equal(3.5))
		Expect(subject.Config.FrequencyPenalty).To(Equal(4.5))
		Expect(subject.Config.PresencePenalty).To(Equal(5.5))
		Expect(subject.Config.CommandPrompt).To(Equal("user-command-prompt"))
	})

	it("should prioritize environment variables over default config", func() {
		os.Setenv(envPrefix+"API_KEY", "env-api-key")
		os.Setenv(envPrefix+"MODEL", "env-model")
		os.Setenv(envPrefix+"MAX_TOKENS", "15")
		os.Setenv(envPrefix+"CONTEXT_WINDOW", "25")
		os.Setenv(envPrefix+"URL", "env-url")
		os.Setenv(envPrefix+"COMPLETIONS_PATH", "env-completions-path")
		os.Setenv(envPrefix+"MODELS_PATH", "env-models-path")
		os.Setenv(envPrefix+"AUTH_HEADER", "env-auth-header")
		os.Setenv(envPrefix+"AUTH_TOKEN_PREFIX", "env-auth-token-prefix")
		os.Setenv(envPrefix+"OMIT_HISTORY", "true")
		os.Setenv(envPrefix+"AUTO_CREATE_NEW_THREAD", "true")
		os.Setenv(envPrefix+"TRACK_TOKEN_USAGE", "true")
		os.Setenv(envPrefix+"DEBUG", "true")
		os.Setenv(envPrefix+"SKIP_TLS_VERIFY", "true")
		os.Setenv(envPrefix+"ROLE", "env-role")
		os.Setenv(envPrefix+"THREAD", "env-thread")
		os.Setenv(envPrefix+"TEMPERATURE", "2.2")
		os.Setenv(envPrefix+"TOP_P", "3.3")
		os.Setenv(envPrefix+"FREQUENCY_PENALTY", "4.4")
		os.Setenv(envPrefix+"PRESENCE_PENALTY", "5.5")
		os.Setenv(envPrefix+"COMMAND_PROMPT", "env-command-prompt")

		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
		mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("config error")).Times(1)

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.APIKey).To(Equal("env-api-key"))
		Expect(subject.Config.Model).To(Equal("env-model"))
		Expect(subject.Config.MaxTokens).To(Equal(15))
		Expect(subject.Config.ContextWindow).To(Equal(25))
		Expect(subject.Config.URL).To(Equal("env-url"))
		Expect(subject.Config.CompletionsPath).To(Equal("env-completions-path"))
		Expect(subject.Config.ModelsPath).To(Equal("env-models-path"))
		Expect(subject.Config.AuthHeader).To(Equal("env-auth-header"))
		Expect(subject.Config.AuthTokenPrefix).To(Equal("env-auth-token-prefix"))
		Expect(subject.Config.OmitHistory).To(BeTrue())
		Expect(subject.Config.AutoCreateNewThread).To(BeTrue())
		Expect(subject.Config.TrackTokenUsage).To(BeTrue())
		Expect(subject.Config.Debug).To(BeTrue())
		Expect(subject.Config.SkipTLSVerify).To(BeTrue())
		Expect(subject.Config.Role).To(Equal("env-role"))
		Expect(subject.Config.Thread).To(Equal("env-thread"))
		Expect(subject.Config.Temperature).To(Equal(2.2))
		Expect(subject.Config.TopP).To(Equal(3.3))
		Expect(subject.Config.FrequencyPenalty).To(Equal(4.4))
		Expect(subject.Config.PresencePenalty).To(Equal(5.5))
		Expect(subject.Config.CommandPrompt).To(Equal("env-command-prompt"))
	})

	it("should prioritize environment variables over user-provided config", func() {
		os.Setenv(envPrefix+"API_KEY", "env-api-key")
		os.Setenv(envPrefix+"MODEL", "env-model")
		os.Setenv(envPrefix+"MAX_TOKENS", "15")
		os.Setenv(envPrefix+"CONTEXT_WINDOW", "25")
		os.Setenv(envPrefix+"URL", "env-url")
		os.Setenv(envPrefix+"COMPLETIONS_PATH", "env-completions-path")
		os.Setenv(envPrefix+"MODELS_PATH", "env-models-path")
		os.Setenv(envPrefix+"AUTH_HEADER", "env-auth-header")
		os.Setenv(envPrefix+"AUTH_TOKEN_PREFIX", "env-auth-token-prefix")
		os.Setenv(envPrefix+"OMIT_HISTORY", "true")
		os.Setenv(envPrefix+"AUTO_CREATE_NEW_THREAD", "true")
		os.Setenv(envPrefix+"TRACK_TOKEN_USAGE", "true")
		os.Setenv(envPrefix+"SKIP_TLS_VERIFY", "true")
		os.Setenv(envPrefix+"DEBUG", "false")
		os.Setenv(envPrefix+"ROLE", "env-role")
		os.Setenv(envPrefix+"THREAD", "env-thread")
		os.Setenv(envPrefix+"TEMPERATURE", "2.2")
		os.Setenv(envPrefix+"TOP_P", "3.3")
		os.Setenv(envPrefix+"FREQUENCY_PENALTY", "4.4")
		os.Setenv(envPrefix+"PRESENCE_PENALTY", "5.5")
		os.Setenv(envPrefix+"COMMAND_PROMPT", "env-command-prompt")

		userConfig := types.Config{
			APIKey:              "user-api-key",
			Model:               "user-model",
			MaxTokens:           20,
			ContextWindow:       30,
			URL:                 "user-url",
			CompletionsPath:     "user-completions-path",
			ModelsPath:          "user-models-path",
			AuthHeader:          "user-auth-header",
			AuthTokenPrefix:     "user-auth-token-prefix",
			OmitHistory:         false,
			AutoCreateNewThread: false,
			TrackTokenUsage:     false,
			SkipTLSVerify:       false,
			Debug:               true,
			Role:                "user-role",
			Thread:              "user-thread",
			Temperature:         1.5,
			TopP:                2.5,
			FrequencyPenalty:    3.5,
			PresencePenalty:     4.5,
			CommandPrompt:       "user-command-prompt",
		}

		mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
		mockConfigStore.EXPECT().Read().Return(userConfig, nil).Times(1)

		subject := configmanager.New(mockConfigStore).WithEnvironment()

		Expect(subject.Config.APIKey).To(Equal("env-api-key"))
		Expect(subject.Config.Model).To(Equal("env-model"))
		Expect(subject.Config.MaxTokens).To(Equal(15))
		Expect(subject.Config.ContextWindow).To(Equal(25))
		Expect(subject.Config.URL).To(Equal("env-url"))
		Expect(subject.Config.CompletionsPath).To(Equal("env-completions-path"))
		Expect(subject.Config.ModelsPath).To(Equal("env-models-path"))
		Expect(subject.Config.AuthHeader).To(Equal("env-auth-header"))
		Expect(subject.Config.AuthTokenPrefix).To(Equal("env-auth-token-prefix"))
		Expect(subject.Config.OmitHistory).To(BeTrue())
		Expect(subject.Config.AutoCreateNewThread).To(BeTrue())
		Expect(subject.Config.TrackTokenUsage).To(BeTrue())
		Expect(subject.Config.SkipTLSVerify).To(BeTrue())
		Expect(subject.Config.Debug).To(BeFalse())
		Expect(subject.Config.Role).To(Equal("env-role"))
		Expect(subject.Config.Thread).To(Equal("env-thread"))
		Expect(subject.Config.Temperature).To(Equal(2.2))
		Expect(subject.Config.TopP).To(Equal(3.3))
		Expect(subject.Config.FrequencyPenalty).To(Equal(4.4))
		Expect(subject.Config.PresencePenalty).To(Equal(5.5))
		Expect(subject.Config.CommandPrompt).To(Equal("env-command-prompt"))
	})

	it("should write the max tokens as expected", func() {
		maxTokens := 123
		performWriteTest(mockConfigStore, defaultConfig, maxTokens, "MaxTokens", func() {
			subject := configmanager.New(mockConfigStore).WithEnvironment()
			subject.WriteMaxTokens(maxTokens)
		})
	})

	it("should write the context window as expected", func() {
		window := 12345
		performWriteTest(mockConfigStore, defaultConfig, window, "ContextWindow", func() {
			subject := configmanager.New(mockConfigStore).WithEnvironment()
			subject.WriteContextWindow(window)
		})
	})

	it("should write the model as expected", func() {
		model := "user-model"
		performWriteTest(mockConfigStore, defaultConfig, model, "Model", func() {
			subject := configmanager.New(mockConfigStore).WithEnvironment()
			subject.WriteModel(model)
		})
	})

	it("should write the thread as expected", func() {
		thread := "user-thread"
		performWriteTest(mockConfigStore, defaultConfig, thread, "Thread", func() {
			subject := configmanager.New(mockConfigStore).WithEnvironment()
			subject.WriteThread(thread)
		})
	})

	when("DeleteThread()", func() {
		var subject *configmanager.ConfigManager

		threadName := "non-active-thread"

		it.Before(func() {
			userConfig := types.Config{Thread: threadName}

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(userConfig, nil).Times(1)

			subject = configmanager.New(mockConfigStore).WithEnvironment()
		})

		it("propagates the error from the config store", func() {
			expectedMsg := "expected-error"

			mockConfigStore.EXPECT().Delete(threadName).Return(errors.New(expectedMsg)).Times(1)

			err := subject.DeleteThread(threadName)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(expectedMsg))
		})
		it("completes successfully the config store throws no error", func() {
			mockConfigStore.EXPECT().Delete(threadName).Return(nil).Times(1)

			err := subject.DeleteThread(threadName)

			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("ListThreads()", func() {
		activeThread := "active-thread"

		it.Before(func() {
			userConfig := types.Config{Thread: activeThread}

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(userConfig, nil).Times(1)
		})

		it("throws an error when the List call fails", func() {
			subject := configmanager.New(mockConfigStore).WithEnvironment()

			errorInstance := errors.New("an error occurred")
			mockConfigStore.EXPECT().List().Return(nil, errorInstance).Times(1)

			_, err := subject.ListThreads()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(errorInstance))
		})

		it("returns the expected threads", func() {
			subject := configmanager.New(mockConfigStore).WithEnvironment()

			threads := []string{"thread1.json", "thread2.json", activeThread + ".json"}
			mockConfigStore.EXPECT().List().Return(threads, nil).Times(1)

			result, err := subject.ListThreads()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(3))
			Expect(result[0]).NotTo(ContainSubstring("current"))
			Expect(result[0]).NotTo(ContainSubstring("json"))
			Expect(result[1]).NotTo(ContainSubstring("current"))
			Expect(result[1]).NotTo(ContainSubstring("json"))
			Expect(result[2]).To(ContainSubstring("current"))
			Expect(result[2]).NotTo(ContainSubstring("json"))
		})
	})
}

func performWriteTest(mockConfigStore *MockConfigStore, defaultConfig types.Config, expectedValue interface{}, fieldName string, action func()) {
	mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
	mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("no user config")).Times(1)

	setValue(&defaultConfig, fieldName, expectedValue)
	mockConfigStore.EXPECT().Write(defaultConfig).Return(nil).Times(1)

	action()
}

func setValue(config *types.Config, fieldName string, value interface{}) {
	// Use reflection to set value of the field by name
	v := reflect.ValueOf(config).Elem() // Get the value that the pointer 'config' points to
	fieldVal := v.FieldByName(fieldName)

	if fieldVal.IsValid() && fieldVal.CanSet() {
		val := reflect.ValueOf(value)
		if val.Type().AssignableTo(fieldVal.Type()) {
			fieldVal.Set(val)
		}
	}
}

func unsetEnvironmentVariables(envPrefix string) {
	variables := []string{
		"API_KEY",
		"MODEL",
		"MAX_TOKENS",
		"CONTEXT_WINDOW",
		"URL",
		"COMPLETIONS_PATH",
		"MODELS_PATH",
		"AUTH_HEADER",
		"AUTH_TOKEN_PREFIX",
		"OMIT_HISTORY",
		"ROLE",
		"THREAD",
		"TEMPERATURE",
		"TOP_P",
		"FREQUENCY_PENALTY",
		"PRESENCE_PENALTY",
		"COMMAND_PROMPT",
		"AUTO_CREATE_NEW_THREAD",
		"TRACK_TOKEN_USAGE",
		"DEBUG",
		"SKIP_TLS_VERIFY",
	}

	for _, variable := range variables {
		Expect(os.Unsetenv(envPrefix + variable)).To(Succeed())
	}
}
