package configmanager_test

import (
	"errors"
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
	var (
		mockCtrl        *gomock.Controller
		mockConfigStore *MockConfigStore
		subject         *configmanager.ConfigManager
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockConfigStore = NewMockConfigStore(mockCtrl)
		subject = configmanager.New(mockConfigStore)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("ReadModel()", func() {
		it("throws an error when the config file does not exist", func() {
			expectedErrorMsg := "file not found"
			mockConfigStore.EXPECT().Read().Return(types.Config{}, errors.New(expectedErrorMsg)).Times(1)
			_, err := subject.ReadModel()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(expectedErrorMsg))
		})
		it("parses a config file as expected", func() {
			modelName := "the-model"
			mockConfigStore.EXPECT().Read().Return(types.Config{Model: modelName}, nil).Times(1)

			result, err := subject.ReadModel()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(modelName))
		})
	})
	when("WriteModel()", func() {
		it("writes the expected config file", func() {
			modelName := "the-model"
			mockConfigStore.EXPECT().Write(types.Config{Model: modelName}).Times(1)
			Expect(subject.WriteModel(modelName)).To(Succeed())
		})
	})
}
