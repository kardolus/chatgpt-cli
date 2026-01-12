package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/test"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testLLM(t *testing.T, when spec.G, it spec.S) {
	const query = "test query"

	when("LLM()", func() {
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
				{
					description: "throws an error when the response cannot be casted to a string",
					setupPostReturn: func() ([]byte, error) {
						response := &api.CompletionsResponse{
							ID:      "id",
							Object:  "object",
							Created: 0,
							Model:   "model",
							Choices: []api.Choice{
								{
									Message: api.Message{
										Role:    client.AssistantRole,
										Content: 123, // cannot be converted to a string
									},
									FinishReason: "",
									Index:        0,
								},
							},
						}

						respBytes, err := json.Marshal(response)
						return respBytes, err
					},
					postError:     nil,
					expectedError: "response cannot be converted to a string",
				},
			}

			for _, tt := range tests {
				tt := tt
				it(tt.description, func() {
					factory.withoutHistory()
					subject := factory.buildClientWithoutConfig()

					messages = createMessages(nil, query)
					body, err = createBody(messages, false)
					Expect(err).NotTo(HaveOccurred())

					respBytes, err := tt.setupPostReturn()
					Expect(err).NotTo(HaveOccurred())

					mockCaller.EXPECT().
						Post(subject.Config.URL+subject.Config.CompletionsPath, body, false).
						Return(respBytes, tt.postError)

					mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

					_, _, err = subject.Query(context.Background(), query)
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(tt.expectedError))
				})
			}

			it("errors when the model is realtime (no HTTP call is made)", func() {
				factory.withoutHistory()
				subject := factory.buildClientWithoutConfig()

				realtimeModel := "gpt-realtime"
				subject.Config.Model = realtimeModel
				config.Model = realtimeModel

				mockCaller.EXPECT().
					Post(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

				_, _, err := subject.Query(context.Background(), query)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(ContainSubstring("realtime"))
			})

			it("errors when web is enabled for a non-gpt5 model", func() {
				factory.withoutHistory()
				subject := factory.buildClientWithoutConfig()

				subject.Config.Model = "gpt-4o"
				subject.Config.Web = true

				mockCaller.EXPECT().
					Post(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

				_, _, err := subject.Query(context.Background(), query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("web search"))
			})

			it("errors when web is enabled for gpt-5-search", func() {
				factory.withoutHistory()
				subject := factory.buildClientWithoutConfig()

				subject.Config.Model = "gpt-5-search"
				subject.Config.Web = true

				mockCaller.EXPECT().
					Post(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

				_, _, err := subject.Query(context.Background(), query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("web search"))
			})

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

					mockCaller.EXPECT().
						Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).
						Return(respBytes, nil)

					var request api.CompletionsRequest
					err = json.Unmarshal(expectedBody, &request)
					Expect(err).NotTo(HaveOccurred())

					mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

					var h []history.History
					if !omitHistory {
						for _, msg := range request.Messages {
							h = append(h, history.History{Message: msg})
						}

						mockHistoryStore.EXPECT().Write(append(h, history.History{
							Message: api.Message{
								Role:    client.AssistantRole,
								Content: answer,
							},
						}))
					}

					result, usage, err := subject.Query(context.Background(), query)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).To(Equal(answer))
					Expect(usage).To(Equal(tokens))
				}

				it("returns the expected result for a non-empty history", func() {
					h := []history.History{
						{Message: api.Message{Role: client.SystemRole, Content: config.Role}},
						{Message: api.Message{Role: client.UserRole, Content: "question 1"}},
						{Message: api.Message{Role: client.AssistantRole, Content: "answer 1"}},
					}

					messages = createMessages(h, query)
					factory.withHistory(h)
					subject := factory.buildClientWithoutConfig()

					body, err = createBody(messages, false)
					Expect(err).NotTo(HaveOccurred())

					testValidHTTPResponse(subject, body, false)
				})

				it("ignores history when configured to do so", func() {
					cfg := MockConfig()
					cfg.OmitHistory = true

					subject := client.New(
						mockCallerFactory,
						mockHistoryStore,
						mockTimer,
						mockReader,
						mockWriter,
						cfg,
					)

					// History should never be read or written
					mockHistoryStore.EXPECT().Read().Times(0)
					mockHistoryStore.EXPECT().Write(gomock.Any()).Times(0)

					var capturedBody []byte

					validHTTPResponseBytes := []byte(`{
					  "id": "chatcmpl_test",
					  "object": "chat.completion",
					  "created": 0,
					  "model": "gpt-4o",
					  "choices": [
						{
						  "index": 0,
						  "message": { "role": "assistant", "content": "ok" },
						  "finish_reason": "stop"
						}
					  ],
					  "usage": { "prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2 }
					}`)

					mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

					mockCaller.EXPECT().
						Post(gomock.Any(), gomock.Any(), gomock.Any()).
						DoAndReturn(func(endpoint string, body []byte, stream bool) ([]byte, error) {
							capturedBody = body
							return validHTTPResponseBytes, nil
						})

					_, _, err := subject.Query(context.Background(), query)
					Expect(err).NotTo(HaveOccurred())

					Expect(capturedBody).NotTo(BeNil())

					var req api.CompletionsRequest
					err = json.Unmarshal(capturedBody, &req)
					Expect(err).NotTo(HaveOccurred())

					Expect(req.Messages).To(HaveLen(1))
					Expect(req.Messages[0].Role).To(Equal("user"))
					Expect(req.Messages[0].Content).To(Equal(query))
				})

				it("truncates the history as expected", func() {
					hs := []history.History{
						{Message: api.Message{Role: client.SystemRole, Content: config.Role}, Timestamp: time.Time{}},
						{Message: api.Message{Role: client.UserRole, Content: "question 1"}, Timestamp: time.Time{}},
						{Message: api.Message{Role: client.AssistantRole, Content: "answer 1"}, Timestamp: time.Time{}},
						{Message: api.Message{Role: client.UserRole, Content: "question 2"}, Timestamp: time.Time{}},
						{Message: api.Message{Role: client.AssistantRole, Content: "answer 2"}, Timestamp: time.Time{}},
						{Message: api.Message{Role: client.UserRole, Content: "question 3"}, Timestamp: time.Time{}},
						{Message: api.Message{Role: client.AssistantRole, Content: "answer 3"}, Timestamp: time.Time{}},
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
					mockCaller.EXPECT().
						Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).
						Return(nil, nil)

					_, _, _ = subject.Query(context.Background(), "test query")
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
					mockCaller.EXPECT().
						Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, false).
						Return(nil, nil)

					_, _, _ = subject.Query(context.Background(), "test query")
				})

				it("should omit Temperature and TopP when the model matches SearchModelPattern", func() {
					searchModel := "gpt-4o-search-preview"
					config.Model = searchModel
					config.Role = "role for search test"

					factory.withHistory([]history.History{
						{Message: api.Message{Role: client.SystemRole, Content: config.Role}},
					})

					subject := factory.buildClientWithoutConfig()
					subject.Config.Model = searchModel

					mockTimer.EXPECT().Now().Return(time.Now()).AnyTimes()

					mockCaller.EXPECT().
						Post(gomock.Any(), gomock.Any(), false).
						DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
							var req map[string]interface{}
							Expect(json.Unmarshal(body, &req)).To(Succeed())
							Expect(req).NotTo(HaveKey("temperature"))
							Expect(req).NotTo(HaveKey("top_p"))
							return nil, nil
						})

					_, _, _ = subject.Query(context.Background(), "test query")
				})

				it("should include Temperature and TopP when the model does not match SearchModelPattern", func() {
					regularModel := "gpt-4o"
					config.Model = regularModel
					config.Role = "regular model test"

					factory.withHistory([]history.History{
						{Message: api.Message{Role: client.SystemRole, Content: config.Role}},
					})

					subject := factory.buildClientWithoutConfig()
					subject.Config.Model = regularModel

					mockTimer.EXPECT().Now().Return(time.Now()).AnyTimes()

					mockCaller.EXPECT().
						Post(gomock.Any(), gomock.Any(), false).
						DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
							var req map[string]interface{}
							Expect(json.Unmarshal(body, &req)).To(Succeed())

							Expect(req).To(HaveKeyWithValue("temperature", BeNumerically("==", config.Temperature)))
							Expect(req).To(HaveKeyWithValue("top_p", BeNumerically("==", config.TopP)))
							return nil, nil
						})

					_, _, _ = subject.Query(context.Background(), "test query")
				})

				it("forces Responses API when web is enabled", func() {
					factory.withoutHistory()
					subject := factory.buildClientWithoutConfig()

					subject.Config.Model = "gpt-5"
					subject.Config.Web = true
					subject.Config.WebContextSize = "low"

					mockTimer.EXPECT().Now().Times(3)
					mockHistoryStore.EXPECT().Write(gomock.Any())

					response := api.ResponsesResponse{
						Output: []api.Output{{
							Type:    "message",
							Content: []api.Content{{Type: "output_text", Text: "hi"}},
						}},
						Usage: api.TokenUsage{TotalTokens: 1},
					}
					raw, _ := json.Marshal(response)

					mockCaller.EXPECT().
						Post(subject.Config.URL+"/v1/responses", gomock.Any(), false).
						DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
							var req map[string]any
							Expect(json.Unmarshal(body, &req)).To(Succeed())
							Expect(req).To(HaveKey("tools"))
							return raw, nil
						})

					_, _, err := subject.Query(context.Background(), query)
					Expect(err).NotTo(HaveOccurred())
				})

				it("adds web_search tool when web is enabled", func() {
					factory.withoutHistory()
					subject := factory.buildClientWithoutConfig()

					subject.Config.Model = "gpt-5"
					subject.Config.Web = true
					subject.Config.WebContextSize = "low"

					mockTimer.EXPECT().Now().Times(3)
					mockHistoryStore.EXPECT().Write(gomock.Any())

					response := api.ResponsesResponse{
						Output: []api.Output{{
							Type:    "message",
							Content: []api.Content{{Type: "output_text", Text: "ok"}},
						}},
						Usage: api.TokenUsage{TotalTokens: 1},
					}
					raw, _ := json.Marshal(response)

					mockCaller.EXPECT().
						Post(subject.Config.URL+"/v1/responses", gomock.Any(), false).
						DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
							var req map[string]any
							Expect(json.Unmarshal(body, &req)).To(Succeed())

							tools := req["tools"].([]any)
							Expect(tools).To(HaveLen(1))

							tool := tools[0].(map[string]any)
							Expect(tool).To(HaveKeyWithValue("type", "web_search"))
							Expect(tool).To(HaveKeyWithValue("search_context_size", "low"))

							return raw, nil
						})

					_, _, err := subject.Query(context.Background(), query)
					Expect(err).NotTo(HaveOccurred())
				})
			})

			when("the model is o1-pro or gpt-5", func() {
				models := []string{"o1-pro", "gpt-5"}

				for _, m := range models {
					m := m
					when(fmt.Sprintf("the model is %s", m), func() {
						const (
							query       = "what's the weather"
							systemRole  = "you are helpful"
							totalTokens = 777
						)

						it.Before(func() {
							config.Model = m
							config.Role = systemRole
							factory.withoutHistory()
						})

						assertResponsesRequest := func(body []byte) {
							var req map[string]any
							Expect(json.Unmarshal(body, &req)).To(Succeed())

							Expect(req).To(HaveKeyWithValue("model", m))
							Expect(req).To(HaveKey("input"))
							Expect(req).To(HaveKey("max_output_tokens"))
							Expect(req).To(HaveKey("reasoning"))
							Expect(req).To(HaveKeyWithValue("stream", false))

							// Validate reasoning.effort
							reasoning, ok := req["reasoning"].(map[string]any)
							Expect(ok).To(BeTrue())
							Expect(reasoning).To(HaveKeyWithValue("effort", "low"))

							// Validate input messages
							input, ok := req["input"].([]any)
							Expect(ok).To(BeTrue())
							Expect(input).To(HaveLen(2))

							msg0, ok := input[0].(map[string]any)
							Expect(ok).To(BeTrue())
							Expect(msg0).To(HaveKeyWithValue("role", client.SystemRole))
							Expect(msg0).To(HaveKeyWithValue("content", systemRole))

							msg1, ok := input[1].(map[string]any)
							Expect(ok).To(BeTrue())
							Expect(msg1).To(HaveKeyWithValue("role", client.UserRole))
							Expect(msg1).To(HaveKeyWithValue("content", query))

							// Temperature / top_p assertions are capability-driven now
							caps := client.GetCapabilities(m)

							if caps.SupportsTemperature {
								Expect(req).To(HaveKeyWithValue("temperature", BeNumerically("==", config.Temperature)))
							} else {
								Expect(req).NotTo(HaveKey("temperature"))
							}

							if caps.SupportsTopP {
								Expect(req).To(HaveKeyWithValue("top_p", BeNumerically("==", config.TopP)))
							} else {
								Expect(req).NotTo(HaveKey("top_p"))
							}
						}

						it("returns the output_text when present", func() {
							subject := factory.buildClientWithoutConfig()
							subject.Config.Model = m
							subject.Config.Role = systemRole

							answer := "yes, it does"

							mockTimer.EXPECT().Now().Times(3)
							mockHistoryStore.EXPECT().Write(gomock.Any())

							response := api.ResponsesResponse{
								Output: []api.Output{{
									Type:    "message",
									Content: []api.Content{{Type: "output_text", Text: answer}},
								}},
								Usage: api.TokenUsage{TotalTokens: 42},
							}
							raw, _ := json.Marshal(response)

							mockCaller.EXPECT().
								Post(subject.Config.URL+"/v1/responses", gomock.Any(), false).
								DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
									assertResponsesRequest(body)
									return raw, nil
								})

							text, tokens, err := subject.Query(context.Background(), query)
							Expect(err).NotTo(HaveOccurred())
							Expect(text).To(Equal(answer))
							Expect(tokens).To(Equal(42))
						})

						it("errors when no output blocks are present", func() {
							subject := factory.buildClientWithoutConfig()
							subject.Config.Model = m
							subject.Config.Role = systemRole

							mockTimer.EXPECT().Now().Times(2)

							response := api.ResponsesResponse{
								Output: []api.Output{},
								Usage:  api.TokenUsage{TotalTokens: totalTokens},
							}
							raw, _ := json.Marshal(response)

							mockCaller.EXPECT().
								Post(subject.Config.URL+"/v1/responses", gomock.Any(), false).
								DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
									assertResponsesRequest(body)
									return raw, nil
								})

							_, _, err := subject.Query(context.Background(), query)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(Equal("no response returned"))
						})

						it("errors when message has no output_text", func() {
							subject := factory.buildClientWithoutConfig()
							subject.Config.Model = m
							subject.Config.Role = systemRole

							mockTimer.EXPECT().Now().Times(2)

							response := api.ResponsesResponse{
								Output: []api.Output{{
									Type:    "message",
									Content: []api.Content{{Type: "refusal", Text: "nope"}},
								}},
								Usage: api.TokenUsage{TotalTokens: totalTokens},
							}
							raw, _ := json.Marshal(response)

							mockCaller.EXPECT().
								Post(subject.Config.URL+"/v1/responses", gomock.Any(), false).
								DoAndReturn(func(_ string, body []byte, _ bool) ([]byte, error) {
									assertResponsesRequest(body)
									return raw, nil
								})

							_, _, err := subject.Query(context.Background(), query)
							Expect(err).To(HaveOccurred())
							Expect(err.Error()).To(Equal("no response returned"))
						})
					})
				}
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
				mockCaller.EXPECT().
					Post(subject.Config.URL+subject.Config.CompletionsPath, body, true).
					Return(nil, errors.New(errorMsg))

				mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

				err := subject.Stream(context.Background(), query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(errorMsg))
			})

			it("errors when the model is realtime (no HTTP call is made)", func() {
				factory.withoutHistory()
				subject := factory.buildClientWithoutConfig()

				realtimeModel := "gpt-4o-realtime-preview"
				subject.Config.Model = realtimeModel
				config.Model = realtimeModel

				mockCaller.EXPECT().
					Post(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)

				mockTimer.EXPECT().Now().Return(time.Time{}).Times(2)

				err := subject.Stream(context.Background(), query)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("realtime"))
			})

			when("a valid http response is received", func() {
				const answer = "answer"

				testValidHTTPResponse := func(subject *client.Client, hs []history.History, expectedBody []byte) {
					messages = createMessages(nil, query)
					body, err = createBody(messages, true)
					Expect(err).NotTo(HaveOccurred())

					mockCaller.EXPECT().
						Post(subject.Config.URL+subject.Config.CompletionsPath, expectedBody, true).
						Return([]byte(answer), nil)

					mockTimer.EXPECT().Now().Return(time.Time{}).AnyTimes()

					messages = createMessages(hs, query)

					var out []history.History
					for _, message := range messages {
						out = append(out, history.History{Message: message})
					}

					mockHistoryStore.EXPECT().Write(append(out, history.History{
						Message: api.Message{
							Role:    client.AssistantRole,
							Content: answer,
						},
					}))

					err := subject.Stream(context.Background(), query)
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
						{Message: api.Message{Role: client.SystemRole, Content: config.Role}},
						{Message: api.Message{Role: client.UserRole, Content: "question x"}},
						{Message: api.Message{Role: client.AssistantRole, Content: "answer x"}},
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
				mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).
					Return(nil, errors.New(errorMsg))

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
				mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).
					Return([]byte(malformed), nil)

				_, err := subject.ListModels()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).Should(HavePrefix("failed to decode response:"))
			})

			it("filters gpt and o1 models as expected and puts them in alphabetical order", func() {
				subject := factory.buildClientWithoutConfig()

				response, err := test.FileToBytes("models.json")
				Expect(err).NotTo(HaveOccurred())

				mockCaller.EXPECT().Get(subject.Config.URL+subject.Config.ModelsPath).
					Return(response, nil)

				result, err := subject.ListModels()
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeEmpty())
				Expect(result).To(HaveLen(5))
				Expect(result[0]).To(Equal("- gpt-3.5-env-model"))
				Expect(result[1]).To(Equal("* gpt-3.5-turbo (current)"))
				Expect(result[2]).To(Equal("- gpt-3.5-turbo-0301"))
				Expect(result[3]).To(Equal("- gpt-4o"))
				Expect(result[4]).To(Equal("- o1-mini"))
			})
		})
	})
}

func testCapabilities(t *testing.T, when spec.G, it spec.S) {
	when("GetCapabilities()", func() {
		type tc struct {
			model string

			supportsTemp      bool
			supportsTopP      bool
			usesResponses     bool
			omitFirstSystem   bool
			supportsStreaming bool
			isRealtime        bool
			supportsWebSearch bool
		}

		tests := []tc{
			{
				model:             "gpt-4o",
				supportsTemp:      true,
				supportsTopP:      true,
				usesResponses:     false,
				omitFirstSystem:   false,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: false,
			},
			{
				model:             "gpt-4o-search-preview",
				supportsTemp:      false,
				supportsTopP:      false,
				usesResponses:     false, // still completions path
				omitFirstSystem:   false,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: false,
			},
			{
				model:      "gpt-realtime",
				isRealtime: true,
			},
			{
				model:             "gpt-5",
				supportsTemp:      true,
				supportsTopP:      false,
				usesResponses:     true,
				omitFirstSystem:   false,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: true,
			},
			{
				model:             "gpt-5-search",
				supportsTemp:      false,
				supportsTopP:      false,
				usesResponses:     true,
				omitFirstSystem:   false,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: false,
			},
			{
				model:             "gpt-5.2",
				supportsTemp:      true,
				supportsTopP:      false,
				usesResponses:     true,
				omitFirstSystem:   false,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: true,
			},
			{
				model:             "gpt-5.2-pro",
				supportsTemp:      true,
				supportsTopP:      false,
				usesResponses:     true,
				omitFirstSystem:   false,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: true,
			},
			{
				model:             "o1-mini",
				supportsTemp:      true,
				supportsTopP:      true,
				usesResponses:     false,
				omitFirstSystem:   true,
				supportsStreaming: true,
				isRealtime:        false,
				supportsWebSearch: false,
			},
			{
				model:             "o1-pro",
				supportsTemp:      true,
				supportsTopP:      true,
				usesResponses:     true,
				omitFirstSystem:   false,
				supportsStreaming: false,
				isRealtime:        false,
				supportsWebSearch: false,
			},
		}

		for _, tt := range tests {
			tt := tt
			it(tt.model, func() {
				RegisterTestingT(t)

				c := client.GetCapabilities(tt.model)

				Expect(c.IsRealtime).To(Equal(tt.isRealtime))

				// Only assert these for non-realtime models.
				if !tt.isRealtime {
					Expect(c.SupportsTemperature).To(Equal(tt.supportsTemp))
					Expect(c.SupportsTopP).To(Equal(tt.supportsTopP))
					Expect(c.UsesResponsesAPI).To(Equal(tt.usesResponses))
					Expect(c.OmitFirstSystemMsg).To(Equal(tt.omitFirstSystem))
					Expect(c.SupportsStreaming).To(Equal(tt.supportsStreaming))
					Expect(c.SupportsWebSearch).To(Equal(tt.supportsWebSearch))
				}
			})
		}
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
