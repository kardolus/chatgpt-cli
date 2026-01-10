package client_test

import (
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	config2 "github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"os"
	"strings"
	"testing"
)

//go:generate mockgen -destination=callermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/http Caller
//go:generate mockgen -destination=historymocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/history Store
//go:generate mockgen -destination=timermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/client Timer
//go:generate mockgen -destination=readermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/client FileReader
//go:generate mockgen -destination=writermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/client FileWriter
//go:generate mockgen -destination=transportmocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/client MCPTransport

const (
	envApiKey = "api-key"
)

var (
	mockCtrl         *gomock.Controller
	mockCaller       *MockCaller
	mockHistoryStore *MockStore
	mockTimer        *MockTimer
	mockReader       *MockFileReader
	mockWriter       *MockFileWriter
	factory          *clientFactory
	apiKeyEnvVar     string
	config           config2.Config
)

func TestUnitClient(t *testing.T) {
	spec.Run(t, "Testing the client package", testClient, spec.Report(report.Terminal{}))
}

func testClient(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockCaller = NewMockCaller(mockCtrl)
		mockHistoryStore = NewMockStore(mockCtrl)
		mockTimer = NewMockTimer(mockCtrl)
		mockReader = NewMockFileReader(mockCtrl)
		mockWriter = NewMockFileWriter(mockCtrl)
		config = MockConfig()

		factory = newClientFactory(mockHistoryStore)

		apiKeyEnvVar = strings.ToUpper(config.Name) + "_API_KEY"
		Expect(os.Setenv(apiKeyEnvVar, envApiKey)).To(Succeed())
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	testMCP(t, when, it)
	testSessionTransport(t, when, it)
	testSessionTransportNonHTTP(t, when, it)
	testNewMCPTransport(t, when, it)
	testHistory(t, when, it)
	testMedia(t, when, it)
	testLLM(t, when, it)
}

func newClientFactory(mhs *MockStore) *clientFactory {
	return &clientFactory{
		mockHistoryStore: mhs,
	}
}

type clientFactory struct {
	mockHistoryStore *MockStore
}

func (f *clientFactory) buildClientWithoutConfig() *client.Client {
	c := client.New(mockCallerFactory, f.mockHistoryStore, mockTimer, mockReader, mockWriter, MockConfig())

	return c.WithContextWindow(config.ContextWindow)
}

func (f *clientFactory) withoutHistory() {
	f.mockHistoryStore.EXPECT().Read().Return(nil, nil).Times(1)
}

func (f *clientFactory) withHistory(history []history.History) {
	f.mockHistoryStore.EXPECT().Read().Return(history, nil).Times(1)
}

func mockCallerFactory(_ config2.Config) http.Caller {
	return mockCaller
}

func MockConfig() config2.Config {
	return config2.Config{
		Name:                "mock-openai",
		APIKey:              "mock-api-key",
		Model:               "gpt-3.5-turbo",
		MaxTokens:           100,
		ContextWindow:       50,
		Role:                "You are a test assistant.",
		Temperature:         0.7,
		TopP:                0.9,
		FrequencyPenalty:    0.1,
		PresencePenalty:     0.2,
		Thread:              "mock-thread",
		OmitHistory:         false,
		URL:                 "https://api.mock-openai.com",
		CompletionsPath:     "/v1/test/completions",
		ModelsPath:          "/v1/test/models",
		AuthHeader:          "MockAuthorization",
		AuthTokenPrefix:     "MockBearer ",
		CommandPrompt:       "[mock-datetime] [Q%counter] [%usage]",
		OutputPrompt:        "[mock-output]",
		AutoCreateNewThread: true,
		TrackTokenUsage:     true,
		SkipTLSVerify:       false,
		Seed:                1,
		Effort:              "low",
		ResponsesPath:       "/v1/responses",
		Voice:               "mock-voice",
		TranscriptionsPath:  "/v1/test/transcriptions",
		SpeechPath:          "/v1/test/speech",
	}
}
