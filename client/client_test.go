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
//go:generate mockgen -destination=iomocks_test.go -package=client_test github.com/kardolus/chatgpt-cli/history Store

var (
	mockCtrl   *gomock.Controller
	mockCaller *MockCaller
	mockStore  *MockStore
	subject    *client.Client
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
		mockStore = NewMockStore(mockCtrl)
		subject = client.New(mockCaller, mockStore).WithCapacity(50)
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

		it.Before(func() {
			messages = createMessages(nil, query)
			body, err = createBody(messages, false)
			Expect(err).NotTo(HaveOccurred())
		})

		it("throws an error when the http callout fails", func() {
			mockStore.EXPECT().Read().Return(nil, nil).Times(1)

			errorMsg := "error message"
			mockCaller.EXPECT().Post(client.CompletionURL, body, false).Return(nil, errors.New(errorMsg))

			_, err := subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		it("throws an error when the response is empty", func() {
			mockStore.EXPECT().Read().Return(nil, nil).Times(1)
			mockCaller.EXPECT().Post(client.CompletionURL, body, false).Return(nil, nil)

			_, err := subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("empty response"))
		})
		it("throws an error when the response is a malformed json", func() {
			mockStore.EXPECT().Read().Return(nil, nil).Times(1)

			malformed := `{"invalid":"json"` // missing closing brace
			mockCaller.EXPECT().Post(client.CompletionURL, body, false).Return([]byte(malformed), nil)

			_, err := subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(HavePrefix("failed to decode response:"))
		})
		it("throws an error when the response is missing Choices", func() {
			mockStore.EXPECT().Read().Return(nil, nil).Times(1)

			response := &types.CompletionsResponse{
				ID:      "id",
				Object:  "object",
				Created: 0,
				Model:   "model",
				Choices: []types.Choice{},
			}

			respBytes, err := json.Marshal(response)
			Expect(err).NotTo(HaveOccurred())
			mockCaller.EXPECT().Post(client.CompletionURL, body, false).Return(respBytes, nil)

			_, err = subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no responses returned"))
		})
		when("a valid http response is received", func() {
			testValidHTTPResponse := func(history []types.Message, expectedBody []byte) {
				mockStore.EXPECT().Read().Return(history, nil).Times(1)

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
					Model:   client.DefaultGPTModel,
					Choices: []types.Choice{choice},
				}

				respBytes, err := json.Marshal(response)
				Expect(err).NotTo(HaveOccurred())
				mockCaller.EXPECT().Post(client.CompletionURL, expectedBody, false).Return(respBytes, nil)

				var request types.CompletionsRequest
				err = json.Unmarshal(expectedBody, &request)
				Expect(err).NotTo(HaveOccurred())

				mockStore.EXPECT().Write(append(request.Messages, types.Message{
					Role:    client.AssistantRole,
					Content: answer,
				}))

				result, err := subject.Query(query)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(answer))
			}

			it("returns the expected result for an empty history", func() {
				testValidHTTPResponse(nil, body)
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
				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(history, body)
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

				// messages get truncated. Index 1+2 are cut out
				messages = append(messages[:1], messages[3:]...)

				body, err = createBody(messages, false)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(history, body)
			})
		})
	})
	when("Stream()", func() {
		var (
			body     []byte
			messages []types.Message
			err      error
		)

		it.Before(func() {
			messages = createMessages(nil, query)
			body, err = createBody(messages, true)
			Expect(err).NotTo(HaveOccurred())
		})

		it("throws an error when the http callout fails", func() {
			mockStore.EXPECT().Read().Return(nil, nil).Times(1)

			errorMsg := "error message"
			mockCaller.EXPECT().Post(client.CompletionURL, body, true).Return(nil, errors.New(errorMsg))

			err := subject.Stream(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		when("a valid http response is received", func() {
			const answer = "answer"

			testValidHTTPResponse := func(history []types.Message, expectedBody []byte) {
				mockStore.EXPECT().Read().Return(history, nil).Times(1)
				mockCaller.EXPECT().Post(client.CompletionURL, expectedBody, true).Return([]byte(answer), nil)

				messages = createMessages(history, query)

				mockStore.EXPECT().Write(append(messages, types.Message{
					Role:    client.AssistantRole,
					Content: answer,
				}))

				err := subject.Stream(query)
				Expect(err).NotTo(HaveOccurred())
			}

			it("returns the expected result for an empty history", func() {
				testValidHTTPResponse(nil, body)
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
				body, err = createBody(messages, true)
				Expect(err).NotTo(HaveOccurred())

				testValidHTTPResponse(history, body)
			})
		})
	})
	when("ListModels()", func() {
		it("throws an error when the http callout fails", func() {
			errorMsg := "error message"
			mockCaller.EXPECT().Get(client.ModelURL).Return(nil, errors.New(errorMsg))

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		it("throws an error when the response is empty", func() {
			mockCaller.EXPECT().Get(client.ModelURL).Return(nil, nil)

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("empty response"))
		})
		it("throws an error when the response is a malformed json", func() {
			malformed := `{"invalid":"json"` // missing closing brace
			mockCaller.EXPECT().Get(client.ModelURL).Return([]byte(malformed), nil)

			_, err := subject.ListModels()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(HavePrefix("failed to decode response:"))
		})
		it("filters gpt models as expected", func() {
			response, err := utils.FileToBytes("models.json")
			Expect(err).NotTo(HaveOccurred())

			mockCaller.EXPECT().Get(client.ModelURL).Return(response, nil)

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
			context := "This is a story about a dog named Kya. Kya loves to play fetch and swim in the lake."
			mockStore.EXPECT().Read().Return(nil, nil).Times(1)
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

func createBody(messages []types.Message, stream bool) ([]byte, error) {
	req := types.CompletionsRequest{
		Model:    client.DefaultGPTModel,
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
