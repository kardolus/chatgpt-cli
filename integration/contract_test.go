package integration_test

import (
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
	"github.com/kardolus/chatgpt-cli/utils"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"os"
	"testing"
)

func TestContract(t *testing.T) {
	spec.Run(t, "Contract Tests", testContract, spec.Report(report.Terminal{}))
}

func testContract(t *testing.T, when spec.G, it spec.S) {
	var restCaller *http.RestCaller

	it.Before(func() {
		RegisterTestingT(t)

		apiKey := os.Getenv(utils.OpenAPIKeyEnv)
		Expect(apiKey).NotTo(BeEmpty())

		restCaller = http.New().WithSecret(apiKey)
	})

	when("accessing the completion endpoint", func() {
		it("should have the expected keys in the response", func() {
			body := types.CompletionsRequest{
				Messages: []types.Message{{
					Role:    client.SystemRole,
					Content: client.AssistantContent,
				}},
				Model:  client.DefaultGPTModel,
				Stream: false,
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(client.CompletionURL, bytes, false)
			Expect(err).NotTo(HaveOccurred())

			var data types.CompletionsResponse
			err = json.Unmarshal(resp, &data)
			Expect(err).NotTo(HaveOccurred())

			Expect(data.ID).ShouldNot(BeEmpty(), "Expected ID to be present in the response")
			Expect(data.Object).ShouldNot(BeEmpty(), "Expected Object to be present in the response")
			Expect(data.Created).ShouldNot(BeZero(), "Expected Created to be present in the response")
			Expect(data.Model).ShouldNot(BeEmpty(), "Expected Model to be present in the response")
			Expect(data.Usage).ShouldNot(BeNil(), "Expected Usage to be present in the response")
			Expect(data.Choices).ShouldNot(BeNil(), "Expected Choices to be present in the response")
		})
	})

	when("accessing the models endpoint", func() {
		it("should have the expected keys in the response", func() {
			resp, err := restCaller.Get(client.ModelURL)
			Expect(err).NotTo(HaveOccurred())

			var data types.ListModelsResponse
			err = json.Unmarshal(resp, &data)
			Expect(err).NotTo(HaveOccurred())

			Expect(data.Object).ShouldNot(BeEmpty(), "Expected Object to be present in the response")
			Expect(data.Data).ShouldNot(BeNil(), "Expected Data to be present in the response")
			Expect(data.Data).NotTo(BeEmpty())

			for _, model := range data.Data {
				Expect(model.Id).ShouldNot(BeEmpty(), "Expected Model Id to be present in the response")
				Expect(model.Object).ShouldNot(BeEmpty(), "Expected Model Object to be present in the response")
				Expect(model.Created).ShouldNot(BeZero(), "Expected Model Created to be present in the response")
				Expect(model.OwnedBy).ShouldNot(BeEmpty(), "Expected Model OwnedBy to be present in the response")
				Expect(model.Permission).ShouldNot(BeNil(), "Expected Model Permission to be present in the response")
				Expect(model.Root).ShouldNot(BeEmpty(), "Expected Model Root to be present in the response")
			}
		})
	})
}
