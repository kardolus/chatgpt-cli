package client_test

import (
	"encoding/json"
	"errors"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/kardolus/chatgpt-poc/client"
	"github.com/kardolus/chatgpt-poc/types"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -destination=mocks_test.go -package=client_test github.com/kardolus/chatgpt-poc/http Caller

var (
	mockCtrl   *gomock.Controller
	mockCaller *MockCaller
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

		subject = client.New(mockCaller)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("Query()", func() {
		const query = "test query"

		var (
			err  error
			body []byte
		)

		it.Before(func() {
			body, err = client.CreateBody(query, false)
			Expect(err).NotTo(HaveOccurred())
		})

		it("throws an error when the http callout fails", func() {
			errorMsg := "error message"
			mockCaller.EXPECT().Post(client.URL, body, false).Return(nil, errors.New(errorMsg))

			_, err = subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errorMsg))
		})
		it("throws an error when the response is empty", func() {
			mockCaller.EXPECT().Post(client.URL, body, false).Return(nil, nil)

			_, err = subject.Query(query)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("empty response"))
		})
		it("throws an error when the response is a malformed json", func() {
			malformed := `{"invalid":"json"` // missing closing brace
			mockCaller.EXPECT().Post(client.URL, body, false).Return([]byte(malformed), nil)

			_, err = subject.Query(query)
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
			choice := types.Choice{
				Message: types.Message{
					Role:    "role",
					Content: "content",
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

			result, err := subject.Query(query)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal("content"))
		})
	})
}
