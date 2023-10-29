package configmanager_test

import (
	"errors"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/types"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"os"
	"strconv"
	"strings"
	"testing"
)

//go:generate mockgen -destination=configmocks_test.go -package=configmanager_test github.com/kardolus/chatgpt-cli/config ConfigStore

func TestUnitConfigManager(t *testing.T) {
	spec.Run(t, "Testing the Config Manager", testConfig, spec.Report(report.Terminal{}))
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
			OmitHistory:      defaultOmitHistory,
			Role:             defaultRole,
			Thread:           defaultThread,
			Temperature:      defaultTemperature,
			TopP:             defaultTopP,
			FrequencyPenalty: defaultFrequencyPenalty,
			PresencePenalty:  defaultPresencePenalty,
		}

		envPrefix = strings.ToUpper(defaultConfig.Name) + "_"
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("Constructing a new ConfigManager", func() {
		it.Before(func() {
			cleanEnv(envPrefix)
		})
		it.After(func() {
			cleanEnv(envPrefix)
		})

		it("applies the default configuration when user config is missing", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("no such file")).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided model", func() {
			userModel := "the-model"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Model: userModel}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Model).To(Equal(userModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided name", func() {
			userName := "the-name"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Name: userName}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.Name).To(Equal(userName))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided max-tokens", func() {
			userMaxTokens := 42

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{MaxTokens: userMaxTokens}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(userMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided URL", func() {
			userURL := "the-user-url"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{URL: userURL}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(userURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
		})
		it("gives precedence to the user provided completions-path", func() {
			completionsPath := "the-completions-path"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{CompletionsPath: completionsPath}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(completionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided models-path", func() {
			modelsPath := "the-models-path"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{ModelsPath: modelsPath}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(modelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided api-key", func() {
			apiKey := "new-api-key"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{APIKey: apiKey}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(apiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided omit-history", func() {
			omitHistory := true

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{OmitHistory: omitHistory}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.OmitHistory).To(Equal(omitHistory))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided thread", func() {
			userThread := "user-thread"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Thread: userThread}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(userThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided role", func() {
			userRole := "user-role"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Role: userRole}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Role).To(Equal(userRole))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided temperature", func() {
			userTemperature := 100.1

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Temperature: userTemperature}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Temperature).To(Equal(userTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided top_p", func() {
			userTopP := 200.2

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{TopP: userTopP}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(userTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided frequency_penalty", func() {
			userFrequencyPenalty := 300.3

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{FrequencyPenalty: userFrequencyPenalty}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(userFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the user provided presence_penalty", func() {
			userPresencePenalty := 400.4

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{PresencePenalty: userPresencePenalty}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(userPresencePenalty))
		})
		it("gives precedence to the OMIT_HISTORY environment variable when environment is true", func() {
			var (
				environmentValue = true
				configValue      = false
			)

			Expect(os.Setenv(envPrefix+"OMIT_HISTORY", strconv.FormatBool(environmentValue))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{OmitHistory: configValue}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.OmitHistory).To(Equal(environmentValue))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the OMIT_HISTORY environment variable when environment is false", func() {
			var (
				environmentValue = false
				configValue      = true
			)

			Expect(os.Setenv(envPrefix+"OMIT_HISTORY", strconv.FormatBool(environmentValue))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{OmitHistory: configValue}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.OmitHistory).To(Equal(environmentValue))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the THREAD environment variable", func() {
			var (
				environmentValue = "env-thread"
				configValue      = "conf-thread"
			)

			Expect(os.Setenv(envPrefix+"THREAD", environmentValue)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Thread: configValue}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(environmentValue))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the ROLE environment variable", func() {
			var (
				environmentValue = "env-role"
				configValue      = "conf-role"
			)

			Expect(os.Setenv(envPrefix+"ROLE", environmentValue)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Role: configValue}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Role).To(Equal(environmentValue))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the API_KEY environment variable", func() {
			var (
				environmentKey = "environment-api-key"
				configKey      = "config-api-key"
			)

			Expect(os.Setenv(envPrefix+"API_KEY", environmentKey)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{APIKey: configKey}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(environmentKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the MODEL environment variable", func() {
			var (
				envModel  = "environment-model"
				confModel = "config-model"
			)

			Expect(os.Setenv(envPrefix+"MODEL", envModel)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Model: confModel}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(envModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the MAX_TOKENS environment variable", func() {
			var (
				envMaxTokens  = 42
				confMaxTokens = 4242
			)

			Expect(os.Setenv(envPrefix+"MAX_TOKENS", strconv.Itoa(envMaxTokens))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{MaxTokens: confMaxTokens}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(envMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the URL environment variable", func() {
			var (
				envURL  = "environment-url"
				confURL = "config-url"
			)

			Expect(os.Setenv(envPrefix+"URL", envURL)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{URL: confURL}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(envURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the COMPLETIONS_PATH environment variable", func() {
			var (
				envCompletionsPath  = "environment-completions-path"
				confCompletionsPath = "config-completions-path"
			)

			Expect(os.Setenv(envPrefix+"COMPLETIONS_PATH", envCompletionsPath)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{CompletionsPath: confCompletionsPath}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(envCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the MODELS_PATH environment variable", func() {
			var (
				envModelsPath  = "environment-models-path"
				confModelsPath = "config-models-path"
			)

			Expect(os.Setenv(envPrefix+"MODELS_PATH", envModelsPath)).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{ModelsPath: confModelsPath}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(envModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the TEMPERATURE environment variable", func() {
			var (
				envTemperature  = 5.5
				confTemperature = 6.6
			)

			Expect(os.Setenv(envPrefix+"TEMPERATURE", fmt.Sprintf("%f", envTemperature))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Temperature: confTemperature}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(envTemperature))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the TOP_P environment variable", func() {
			var (
				envTopP  = 7.7
				confTopP = 8.8
			)

			Expect(os.Setenv(envPrefix+"TOP_P", fmt.Sprintf("%f", envTopP))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{TopP: confTopP}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.TopP).To(Equal(envTopP))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the FREQUENCY_PENALTY environment variable", func() {
			var (
				envFrequencyPenalty  = 5.5
				confFrequencyPenalty = 6.6
			)

			Expect(os.Setenv(envPrefix+"FREQUENCY_PENALTY", fmt.Sprintf("%f", envFrequencyPenalty))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{FrequencyPenalty: confFrequencyPenalty}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.FrequencyPenalty).To(Equal(envFrequencyPenalty))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
			Expect(subject.Config.PresencePenalty).To(Equal(defaultPresencePenalty))
		})
		it("gives precedence to the PRESENCE_PENALTY environment variable", func() {
			var (
				envPresencePenalty  = 5.5
				confPresencePenalty = 6.6
			)

			Expect(os.Setenv(envPrefix+"PRESENCE_PENALTY", fmt.Sprintf("%f", envPresencePenalty))).To(Succeed())

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{PresencePenalty: confPresencePenalty}, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			Expect(subject.Config.Name).To(Equal(defaultName))
			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
			Expect(subject.Config.APIKey).To(Equal(defaultApiKey))
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
			Expect(subject.Config.Role).To(Equal(defaultRole))
			Expect(subject.Config.Thread).To(Equal(defaultThread))
			Expect(subject.Config.Temperature).To(Equal(defaultTemperature))
			Expect(subject.Config.FrequencyPenalty).To(Equal(defaultFrequencyPenalty))
			Expect(subject.Config.PresencePenalty).To(Equal(envPresencePenalty))
			Expect(subject.Config.TopP).To(Equal(defaultTopP))
		})
	})

	when("ShowConfig()", func() {
		it("displays the expected configuration", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(defaultConfig, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			result, err := subject.ShowConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring(defaultName))
			Expect(result).To(ContainSubstring(defaultApiKey))
			Expect(result).To(ContainSubstring(defaultModel))
			Expect(result).To(ContainSubstring(defaultURL))
			Expect(result).To(ContainSubstring(defaultCompletionsPath))
			Expect(result).To(ContainSubstring(defaultModelsPath))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%d", defaultMaxTokens)))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%t", defaultOmitHistory)))
			Expect(result).To(ContainSubstring(defaultRole))
			Expect(result).To(ContainSubstring(defaultThread))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%.1f", defaultTemperature)))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%.1f", defaultTopP)))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%.1f", defaultFrequencyPenalty)))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%.1f", defaultPresencePenalty)))
		})
	})

	when("WriteModel()", func() {
		it("writes the expected config file", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(defaultConfig, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			modelName := "the-model"
			subject.Config.Model = modelName

			mockConfigStore.EXPECT().Write(subject.Config).Times(1)
			Expect(subject.WriteModel(modelName)).To(Succeed())
		})
	})

	when("WriteMaxTokens()", func() {
		it("writes the expected config file", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(defaultConfig, nil).Times(1)

			subject := configmanager.New(mockConfigStore).WithEnvironment()

			maxTokens := 9879284
			subject.Config.MaxTokens = maxTokens

			mockConfigStore.EXPECT().Write(subject.Config).Times(1)
			Expect(subject.WriteMaxTokens(maxTokens)).To(Succeed())
		})
	})
}

func cleanEnv(envPrefix string) {
	Expect(os.Unsetenv(envPrefix + "API_KEY")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "MODEL")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "MAX_TOKENS")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "URL")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "COMPLETIONS_PATH")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "MODELS_PATH")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "OMIT_HISTORY")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "THREAD")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "ROLE")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "TEMPERATURE")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "TOP_P")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "FREQUENCY_PENALTY")).To(Succeed())
	Expect(os.Unsetenv(envPrefix + "PRESENCE_PENALTY")).To(Succeed())
}
