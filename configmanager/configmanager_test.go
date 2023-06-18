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
	"testing"
)

//go:generate mockgen -destination=configmocks_test.go -package=configmanager_test github.com/kardolus/chatgpt-cli/config ConfigStore

func TestUnitConfigManager(t *testing.T) {
	spec.Run(t, "Testing the Config Manager", testConfig, spec.Report(report.Terminal{}))
}

func testConfig(t *testing.T, when spec.G, it spec.S) {
	const (
		defaultMaxTokens       = 10
		defaultURL             = "default-url"
		defaultModel           = "default-model"
		defaultCompletionsPath = "default-completions-path"
		defaultModelsPath      = "default-models-path"
	)

	var (
		mockCtrl        *gomock.Controller
		mockConfigStore *MockConfigStore
		defaultConfig   types.Config
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockConfigStore = NewMockConfigStore(mockCtrl)

		defaultConfig = types.Config{
			Model:           defaultModel,
			MaxTokens:       defaultMaxTokens,
			URL:             defaultURL,
			CompletionsPath: defaultCompletionsPath,
			ModelsPath:      defaultModelsPath,
		}
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("Constructing a new ConfigManager", func() {
		it("applies the default configuration when user config is missing", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New("no such file")).Times(1)

			subject := configmanager.New(mockConfigStore)

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
		})
		it("gives precedence to the user provided model", func() {
			userModel := "the-model"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{Model: userModel}, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			Expect(subject.Config.Model).To(Equal(userModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
		})
		it("gives precedence to the user provided max-tokens", func() {
			userMaxTokens := 42

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{MaxTokens: userMaxTokens}, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(userMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
		})
		it("gives precedence to the user provided URL", func() {
			userURL := "the-user-url"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{URL: userURL}, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(userURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
		})
		it("gives precedence to the user provided completions-path", func() {
			completionsPath := "the-completions-path"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{CompletionsPath: completionsPath}, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(completionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(defaultModelsPath))
		})
		it("gives precedence to the user provided models-path", func() {
			modelsPath := "the-models-path"

			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(types.Config{ModelsPath: modelsPath}, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			Expect(subject.Config.Model).To(Equal(defaultModel))
			Expect(subject.Config.MaxTokens).To(Equal(defaultMaxTokens))
			Expect(subject.Config.URL).To(Equal(defaultURL))
			Expect(subject.Config.CompletionsPath).To(Equal(defaultCompletionsPath))
			Expect(subject.Config.ModelsPath).To(Equal(modelsPath))
		})
	})

	when("ShowConfig()", func() {
		it("displays the expected configuration", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(defaultConfig, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			result, err := subject.ShowConfig()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ContainSubstring(defaultModel))
			Expect(result).To(ContainSubstring(defaultURL))
			Expect(result).To(ContainSubstring(defaultCompletionsPath))
			Expect(result).To(ContainSubstring(defaultModelsPath))
			Expect(result).To(ContainSubstring(fmt.Sprintf("%d", defaultMaxTokens)))
		})
	})

	when("WriteModel()", func() {
		it("writes the expected config file", func() {
			mockConfigStore.EXPECT().ReadDefaults().Return(defaultConfig).Times(1)
			mockConfigStore.EXPECT().Read().Return(defaultConfig, nil).Times(1)

			subject := configmanager.New(mockConfigStore)

			modelName := "the-model"
			subject.Config.Model = modelName

			mockConfigStore.EXPECT().Write(subject.Config).Times(1)
			Expect(subject.WriteModel(modelName)).To(Succeed())
		})
	})
}
