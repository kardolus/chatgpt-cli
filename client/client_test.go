package client_test

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/types"
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
	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockCaller = NewMockCaller(mockCtrl)
		mockStore = NewMockStore(mockCtrl)

		mockStore.EXPECT().Read().Return(nil, nil).Times(1)
		subject = client.New(mockCaller, mockStore)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("Query()", func() {
		const query = "test query"

		var (
			body     []byte
			messages []types.Message
			err      error
		)

		it.Before(func() {
			messages = createMessages(nil, query)
			body, err = createBody(messages)
			Expect(err).NotTo(HaveOccurred())
		})

		it("throws an error when the http callout fails", func() {
			errorMsg := "error message"
			mockCaller.EXPECT().Post(client.URL, body, false).Return(nil, errors.New(errorMsg))

			_, err := subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		it("throws an error when the response is empty", func() {
			mockCaller.EXPECT().Post(client.URL, body, false).Return(nil, nil)

			_, err := subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("empty response"))
		})
		it("throws an error when the response is a malformed json", func() {
			malformed := `{"invalid":"json"` // missing closing brace
			mockCaller.EXPECT().Post(client.URL, body, false).Return([]byte(malformed), nil)

			_, err := subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).Should(HavePrefix("failed to decode response:"))
		})
		it("throws an error when the response is missing Choices", func() {
			response := &types.Response{
				ID:      "id",
				Object:  "object",
				Created: 0,
				Model:   "model",
				Choices: []types.Choice{},
			}

			respBytes, err := json.Marshal(response)
			Expect(err).NotTo(HaveOccurred())
			mockCaller.EXPECT().Post(client.URL, body, false).Return(respBytes, nil)

			_, err = subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("no responses returned"))
		})
		it("parses a valid http response as expected", func() {
			const answer = "content"

			choice := types.Choice{
				Message: types.Message{
					Role:    "role",
					Content: answer,
				},
				FinishReason: "",
				Index:        0,
			}
			response := &types.Response{
				ID:      "id",
				Object:  "object",
				Created: 0,
				Model:   "model",
				Choices: []types.Choice{choice},
			}

			respBytes, err := json.Marshal(response)
			Expect(err).NotTo(HaveOccurred())
			mockCaller.EXPECT().Post(client.URL, body, false).Return(respBytes, nil)

			mockStore.EXPECT().Write(append(messages, types.Message{
				Role:    client.AssistantRole,
				Content: answer,
			}))

			result, err := subject.Query(query)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(answer))
		})
	})
}

func createBody(messages []types.Message) ([]byte, error) {
	req := types.Request{
		Model:    client.GPTModel,
		Messages: messages,
		Stream:   false,
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
	}

	messages = append(messages, types.Message{
		Role:    client.UserRole,
		Content: query,
	})

	return messages
}
