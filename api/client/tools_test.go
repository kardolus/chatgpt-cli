package client_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kardolus/chatgpt-cli/api"
	"github.com/kardolus/chatgpt-cli/api/client"
	"github.com/kardolus/chatgpt-cli/api/http"
	config2 "github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

// --- test doubles -----------------------------------------------------------

// scriptCaller returns queued Post responses in order and records the request
// bodies it was given, so multi-turn tool round-trips can be scripted.
type scriptCaller struct {
	responses [][]byte
	errs      []error
	bodies    [][]byte
	calls     int
}

func (s *scriptCaller) Post(_ context.Context, _ string, body []byte, _ bool) ([]byte, error) {
	s.bodies = append(s.bodies, body)
	i := s.calls
	s.calls++
	var err error
	if i < len(s.errs) {
		err = s.errs[i]
	}
	if i < len(s.responses) {
		return s.responses[i], err
	}
	return nil, err
}
func (s *scriptCaller) PostWithHeaders(_ context.Context, _ string, _ []byte, _ map[string]string) ([]byte, error) {
	return nil, nil
}
func (s *scriptCaller) Get(_ context.Context, _ string) ([]byte, error) { return nil, nil }
func (s *scriptCaller) PostWithHeadersResponse(_ context.Context, _ string, _ []byte, _ map[string]string) (api.HTTPResponse, error) {
	return api.HTTPResponse{}, nil
}

type stubExec struct {
	defs    []api.FunctionTool
	results []string
	errs    []error
	calls   []stubCall
}
type stubCall struct{ name, args string }

func (e *stubExec) Definitions(context.Context) ([]api.FunctionTool, error) { return e.defs, nil }
func (e *stubExec) Execute(_ context.Context, name, args string) (string, error) {
	i := len(e.calls)
	e.calls = append(e.calls, stubCall{name, args})
	var err error
	if i < len(e.errs) {
		err = e.errs[i]
	}
	res := "ok"
	if i < len(e.results) {
		res = e.results[i]
	}
	return res, err
}

type nopStore struct{}

func (nopStore) Read() ([]history.History, error)             { return nil, nil }
func (nopStore) ReadThread(string) ([]history.History, error) { return nil, nil }
func (nopStore) Write([]history.History) error                { return nil }
func (nopStore) SetThread(string)                             {}
func (nopStore) GetThread() string                            { return "" }

type zeroTimer struct{}

func (zeroTimer) Now() time.Time { return time.Time{} }

func buildToolClient(caller http.Caller, exec client.ToolExecutor, cfg config2.Config) *client.Client {
	c := client.New(
		func(config2.Config) http.Caller { return caller },
		nopStore{},
		zeroTimer{},
		fsio.NewRealReader(fsio.DefaultBufferSize),
		&fsio.RealWriter{},
		cfg,
	)
	if exec != nil {
		c = c.WithToolExecutor(exec)
	}
	return c
}

// canned responses
const (
	toolCallResp = `{"choices":[{"message":{"role":"assistant","content":null,` +
		`"tool_calls":[{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"NYC\"}"}}]},` +
		`"finish_reason":"tool_calls"}],"usage":{"total_tokens":10}}`
	finalResp = `{"choices":[{"message":{"role":"assistant","content":"It is sunny."}}],"usage":{"total_tokens":5}}`
)

func weatherTool() api.FunctionTool {
	return api.FunctionTool{
		Type:     "function",
		Function: api.FunctionDef{Name: "get_weather", Description: "get weather", Parameters: json.RawMessage(`{"type":"object"}`)},
	}
}

func TestUnitFunctionCalling(t *testing.T) {
	spec.Run(t, "Testing function calling + JSON mode", testFunctionCalling, spec.Report(report.Terminal{}))
}

func testFunctionCalling(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() { RegisterTestingT(t) })

	baseCfg := func() config2.Config {
		return config2.Config{Model: "gpt-4o", OmitHistory: true}
	}

	when("JSON mode (response_format)", func() {
		it("sends json_object when configured", func() {
			cfg := baseCfg()
			cfg.ResponseFormat = "json_object"
			caller := &scriptCaller{responses: [][]byte{[]byte(finalResp)}}

			_, _, err := buildToolClient(caller, nil, cfg).Query(context.Background(), "hi")
			Expect(err).NotTo(HaveOccurred())

			var req api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[0], &req)).To(Succeed())
			Expect(req.ResponseFormat).NotTo(BeNil())
			Expect(req.ResponseFormat.Type).To(Equal("json_object"))
		})

		it("wraps a JSON schema as json_schema", func() {
			cfg := baseCfg()
			cfg.ResponseFormat = `{"type":"object","properties":{"x":{"type":"number"}}}`
			caller := &scriptCaller{responses: [][]byte{[]byte(finalResp)}}

			_, _, err := buildToolClient(caller, nil, cfg).Query(context.Background(), "hi")
			Expect(err).NotTo(HaveOccurred())

			var req api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[0], &req)).To(Succeed())
			Expect(req.ResponseFormat).NotTo(BeNil())
			Expect(req.ResponseFormat.Type).To(Equal("json_schema"))
			Expect(string(req.ResponseFormat.JSONSchema)).To(ContainSubstring(`"strict":false`))
			Expect(string(req.ResponseFormat.JSONSchema)).To(ContainSubstring(`"schema"`))
		})

		it("errors on an invalid schema", func() {
			cfg := baseCfg()
			cfg.ResponseFormat = "not json{"
			caller := &scriptCaller{responses: [][]byte{[]byte(finalResp)}}

			_, _, err := buildToolClient(caller, nil, cfg).Query(context.Background(), "hi")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("json_object"))
		})

		it("omits response_format when unset", func() {
			caller := &scriptCaller{responses: [][]byte{[]byte(finalResp)}}
			_, _, err := buildToolClient(caller, nil, baseCfg()).Query(context.Background(), "hi")
			Expect(err).NotTo(HaveOccurred())
			var req api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[0], &req)).To(Succeed())
			Expect(req.ResponseFormat).To(BeNil())
		})
	})

	when("function-calling round-trip", func() {
		it("advertises tools, dispatches the call, feeds the result back, and returns the final answer", func() {
			cfg := baseCfg()
			cfg.Tools = true
			exec := &stubExec{defs: []api.FunctionTool{weatherTool()}, results: []string{"sunny, 75F"}}
			caller := &scriptCaller{responses: [][]byte{[]byte(toolCallResp), []byte(finalResp)}}

			c := buildToolClient(caller, exec, cfg)
			out, tokens, err := c.Query(context.Background(), "weather in NYC?")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("It is sunny."))
			Expect(tokens).To(Equal(15)) // 10 + 5 across both rounds

			// The ephemeral tool-round messages must NOT persist: history is just
			// the user turn + the final answer (no orphan-able role:"tool" msgs).
			var roles []string
			for _, h := range c.History {
				roles = append(roles, h.Message.Role)
			}
			Expect(roles).To(Equal([]string{"user", "assistant"}))

			// The tool was dispatched with the model's name + args.
			Expect(exec.calls).To(HaveLen(1))
			Expect(exec.calls[0].name).To(Equal("get_weather"))
			Expect(exec.calls[0].args).To(Equal(`{"city":"NYC"}`))

			// Two requests were made.
			Expect(caller.bodies).To(HaveLen(2))

			// Request 1 advertises the tool with tool_choice=auto.
			var req0 api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[0], &req0)).To(Succeed())
			Expect(req0.Tools).To(HaveLen(1))
			Expect(req0.Tools[0].Function.Name).To(Equal("get_weather"))
			Expect(req0.ToolChoice).To(Equal("auto"))

			// Request 2 carries the assistant tool-call turn + the tool result.
			var req1 api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[1], &req1)).To(Succeed())
			var sawToolResult bool
			for _, m := range req1.Messages {
				if m.Role == "tool" && m.ToolCallID == "call_1" {
					sawToolResult = true
					Expect(m.Content).To(Equal("sunny, 75F"))
				}
			}
			Expect(sawToolResult).To(BeTrue())
		})

		it("does not advertise tools when the config flag is off", func() {
			cfg := baseCfg() // Tools=false
			exec := &stubExec{defs: []api.FunctionTool{weatherTool()}}
			caller := &scriptCaller{responses: [][]byte{[]byte(finalResp)}}

			_, _, err := buildToolClient(caller, exec, cfg).Query(context.Background(), "hi")
			Expect(err).NotTo(HaveOccurred())
			var req api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[0], &req)).To(Succeed())
			Expect(req.Tools).To(BeEmpty())
			Expect(exec.calls).To(BeEmpty())
		})

		it("feeds an execution error back to the model instead of aborting", func() {
			cfg := baseCfg()
			cfg.Tools = true
			exec := &stubExec{defs: []api.FunctionTool{weatherTool()}, errs: []error{context.DeadlineExceeded}}
			caller := &scriptCaller{responses: [][]byte{[]byte(toolCallResp), []byte(finalResp)}}

			out, _, err := buildToolClient(caller, exec, cfg).Query(context.Background(), "hi")
			Expect(err).NotTo(HaveOccurred())
			Expect(out).To(Equal("It is sunny."))

			var req1 api.CompletionsRequest
			Expect(json.Unmarshal(caller.bodies[1], &req1)).To(Succeed())
			var toolMsg string
			for _, m := range req1.Messages {
				if m.Role == "tool" {
					toolMsg, _ = m.Content.(string)
				}
			}
			Expect(toolMsg).To(ContainSubstring("error executing tool"))
		})

		it("bounds the number of tool-call rounds", func() {
			cfg := baseCfg()
			cfg.Tools = true
			cfg.MaxToolCalls = 2
			exec := &stubExec{defs: []api.FunctionTool{weatherTool()}}
			// Always returns a tool call -> would loop forever without the bound.
			caller := &scriptCaller{responses: [][]byte{
				[]byte(toolCallResp), []byte(toolCallResp), []byte(toolCallResp), []byte(toolCallResp),
			}}

			_, _, err := buildToolClient(caller, exec, cfg).Query(context.Background(), "hi")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("max tool-call rounds"))
		})
	})

	when("MCP tool bridge", func() {
		it("advertises tools/list as function definitions", func() {
			tr := &fakeTransport{listResult: json.RawMessage(
				`{"tools":[{"name":"get_weather","description":"gets weather","inputSchema":{"type":"object","properties":{"city":{"type":"string"}}}}]}`)}

			defs, err := client.NewMCPToolExecutor(tr, "http://mcp", nil).Definitions(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(defs).To(HaveLen(1))
			Expect(defs[0].Type).To(Equal("function"))
			Expect(defs[0].Function.Name).To(Equal("get_weather"))
			Expect(defs[0].Function.Description).To(Equal("gets weather"))
			Expect(string(defs[0].Function.Parameters)).To(ContainSubstring("city"))
			Expect(tr.lastCall.Method).To(Equal("tools/list"))
		})

		it("defaults an object schema when a tool omits inputSchema", func() {
			tr := &fakeTransport{listResult: json.RawMessage(`{"tools":[{"name":"ping"}]}`)}
			defs, err := client.NewMCPToolExecutor(tr, "http://mcp", nil).Definitions(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(string(defs[0].Function.Parameters)).To(Equal(`{"type":"object"}`))
		})

		it("dispatches a tool call via tools/call and returns the text result", func() {
			tr := &fakeTransport{callResult: json.RawMessage(`{"content":[{"type":"text","text":"sunny, 75F"}]}`)}

			out, err := client.NewMCPToolExecutor(tr, "http://mcp", nil).
				Execute(context.Background(), "get_weather", `{"city":"NYC"}`)
			Expect(err).NotTo(HaveOccurred())
			// Raw text result, no decorative "[MCP: ...]" prefix.
			Expect(out).To(Equal("sunny, 75F"))

			Expect(tr.lastCall.Method).To(Equal("tools/call"))
			Expect(string(tr.lastCall.Params)).To(ContainSubstring(`"name":"get_weather"`))
			Expect(string(tr.lastCall.Params)).To(ContainSubstring(`"city":"NYC"`))
		})

		it("coerces empty/invalid/non-object arguments to an empty object", func() {
			for _, args := range []string{"", "not json", "[1,2,3]", `"scalar"`, "42"} {
				tr := &fakeTransport{callResult: json.RawMessage(`{"content":[{"type":"text","text":"ok"}]}`)}
				_, err := client.NewMCPToolExecutor(tr, "http://mcp", nil).
					Execute(context.Background(), "ping", args)
				Expect(err).NotTo(HaveOccurred(), args)
				Expect(string(tr.lastCall.Params)).To(ContainSubstring(`"arguments":{}`), args)
			}
		})
	})
}

// fakeTransport is an in-memory MCPTransport returning canned tools/list and
// tools/call results based on the request method.
type fakeTransport struct {
	listResult json.RawMessage
	callResult json.RawMessage
	lastCall   api.MCPMessage
}

func (f *fakeTransport) Call(_ string, req api.MCPMessage, _ map[string]string) (api.MCPResponse, error) {
	f.lastCall = req
	var result json.RawMessage
	switch req.Method {
	case "tools/list":
		result = f.listResult
	case "tools/call":
		result = f.callResult
	}
	return api.MCPResponse{Message: api.MCPMessage{Result: result}}, nil
}
