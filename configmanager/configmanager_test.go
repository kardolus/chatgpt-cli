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
		defaultMaxTokens       = 10
		defaultName            = "default-name"
		defaultURL             = "default-url"
		defaultModel           = "default-model"
		defaultApiKey          = "default-api-key"
		defaultCompletionsPath = "default-completions-path"
		defaultModelsPath      = "default-models-path"
		defaultOmitHistory     = false
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
			Name:            defaultName,
			APIKey:          defaultApiKey,
			Model:           defaultModel,
			MaxTokens:       defaultMaxTokens,
			URL:             defaultURL,
			CompletionsPath: defaultCompletionsPath,
			ModelsPath:      defaultModelsPath,
			OmitHistory:     defaultOmitHistory,
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
			Expect(subject.Config.OmitHistory).To(Equal(omitHistory))
		})
		it("gives precedence to the OMIT_HISTORY environment variable", func() {
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
			Expect(subject.Config.OmitHistory).To(Equal(environmentValue))
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
			Expect(subject.Config.OmitHistory).To(Equal(defaultOmitHistory))
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
}
