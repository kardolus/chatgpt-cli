package client_test

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=callermocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/http Caller
//go:generate mockgen -destination=historymocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/history HistoryStore
//go:generate mockgen -destination=configmocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/config ConfigStore

var (
	mockCtrl         *gomock.Controller
	mockCaller       *MockCaller
	mockHistoryStore *MockHistoryStore
	mockConfigStore  *MockConfigStore
	factory          *clientFactory
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
		mockConfigStore = NewMockConfigStore(mockCtrl)
		factory = newClientFactory(mockCaller, mockConfigStore, mockHistoryStore)
	})

	it.After(func() {
		mockCtrl.Finish()
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
				body, err = createBody(messages, subject.Model, false)
				Expect(err).NotTo(HaveOccurred())

				respBytes, err := tt.setupPostReturn()
				Expect(err).NotTo(HaveOccurred())
				mockCaller.EXPECT().Post(client.DefaultServiceURL+client.CompletionPath, body, false).Return(respBytes, tt.postError)

				_, err = subject.Query(query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(tt.expectedError))
			})
		}

		when("a valid http response is received", func() {
			testValidHTTPResponse := func(subject *client.Client, history []types.Message, expectedBody []byte) {
				const answer = "content"

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
					Model:   subject.Model,
					Choices: []types.Choice{choice},
				}

				respBytes, err := json.Marshal(response)
				Expect(err).NotTo(HaveOccurred())
				mockCaller.EXPECT().Post(client.DefaultServiceURL+client.CompletionPath, expectedBody, false).Return(respBytes, nil)

				var request types.CompletionsRequest
				err = json.Unmarshal(expectedBody, &request)
				Expect(err).NotTo(HaveOccurred())

				mockHistoryStore.EXPECT().Write(append(request.Messages, types.Message{
					Role:    client.AssistantRole,
					Content: answer,
				}))

				result, err := subject.Query(query)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(answer))
			}

			it("uses the model specified by the WithModel method instead of the default model", func() {
				const model = "overwritten"

				messages = createMessages(nil, query)
				factory.withoutHistory()
				subject := factory.buildClientWithoutConfig().WithModel(model)

				body, err = createBody(messages, model, false)
				Expect(err).NotTo(HaveOccurred())
				testValidHTTPResponse(subject, nil, body)
			})
			it("uses the model specified by the configuration instead of the default model", func() {
				const model = "overwritten"

				messages = createMessages(nil, query)
				factory.withoutHistory()
				subject := factory.buildClientWithConfig(types.Config{
					Model: model,
				})

				body, err = createBody(messages, model, false)
				Expect(err).NotTo(HaveOccurred())
				testValidHTTPResponse(subject, nil, body)
			})
			it("when WithModel is used and a configuration is present, WithModel takes precedence", func() {
				const model = "with-model"

				messages = createMessages(nil, query)
				factory.withoutHistory()
				subject := factory.buildClientWithConfig(types.Config{
					Model: "config-model",
				}).WithModel(model)

				body, err = createBody(messages, model, false)
				Expect(err).NotTo(HaveOccurred())
				testValidHTTPResponse(subject, nil, body)
			})
			it("returns the expected result for a non-empty history", func() {
				history := []types.Message{
					{
						Role:    client.SystemRole,
						Content: client.AssistantContent,
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

				body, err = createBody(messages, subject.Model, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, history, body)
			})
			it("truncates the history as expected", func() {
				history := []types.Message{
					{
						Role:    client.SystemRole,
						Content: client.AssistantContent,
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

				body, err = createBody(messages, subject.Model, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, history, body)
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
			body, err = createBody(messages, subject.Model, true)
			Expect(err).NotTo(HaveOccurred())

			errorMsg := "error message"
			mockCaller.EXPECT().Post(client.DefaultServiceURL+client.CompletionPath, body, true).Return(nil, errors.New(errorMsg))

			err := subject.Stream(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		when("a valid http response is received", func() {
			const answer = "answer"

			testValidHTTPResponse := func(subject *client.Client, history []types.Message, expectedBody []byte) {
				messages = createMessages(nil, query)
				body, err = createBody(messages, subject.Model, true)
				Expect(err).NotTo(HaveOccurred())

				mockCaller.EXPECT().Post(client.DefaultServiceURL+client.CompletionPath, expectedBody, true).Return([]byte(answer), nil)

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
				body, err = createBody(messages, subject.Model, true)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, nil, body)
			})
			it("returns the expected result for a non-empty history", func() {
				history := []types.Message{
					{
						Role:    client.SystemRole,
						Content: client.AssistantContent,
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
				body, err = createBody(messages, subject.Model, true)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(subject, history, body)
			})
		})
	})
	when("ListModels()", func() {
		it("throws an error when the http callout fails", func() {
			subject := factory.buildClientWithoutConfig()

			errorMsg := "error message"
			mockCaller.EXPECT().Get(client.DefaultServiceURL+client.ModelPath).Return(nil, errors.New(errorMsg))

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		it("throws an error when the response is empty", func() {
			subject := factory.buildClientWithoutConfig()

			mockCaller.EXPECT().Get(client.DefaultServiceURL+client.ModelPath).Return(nil, nil)

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("empty response"))
		})
		it("throws an error when the response is a malformed json", func() {
			subject := factory.buildClientWithoutConfig()

			malformed := `{"invalid":"json"` // missing closing brace
			mockCaller.EXPECT().Get(client.DefaultServiceURL+client.ModelPath).Return([]byte(malformed), nil)

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(HavePrefix("failed to decode response:"))
		})
		it("filters gpt models as expected", func() {
			subject := factory.buildClientWithoutConfig()

			response, err := utils.FileToBytes("models.json")
			Expect(err).NotTo(HaveOccurred())

			mockCaller.EXPECT().Get(client.DefaultServiceURL+client.ModelPath).Return(response, nil)

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
			Expect(systemMessage.Content).To(Equal("You are a helpful assistant."))

			contextMessage := subject.History[1]
			Expect(contextMessage.Role).To(Equal(client.UserRole))
			Expect(contextMessage.Content).To(Equal(context))
		})
	})
}

func createBody(messages []types.Message, model string, stream bool) ([]byte, error) {
	req := types.CompletionsRequest{
		Model:    model,
		Messages: messages,
		Stream:   stream,
	}

	return json.Marshal(req)
}

func createMessages(history []types.Message, query string) []types.Message {
	var messages []types.Message

	if len(history) == 0 {
		messages = append(messages, types.Message{
			Role:    client.SystemRole,
			Content: client.AssistantContent,
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
	mockCaller       *MockCaller
	mockConfigStore  *MockConfigStore
	mockHistoryStore *MockHistoryStore
}

func newClientFactory(mc *MockCaller, mcs *MockConfigStore, mhs *MockHistoryStore) *clientFactory {
	return &clientFactory{
		mockCaller:       mc,
		mockConfigStore:  mcs,
		mockHistoryStore: mhs,
	}
}

func (f *clientFactory) buildClientWithoutConfig() *client.Client {
	f.mockConfigStore.EXPECT().Read().Return(types.Config{}, nil).Times(1)
	return client.New(f.mockCaller, f.mockConfigStore, f.mockHistoryStore).WithCapacity(50)
}

func (f *clientFactory) buildClientWithConfig(config types.Config) *client.Client {
	f.mockConfigStore.EXPECT().Read().Return(config, nil).Times(1)
	return client.New(f.mockCaller, f.mockConfigStore, f.mockHistoryStore).WithCapacity(50)
}

func (f *clientFactory) withoutHistory() {
	f.mockHistoryStore.EXPECT().Read().Return(nil, nil).Times(1)
}

func (f *clientFactory) withHistory(history []types.Message) {
	f.mockHistoryStore.EXPECT().Read().Return(history, nil).Times(1)
}
