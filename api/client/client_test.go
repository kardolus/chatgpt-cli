package client_test

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	config2 "github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/test"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=callermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/http Caller
//go:generate mockgen -destination=historymocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/history Store
//go:generate mockgen -destination=timermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/client Timer
//go:generate mockgen -destination=readermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/api/client ImageReader

const (
	envApiKey       = "api-key"
	commandLineMode = false
	interactiveMode = true
)

var (
	mockCtrl         *gomock.Controller
	mockCaller       *MockCaller
	mockHistoryStore *MockStore
	mockTimer        *MockTimer
	mockReader       *MockImageReader
	factory          *clientFactory
	apiKeyEnvVar     string
	config           config2.Config
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
		mockHistoryStore = NewMockStore(mockCtrl)
		mockTimer = NewMockTimer(mockCtrl)
		mockReader = NewMockImageReader(mockCtrl)
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

			client.New(mockCallerFactory, mockHistoryStore, mockTimer, mockReader, MockConfig(), interactiveMode)

			Expect(capturedThread).To(HavePrefix(client.InteractiveThreadPrefix))
			Expect(len(capturedThread)).To(Equal(8)) // "int_" (4 chars) + 4 random characters
		})
		it("should not overwrite the thread in interactive mode when AutoCreateNewThread is false", func() {
			var capturedThread string
			mockHistoryStore.EXPECT().SetThread(gomock.Any()).DoAndReturn(func(thread string) {
				capturedThread = thread
			}).Times(1)

			cfg := MockConfig()
			cfg.AutoCreateNewThread = false

			client.New(mockCallerFactory, mockHistoryStore, mockTimer, mockReader, cfg, interactiveMode)

			Expect(capturedThread).To(Equal(config.Thread))
		})
		it("should never overwrite the thread in non-interactive mode", func() {
			var capturedThread string
			mockHistoryStore.EXPECT().SetThread(config.Thread).DoAndReturn(func(thread string) {
				capturedThread = thread
			}).Times(1)

			client.New(mockCallerFactory, mockHistoryStore, mockTimer, mockReader, MockConfig(), commandLineMode)

			Expect(capturedThread).To(Equal(config.Thread))
		})
	})
	when("Query()", func() {
		var (
			body     []byte
			messages []api.Message
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
					response := &api.CompletionsResponse{
						ID:      "id",
						Object:  "object",
						Created: 0,
						Model:   "model",
						Choices: []api.Choice{},
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

				mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

				_, _, err = subject.Query(query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(tt.expectedError))
			})
		}

		when("a valid http response is received", func() {
			testValidHTTPResponse := func(subject *client.Client, expectedBody []byte, omitHistory bool) {
				const (
					answer = "content"
					tokens = 789
				)

				choice := api.Choice{
					Message: api.Message{
						Role:    client.AssistantRole,
						Content: answer,
					},
					FinishReason: "",
					Index:        0,
				}
				response := &api.CompletionsResponse{
					ID:      "id",
					Object:  "object",
					Created: 0,
					Model:   subject.Config.Model,
					Choices: []api.Choice{choice},
					Usage: api.Usage{
						PromptTokens:     123,
						CompletionTokens: 456,
						TotalTokens:      tokens,
					},
				}

				respBytes, err := json.Marshal(response)
				Expect(err).NotTo(HaveOccurred())
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).Return(respBytes, nil)

				var request api.CompletionsRequest
				err = json.Unmarshal(expectedBody, &request)
				Expect(err).NotTo(HaveOccurred())

				mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

				var h []history.History
				if !omitHistory {
					for _, msg := range request.Messages {
						h = append(h, history.History{
							Message: msg,
						})
					}

					mockHistoryStore.EXPECT().Write(append(h, history.History{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: answer,
						},
					}))
				}

				result, usage, err := subject.Query(query)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(answer))
				Expect(usage).To(Equal(tokens))
			}
			it("returns the expected result for a non-empty history", func() {
				h := []history.History{
					{
						Message: api.Message{
							Role:    client.SystemRole,
							Content: config.Role,
						},
					},
					{
						Message: api.Message{
							Role:    client.UserRole,
							Content: "question 1",
						},
					},
					{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: "answer 1",
						},
					},
				}

				messages = createMessages(h, query)
				factory.withHistory(h)
				subject := factory.buildClientWithoutConfig()

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, body, false)
			})
			it("ignores history when configured to do so", func() {
				mockHistoryStore.EXPECT().SetThread(config.Thread).Times(1)

				config := MockConfig()
				config.OmitHistory = true

				subject := client.New(mockCallerFactory, mockHistoryStore, mockTimer, mockReader, config, commandLineMode)
				Expect(err).NotTo(HaveOccurred())

				// Read and Write are never called on the history store
				mockHistoryStore.EXPECT().Read().Times(0)
				mockHistoryStore.EXPECT().Write(gomock.Any()).Times(0)

				messages = createMessages(nil, query)

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, body, true)
			})
			it("truncates the history as expected", func() {
				hs := []history.History{
					{
						Message: api.Message{
							Role:    client.SystemRole,
							Content: config.Role,
						},
						Timestamp: time.Time{},
					},
					{
						Message: api.Message{
							Role:    client.UserRole,
							Content: "question 1",
						},
						Timestamp: time.Time{},
					},
					{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: "answer 1",
						},
						Timestamp: time.Time{},
					},
					{
						Message: api.Message{
							Role:    client.UserRole,
							Content: "question 2",
						},
						Timestamp: time.Time{},
					},
					{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: "answer 2",
						},
						Timestamp: time.Time{},
					},
					{
						Message: api.Message{
							Role:    client.UserRole,
							Content: "question 3",
						},
						Timestamp: time.Time{},
					},
					{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: "answer 3",
						},
						Timestamp: time.Time{},
					},
				}

				messages = createMessages(hs, query)

				factory.withHistory(hs)
				subject := factory.buildClientWithoutConfig()

				// messages get truncated. Index 1+2 are cut out
				messages = append(messages[:1], messages[3:]...)

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, body, false)
			})
			it("should skip the first message when the model starts with o1Prefix", func() {
				factory.withHistory([]history.History{
					{Message: api.Message{Role: client.SystemRole, Content: "First message"}},
					{Message: api.Message{Role: client.UserRole, Content: "Second message"}},
				})

				o1Model := "o1-example-model"
				config.Model = o1Model

				subject := factory.buildClientWithoutConfig()
				subject.Config.Model = o1Model

				expectedBody, err := createBody([]api.Message{
					{Role: client.UserRole, Content: "Second message"},
					{Role: client.UserRole, Content: "test query"},
				}, false)
				Expect(err).NotTo(HaveOccurred())

				mockTimer.EXPECT().Now().Return(time.Now()).AnyTimes()
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).Return(nil, nil)

				_, _, _ = subject.Query("test query")
			})
			it("should include all messages when the model does not start with o1Prefix", func() {
				const systemRole = "System role for this test"

				factory.withHistory([]history.History{
					{Message: api.Message{Role: client.SystemRole, Content: systemRole}},
					{Message: api.Message{Role: client.UserRole, Content: "Second message"}},
				})

				regularModel := "gpt-4o"
				config.Model = regularModel

				subject := factory.buildClientWithoutConfig()
				subject.Config.Model = regularModel
				subject.Config.Role = systemRole

				expectedBody, err := createBody([]api.Message{
					{Role: client.SystemRole, Content: systemRole},
					{Role: client.UserRole, Content: "Second message"},
					{Role: client.UserRole, Content: "test query"},
				}, false)
				Expect(err).NotTo(HaveOccurred())

				mockTimer.EXPECT().Now().Return(time.Now()).AnyTimes()
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).Return(nil, nil)

				_, _, _ = subject.Query("test query")
			})
		})

		when("an image is provided", func() {
			const (
				query        = "test query"
				systemRole   = "System role for this test"
				errorMessage = "error message"
				image        = "path/to/image.wrong"
				website      = "https://website.com"
			)

			it.Before(func() {
				factory.withoutHistory()
			})

			it("should update a callout as expected when a valid image URL is provided", func() {
				subject := factory.buildClientWithoutConfig()
				subject.Config.Image = website
				subject.Config.Role = systemRole

				expectedBody, err := createBody([]api.Message{
					{Role: client.SystemRole, Content: systemRole},
					{Role: client.UserRole, Content: query},
					{Role: client.UserRole, Content: []api.ImageContent{{
						Type: "image_url",
						ImageURL: struct {
							URL string `json:"url"`
						}{
							URL: website,
						},
					}}},
				}, false)
				Expect(err).NotTo(HaveOccurred())

				mockTimer.EXPECT().Now().Return(time.Now()).Times(2)
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).Return(nil, nil)

				_, _, _ = subject.Query(query)
			})
			it("throws an error when the image mime type cannot be obtained due to an open-error", func() {
				subject := factory.buildClientWithoutConfig()
				subject.Config.Image = image
				subject.Config.Role = systemRole

				mockTimer.EXPECT().Now().Return(time.Now()).Times(2)
				mockReader.EXPECT().Open(image).Return(nil, errors.New(errorMessage))

				_, _, err := subject.Query(query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errorMessage))
			})
			it("throws an error when the image mime type cannot be obtained due to a read-error", func() {
				imageFile := &os.File{}

				subject := factory.buildClientWithoutConfig()
				subject.Config.Image = image
				subject.Config.Role = systemRole

				mockTimer.EXPECT().Now().Return(time.Now()).Times(2)
				mockReader.EXPECT().Open(image).Return(imageFile, nil)
				mockReader.EXPECT().ReadBufferFromFile(imageFile).Return(nil, errors.New(errorMessage))

				_, _, err := subject.Query(query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errorMessage))
			})
			it("throws an error when the image base64 encoded content cannot be obtained due to a read-error", func() {
				imageFile := &os.File{}

				subject := factory.buildClientWithoutConfig()
				subject.Config.Image = image
				subject.Config.Role = systemRole

				mockTimer.EXPECT().Now().Return(time.Now()).Times(2)
				mockReader.EXPECT().Open(image).Return(imageFile, nil)
				mockReader.EXPECT().ReadBufferFromFile(imageFile).Return(nil, nil)
				mockReader.EXPECT().ReadFile(image).Return(nil, errors.New(errorMessage))

				_, _, err := subject.Query(query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errorMessage))
			})
			it("should update a callout as expected when a valid local image is provided", func() {
				imageFile := &os.File{}

				subject := factory.buildClientWithoutConfig()
				subject.Config.Image = image
				subject.Config.Role = systemRole

				mockReader.EXPECT().Open(image).Return(imageFile, nil)
				mockReader.EXPECT().ReadBufferFromFile(imageFile).Return(nil, nil)
				mockReader.EXPECT().ReadFile(image).Return(nil, nil)

				expectedBody, err := createBody([]api.Message{
					{Role: client.SystemRole, Content: systemRole},
					{Role: client.UserRole, Content: query},
					{Role: client.UserRole, Content: []api.ImageContent{{
						Type: "image_url",
						ImageURL: struct {
							URL string `json:"url"`
						}{
							URL: "data:text/plain; charset=utf-8;base64,",
						},
					}}},
				}, false)
				Expect(err).NotTo(HaveOccurred())

				mockTimer.EXPECT().Now().Return(time.Now()).Times(2)
				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).Return(nil, nil)

				_, _, _ = subject.Query(query)
			})
		})
	})
	when("Stream()", func() {
		var (
			body     []byte
			messages []api.Message
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

			mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

			err := subject.Stream(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		when("a valid http response is received", func() {
			const answer = "answer"

			testValidHTTPResponse := func(subject *client.Client, hs []history.History, expectedBody []byte) {
				messages = createMessages(nil, query)
				body, err = createBody(messages, true)
				Expect(err).NotTo(HaveOccurred())

				mockCaller.EXPECT().Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, true).Return([]byte(answer), nil)

				mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

				messages = createMessages(hs, query)

				hs = []history.History{}

				for _, message := range messages {
					hs = append(hs, history.History{
						Message: message,
					})
				}

				mockHistoryStore.EXPECT().Write(append(hs, history.History{
					Message: api.Message{
						Role:    client.AssistantRole,
						Content: answer,
					},
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
				h := []history.History{
					{
						Message: api.Message{
							Role:    client.SystemRole,
							Content: config.Role,
						},
					},
					{
						Message: api.Message{
							Role:    client.UserRole,
							Content: "question x",
						},
					},
					{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: "answer x",
						},
					},
				}
				factory.withHistory(h)
				subject := factory.buildClientWithoutConfig()

				messages = createMessages(h, query)
				body, err = createBody(messages, true)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, h, body)
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
		it("filters gpt and o1 models as expected", func() {
			subject := factory.buildClientWithoutConfig()

			response, err := test.FileToBytes("models.json")
			Expect(err).NotTo(HaveOccurred())

			mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).Return(response, nil)

			result, err := subject.ListModels()
			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeEmpty())
			Expect(result).To(HaveLen(4))
			Expect(result[0]).To(Equal("* gpt-3.5-turbo (current)"))
			Expect(result[1]).To(Equal("- o1-mini"))
			Expect(result[2]).To(Equal("- gpt-3.5-turbo-0301"))
		})
	})
	when("ProvideContext()", func() {
		it("updates the history with the provided context", func() {
			subject := factory.buildClientWithoutConfig()

			context := "This is a story about a dog named Kya. Kya loves to play fetch and swim in the lake."
			mockHistoryStore.EXPECT().Read().Return(nil, nil).Times(1)

			mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

			subject.ProvideContext(context)

			Expect(len(subject.History)).To(Equal(2)) // The system message and the provided context

			systemMessage := subject.History[0]
			Expect(systemMessage.Role).To(Equal(client.SystemRole))
			Expect(systemMessage.Content).To(Equal(config.Role))

			contextMessage := subject.History[1]
			Expect(contextMessage.Role).To(Equal(client.UserRole))
			Expect(contextMessage.Content).To(Equal(context))
		})
		it("behaves as expected with a non empty initial history", func() {
			subject := factory.buildClientWithoutConfig()

			subject.History = []history.History{
				{
					Message: api.Message{
						Role:    client.SystemRole,
						Content: "system message",
					},
				},
				{
					Message: api.Message{
						Role: client.UserRole,
					},
				},
			}

			mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

			context := "test context"
			subject.ProvideContext(context)

			Expect(len(subject.History)).To(Equal(3))

			contextMessage := subject.History[2]
			Expect(contextMessage.Role).To(Equal(client.UserRole))
			Expect(contextMessage.Content).To(Equal(context))
		})
	})
}

func createBody(messages []api.Message, stream bool) ([]byte, error) {
	req := api.CompletionsRequest{
		Model:            config.Model,
		Messages:         messages,
		Stream:           stream,
		Temperature:      config.Temperature,
		TopP:             config.TopP,
		FrequencyPenalty: config.FrequencyPenalty,
		MaxTokens:        config.MaxTokens,
		PresencePenalty:  config.PresencePenalty,
		Seed:             config.Seed,
	}

	return json.Marshal(req)
}

func createMessages(historyEntries []history.History, query string) []api.Message {
	var messages []api.Message

	if len(historyEntries) == 0 {
		messages = append(messages, api.Message{
			Role:    client.SystemRole,
			Content: config.Role,
		})
	} else {
		for _, entry := range historyEntries {
			messages = append(messages, entry.Message)
		}
	}

	messages = append(messages, api.Message{
		Role:    client.UserRole,
		Content: query,
	})

	return messages
}

type clientFactory struct {
	mockHistoryStore *MockStore
}

func newClientFactory(mhs *MockStore) *clientFactory {
	return &clientFactory{
		mockHistoryStore: mhs,
	}
}

func (f *clientFactory) buildClientWithoutConfig() *client.Client {
	f.mockHistoryStore.EXPECT().SetThread(config.Thread).Times(1)

	c := client.New(mockCallerFactory, f.mockHistoryStore, mockTimer, mockReader, MockConfig(), commandLineMode)

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
		Debug:               false,
		Seed:                1,
	}
}
