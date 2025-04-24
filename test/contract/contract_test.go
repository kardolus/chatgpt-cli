package contract_test

import (
	"bytes"
	"encoding/json"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/config"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContract(t *testing.T) {
	spec.Run(t, "Contract Tests", testContract, spec.Report(report.Terminal{}), spec.Parallel())
}

func testContract(t *testing.T, when spec.G, it spec.S) {
	var (
		restCaller *http.RestCaller
		cfg        config.Config
	)

	it.Before(func() {
		RegisterTestingT(t)

		apiKey := os.Getenv(config.NewManager(config.NewStore()).WithEnvironment().APIKeyEnvVarName())
		Expect(apiKey).NotTo(BeEmpty())

		cfg = config.NewStore().ReadDefaults()
		cfg.APIKey = apiKey

		restCaller = http.New(cfg)
	})

	when("accessing the completion endpoint", func() {
		it("should return a successful response with expected keys", func() {
			body := api.CompletionsRequest{
				Messages: []api.Message{{
					Role:    client.SystemRole,
					Content: cfg.Role,
				}},
				MaxTokens: 1234,
				Model:     cfg.Model,
				Stream:    false,
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(cfg.URL+cfg.CompletionsPath, bytes, false)
			Expect(err).NotTo(HaveOccurred())

			var data api.CompletionsResponse
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
			body := api.CompletionsRequest{
				Messages: []api.Message{{
					Role:    client.SystemRole,
					Content: cfg.Role,
				}},
				Model:  "no-such-model",
				Stream: false,
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(cfg.URL+cfg.CompletionsPath, bytes, false)
			Expect(err).To(HaveOccurred())

			var errorData api.ErrorResponse
			err = json.Unmarshal(resp, &errorData)
			Expect(err).NotTo(HaveOccurred())

			Expect(errorData.Error.Message).ShouldNot(BeEmpty(), "Expected error message to be present in the response")
			Expect(errorData.Error.Type).ShouldNot(BeEmpty(), "Expected error type to be present in the response")
			Expect(errorData.Error.Code).ShouldNot(BeEmpty(), "Expected error code to be present in the response")
		})
	})

	when("accessing the models endpoint", func() {
		it("should have the expected keys in the response", func() {
			resp, err := restCaller.Get(cfg.URL + cfg.ModelsPath)
			Expect(err).NotTo(HaveOccurred())

			var data api.ListModelsResponse
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
			}
		})
	})

	when("accessing the responses endpoint", func() {
		it("should return a successful response with expected keys and content", func() {
			body := api.ResponsesRequest{
				Model: "o1-pro",
				Input: []api.Message{{
					Role:    client.UserRole,
					Content: "what is the capital of sweden",
				}},
				MaxOutputTokens: 64,
				Reasoning: api.Reasoning{
					Effort: "low",
				},
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(cfg.URL+cfg.ResponsesPath, bytes, false)
			Expect(err).NotTo(HaveOccurred())

			var data api.ResponsesResponse
			err = json.Unmarshal(resp, &data)
			Expect(err).NotTo(HaveOccurred())

			// High-level structure
			Expect(data.ID).To(HavePrefix("resp_"))
			Expect(data.Object).To(Equal("response"))
			Expect(data.CreatedAt).To(BeNumerically(">", 0))
			Expect(data.Model).To(ContainSubstring("o1-pro"))
			Expect(data.Status).To(SatisfyAny(Equal("completed"), Equal("incomplete")))
			Expect(data.Output).NotTo(BeEmpty())

			// Check for a message block with output_text
			var textFound bool
			for _, block := range data.Output {
				if block.Type == "message" {
					for _, content := range block.Content {
						if content.Type == "output_text" && strings.Contains(strings.ToLower(content.Text), "stockholm") {
							textFound = true
						}
					}
				}
			}
			Expect(textFound).To(BeTrue(), "expected to find 'Stockholm' in an output_text block")

			// Usage stats present
			Expect(data.Usage.TotalTokens).To(BeNumerically(">", 0))
			Expect(data.Usage.InputTokens).To(BeNumerically(">", 0))
			Expect(data.Usage.OutputTokens).To(BeNumerically(">", 0))
		})
	})

	when("accessing the speech endpoint", func() {
		it("should return audio data for a valid request", func() {
			body := api.Speech{
				Model:          "gpt-4o-mini-tts",
				Input:          "Hello world",
				Voice:          "nova",
				ResponseFormat: "mp3",
			}

			bytes, err := json.Marshal(body)
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.Post(cfg.URL+cfg.SpeechPath, bytes, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeEmpty())

			tmpFile, err := os.CreateTemp("", "speech-*.mp3")
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write(resp)
			Expect(err).NotTo(HaveOccurred())

			err = tmpFile.Close()
			Expect(err).NotTo(HaveOccurred())

			isMP3 := strings.HasPrefix(string(resp), "ID3") || (len(resp) > 2 && resp[0] == 0xFF && resp[1]&0xE0 == 0xE0)
			Expect(isMP3).To(BeTrue(), "response does not appear to be valid MP3 audio")
		})
	})

	when("accessing the transcriptions endpoint", func() {
		it("should return transcribed text for a valid audio file", func() {
			audioPath := "../data/hello.wav"
			file, err := os.Open(audioPath)
			Expect(err).NotTo(HaveOccurred())
			defer file.Close()

			var buf bytes.Buffer
			writer := multipart.NewWriter(&buf)

			err = writer.WriteField("model", "gpt-4o-transcribe")
			Expect(err).NotTo(HaveOccurred())

			part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
			Expect(err).NotTo(HaveOccurred())

			_, err = io.Copy(part, file)
			Expect(err).NotTo(HaveOccurred())

			err = writer.Close()
			Expect(err).NotTo(HaveOccurred())

			resp, err := restCaller.PostWithHeaders(cfg.URL+cfg.TranscriptionsPath, buf.Bytes(), map[string]string{
				"Content-Type":  writer.FormDataContentType(),
				"Authorization": "Bearer " + cfg.APIKey,
			})
			Expect(err).NotTo(HaveOccurred())

			var res struct {
				Text string `json:"text"`
			}
			err = json.Unmarshal(resp, &res)
			Expect(err).NotTo(HaveOccurred())

			Expect(res.Text).NotTo(BeEmpty(), "Expected transcribed text to be returned")
			Expect(strings.ToLower(res.Text)).To(ContainSubstring("hello"))
		})
	})
}
