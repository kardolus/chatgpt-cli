package integration_test

import (
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/client"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/configmanager"
	"github.com/kardolus/chatgpt-cli/http"
	"github.com/kardolus/chatgpt-cli/types"
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
	var (
		restCaller *http.RestCaller
		defaults   types.Config
	)

	it.Before(func() {
		RegisterTestingT(t)

		apiKey := os.Getenv(configmanager.New(config.New()).WithEnvironment().APIKeyEnvVarName())
		Expect(apiKey).NotTo(BeEmpty())

		restCaller = http.New()
		restCaller.SetAPIKey(apiKey)

		defaults = config.New().ReadDefaults()
	})

	when("accessing the completion endpoint", func() {
		it("should return a successful response with expected keys", func() {
			body := types.CompletionsRequest{
				Messages: []types.Message{{
					Role:    client.SystemRole,
					Content: defaults.Role,
				}},
				Model:  defaults.Model,
				Stream: false,
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(defaults.URL+defaults.CompletionsPath, bytes, false)
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

		it("should return an error response with appropriate error details", func() {
			// Set the wrong API key
			restCaller.SetAPIKey("wrong-key")

			body := types.CompletionsRequest{
				Messages: []types.Message{{
					Role:    client.SystemRole,
					Content: defaults.Role,
				}},
				Model:  defaults.Model,
				Stream: false,
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(defaults.URL+defaults.CompletionsPath, bytes, false)
			Expect(err).To(HaveOccurred())

			var errorData types.ErrorResponse
			err = json.Unmarshal(resp, &errorData)
			Expect(err).NotTo(HaveOccurred())

			Expect(errorData.Error.Message).ShouldNot(BeEmpty(), "Expected error message to be present in the response")
			Expect(errorData.Error.Type).ShouldNot(BeEmpty(), "Expected error type to be present in the response")
			Expect(errorData.Error.Code).ShouldNot(BeEmpty(), "Expected error code to be present in the response")
		})
	})

	when("accessing the models endpoint", func() {
		it("should have the expected keys in the response", func() {
			resp, err := restCaller.Get(defaults.URL + defaults.ModelsPath)
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
