package client_test

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=callermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/http Caller
//go:generate mockgen -destination=historymocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/history HistoryStore

const (
	envApiKey       = "api-key"
	commandLineMode = false
	interactiveMode = true
)

var (
	mockCtrl         *gomock.Controller
	mockCaller       *MockCaller
	mockHistoryStore *MockHistoryStore
	factory          *clientFactory
	apiKeyEnvVar     string
	config           types.Config
)

func TestUnitClient(t *testing.T) {
	spec.Run(t, "Testing the client package", testClient, spec.Report(report.Terminal{}))
}

func testClient(t *testing.T, when spec.G, it spec.S) {
	const query = "test query"

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockCaller = NewMockCaller(mockCtrl)
		mockHistoryStore = NewMockHistoryStore(mockCtrl)
		config = MockConfig()

		factory = newClientFactory(mockHistoryStore)

		apiKeyEnvVar = strings.ToUpper(config.Name) + "_API_KEY"
		Expect(os.Setenv(apiKeyEnvVar, envApiKey)).To(Succeed())
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("New()", func() {
		it("should set a unique thread slug in interactive mode when AutoCreateNewThread is true", func() {
			var capturedThread string
			mockHistoryStore.EXPECT().SetThread(gomock.Any()).DoAndReturn(func(thread string) {
				capturedThread = thread
			}).Times(1)

			client.New(mockCallerFactory, mockHistoryStore, MockConfig(), interactiveMode)

			Expect(capturedThread).To(HavePrefix(client.InteractiveThreadPrefix)) // Assuming `InteractiveThreadPrefix` is "int_"
			Expect(len(capturedThread)).To(Equal(8))                              // "int_" (4 chars) + 4 random characters
		})
		it("should not overwrite the thread in interactive mode when AutoCreateNewThread is false", func() {
			var capturedThread string
			mockHistoryStore.EXPECT().SetThread(gomock.Any()).DoAndReturn(func(thread string) {
				capturedThread = thread
			}).Times(1)

			cfg := MockConfig()
			cfg.AutoCreateNewThread = false

			client.New(mockCallerFactory, mockHistoryStore, cfg, interactiveMode)

			Expect(capturedThread).To(Equal(config.Thread))
		})
		it("should never overwrite the thread in non-interactive mode", func() {
			var capturedThread string
			mockHistoryStore.EXPECT().SetThread(config.Thread).DoAndReturn(func(thread string) {
				capturedThread = thread
			}).Times(1)

			client.New(mockCallerFactory, mockHistoryStore, MockConfig(), commandLineMode)

			Expect(capturedThread).To(Equal(config.Thread))
		})
	})

	when("Query()", func() {
		var (
			body     []byte
			messages []types.Message
			err      error
		)

		type TestCase struct {
			description     string
			setupPostReturn func() ([]byte, error)
			postError       error
			expectedError   string
		}

		tests := []TestCase{
			{
				description:     "throws an error when the http callout fails",
				setupPostReturn: func() ([]byte, error) { return nil, nil },
				postError:       errors.New("error message"),
				expectedError:   "error message",
			},
			{
				description:     "throws an error when the response is empty",
				setupPostReturn: func() ([]byte, error) { return nil, nil },
				postError:       nil,
				expectedError:   "empty response",
			},
			{
				description: "throws an error when the response is a malformed json",
				setupPostReturn: func() ([]byte, error) {
					malformed := `{"invalid":"json"` // missing closing brace
					return []byte(malformed), nil
				},
				postError:     nil,
				expectedError: "failed to decode response:",
			},
			{
				description: "throws an error when the response is missing Choices",
				setupPostReturn: func() ([]byte, error) {
					response := &types.CompletionsResponse{
						ID:      "id",
						Object:  "object",
						Created: 0,
						Model:   "model",
						Choices: []types.Choice{},
					}

					respBytes, err := json.Marshal(response)
					return respBytes, err
				},
				postError:     nil,
				expectedError: "no responses returned",
			},
		}

		for _, tt := range tests {
			it(tt.description, func() {
				factory.withoutHistory()
				subject := factory.buildClientWithoutConfig()

				messages = createMessages(nil, query)
				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				respBytes, err := tt.setupPostReturn()
				Expect(err).NotTo(HaveOccurred())
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, body, false).Return(respBytes, tt.postError)

				_, _, err = subject.Query(query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(tt.expectedError))
			})
		}

		when("a valid http response is received", func() {
			testValidHTTPResponse := func(subject *client.Client, history []types.Message, expectedBody []byte, omitHistory bool) {
				const (
					answer = "content"
					tokens = 789
				)

				choice := types.Choice{
					Message: types.Message{
						Role:    client.AssistantRole,
						Content: answer,
					},
					FinishReason: "",
					Index:        0,
				}
				response := &types.CompletionsResponse{
					ID:      "id",
					Object:  "object",
					Created: 0,
					Model:   subject.Config.Model,
					Choices: []types.Choice{choice},
					Usage: types.Usage{
						PromptTokens:     123,
						CompletionTokens: 456,
						TotalTokens:      tokens,
					},
				}

				respBytes, err := json.Marshal(response)
				Expect(err).NotTo(HaveOccurred())
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).Return(respBytes, nil)

				var request types.CompletionsRequest
				err = json.Unmarshal(expectedBody, &request)
				Expect(err).NotTo(HaveOccurred())

				if !omitHistory {
					mockHistoryStore.EXPECT().Write(append(request.Messages, types.Message{
						Role:    client.AssistantRole,
						Content: answer,
					}))
				}

				result, usage, err := subject.Query(query)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(answer))
				Expect(usage).To(Equal(tokens))
			}
			it("returns the expected result for a non-empty history", func() {
				history := []types.Message{
					{
						Role:    client.SystemRole,
						Content: config.Role,
					},
					{
						Role:    client.UserRole,
						Content: "question 1",
					},
					{
						Role:    client.AssistantRole,
						Content: "answer 1",
					},
				}
				messages = createMessages(history, query)
				factory.withHistory(history)
				subject := factory.buildClientWithoutConfig()

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, history, body, false)
			})
			it("ignores history when configured to do so", func() {
				mockHistoryStore.EXPECT().SetThread(config.Thread).Times(1)

				config := MockConfig()
				config.OmitHistory = true

				subject := client.New(mockCallerFactory, mockHistoryStore, config, commandLineMode)
				Expect(err).NotTo(HaveOccurred())

				// Read and Write are never called on the history store
				mockHistoryStore.EXPECT().Read().Times(0)
				mockHistoryStore.EXPECT().Write(gomock.Any()).Times(0)

				messages = createMessages(nil, query)

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, nil, body, true)
			})
			it("truncates the history as expected", func() {
				history := []types.Message{
					{
						Role:    client.SystemRole,
						Content: config.Role,
					},
					{
						Role:    client.UserRole,
						Content: "question 1",
					},
					{
						Role:    client.AssistantRole,
						Content: "answer 1",
					},
					{
						Role:    client.UserRole,
						Content: "question 2",
					},
					{
						Role:    client.AssistantRole,
						Content: "answer 2",
					},
					{
						Role:    client.UserRole,
						Content: "question 3",
					},
					{
						Role:    client.AssistantRole,
						Content: "answer 3",
					},
				}

				messages = createMessages(history, query)

				factory.withHistory(history)
				subject := factory.buildClientWithoutConfig()

				// messages get truncated. Index 1+2 are cut out
				messages = append(messages[:1], messages[3:]...)

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, history, body, false)
			})
		})
	})
	when("Stream()", func() {
		var (
			body     []byte
			messages []types.Message
			err      error
		)

		it("throws an error when the http callout fails", func() {
			factory.withoutHistory()
			subject := factory.buildClientWithoutConfig()

			messages = createMessages(nil, query)
			body, err = createBody(messages, true)
			Expect(err).NotTo(HaveOccurred())

			errorMsg := "error message"
			mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, body, true).Return(nil, errors.New(errorMsg))

			err := subject.Stream(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		when("a valid http response is received", func() {
			const answer = "answer"

			testValidHTTPResponse := func(subject *client.Client, history []types.Message, expectedBody []byte) {
				messages = createMessages(nil, query)
				body, err = createBody(messages, true)
				Expect(err).NotTo(HaveOccurred())

				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, true).Return([]byte(answer), nil)

				messages = createMessages(history, query)

				mockHistoryStore.EXPECT().Write(append(messages, types.Message{
					Role:    client.AssistantRole,
					Content: answer,
				}))

				err := subject.Stream(query)
				Expect(err).NotTo(HaveOccurred())
			}

			it("returns the expected result for an empty history", func() {
				factory.withHistory(nil)
				subject := factory.buildClientWithoutConfig()

				messages = createMessages(nil, query)
				body, err = createBody(messages, true)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, nil, body)
			})
			it("returns the expected result for a non-empty history", func() {
				history := []types.Message{
					{
						Role:    client.SystemRole,
						Content: config.Role,
					},
					{
						Role:    client.UserRole,
						Content: "question 1",
					},
					{
						Role:    client.AssistantRole,
						Content: "answer 1",
					},
				}
				factory.withHistory(history)
				subject := factory.buildClientWithoutConfig()

				messages = createMessages(history, query)
				body, err = createBody(messages, true)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, history, body)
			})
		})
	})
	when("ListModels()", func() {
		it("throws an error when the http callout fails", func() {
			subject := factory.buildClientWithoutConfig()

			errorMsg := "error message"
			mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).Return(nil, errors.New(errorMsg))

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		it("throws an error when the response is empty", func() {
			subject := factory.buildClientWithoutConfig()

			mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).Return(nil, nil)

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("empty response"))
		})
		it("throws an error when the response is a malformed json", func() {
			subject := factory.buildClientWithoutConfig()

			malformed := `{"invalid":"json"` // missing closing brace
			mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).Return([]byte(malformed), nil)

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(HavePrefix("failed to decode response:"))
		})
		it("filters gpt models as expected", func() {
			subject := factory.buildClientWithoutConfig()

			response, err := utils.FileToBytes("models.json")
			Expect(err).NotTo(HaveOccurred())

			mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).Return(response, nil)

			result, err := subject.ListModels()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())
			Expect(result).To(HaveLen(2))
			Expect(result[0]).To(Equal("* gpt-3.5-turbo (current)"))
			Expect(result[1]).To(Equal("- gpt-3.5-turbo-0301"))
		})
	})
	when("ProvideContext()", func() {
		it("updates the history with the provided context", func() {
			subject := factory.buildClientWithoutConfig()

			context := "This is a story about a dog named Kya. Kya loves to play fetch and swim in the lake."
			mockHistoryStore.EXPECT().Read().Return(nil, nil).Times(1)

			subject.ProvideContext(context)

			Expect(len(subject.History)).To(Equal(2)) // The system message and the provided context

			systemMessage := subject.History[0]
			Expect(systemMessage.Role).To(Equal(client.SystemRole))
			Expect(systemMessage.Content).To(Equal(config.Role))

			contextMessage := subject.History[1]
			Expect(contextMessage.Role).To(Equal(client.UserRole))
			Expect(contextMessage.Content).To(Equal(context))
		})
	})
}

func createBody(messages []types.Message, stream bool) ([]byte, error) {
	req := types.CompletionsRequest{
		Model:            config.Model,
		Messages:         messages,
		Stream:           stream,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		FrequencyPenalty: config.FrequencyPenalty,
		MaxTokens:        config.MaxTokens,
		PresencePenalty:  config.PresencePenalty,
	}

	return json.Marshal(req)
}

func createMessages(history []types.Message, query string) []types.Message {
	var messages []types.Message

	if len(history) == 0 {
		messages = append(messages, types.Message{
			Role:    client.SystemRole,
			Content: config.Role,
		})
	} else {
		messages = history
	}

	messages = append(messages, types.Message{
		Role:    client.UserRole,
		Content: query,
	})

	return messages
}

type clientFactory struct {
	mockHistoryStore *MockHistoryStore
}

func newClientFactory(mhs *MockHistoryStore) *clientFactory {
	return &clientFactory{
		mockHistoryStore: mhs,
	}
}

func (f *clientFactory) buildClientWithoutConfig() *client.Client {
	f.mockHistoryStore.EXPECT().SetThread(config.Thread).Times(1)

	c := client.New(mockCallerFactory, f.mockHistoryStore, MockConfig(), commandLineMode)

	return c.WithContextWindow(config.ContextWindow)
}

func (f *clientFactory) buildClientWithConfig(config types.Config) *client.Client {
	c := client.New(mockCallerFactory, f.mockHistoryStore, config, commandLineMode)
	return c.WithContextWindow(config.ContextWindow)
}

func (f *clientFactory) withoutHistory() {
	f.mockHistoryStore.EXPECT().Read().Return(nil, nil).Times(1)
}

func (f *clientFactory) withHistory(history []types.Message) {
	f.mockHistoryStore.EXPECT().Read().Return(history, nil).Times(1)
}

func mockCallerFactory(cfg types.Config) http.Caller {
	return mockCaller
}

func MockConfig() types.Config {
	return types.Config{
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
		Debug:               false,
	}
}
