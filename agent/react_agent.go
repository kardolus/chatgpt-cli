package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

type ReActAgent struct {
	*BaseAgent
	llm    LLM
	runner Runner
	budget Budget
}

type ReActOption func(*ReActAgent)

func WithReActDryRun(v bool) ReActOption { return func(a *ReActAgent) { a.config.DryRun = v } }
func WithReActWorkDir(d string) ReActOption {
	return func(a *ReActAgent) { a.config.WorkDir = d }
}

func WithReActHumanLogger(l *zap.SugaredLogger) ReActOption {
	return func(a *ReActAgent) {
		if l != nil {
			a.out = l
		}
	}
}

func WithReActDebugLogger(l *zap.SugaredLogger) ReActOption {
	return func(a *ReActAgent) {
		if l != nil {
			a.debug = l
		}
	}
}

func NewReActAgent(llm LLM, runner Runner, budget Budget, clock Clock, opts ...ReActOption) *ReActAgent {
	a := &ReActAgent{
		BaseAgent: NewBaseAgent(clock),
		llm:       llm,
		runner:    runner,
		budget:    budget,
	}
	for _, o := range opts {
		o(a)
	}
	return a
}

type reActAction struct {
	Thought     string   `json:"thought"`
	ActionType  string   `json:"action_type"` // "tool" or "answer"
	Tool        string   `json:"tool,omitempty"`
	Command     string   `json:"command,omitempty"`
	Args        []string `json:"args,omitempty"`
	Prompt      string   `json:"prompt,omitempty"`
	Op          string   `json:"op,omitempty"`
	Path        string   `json:"path,omitempty"`
	Data        string   `json:"data,omitempty"`
	FinalAnswer string   `json:"final_answer,omitempty"`
}

func (a *ReActAgent) RunAgentGoal(ctx context.Context, goal string) (string, error) {
	start := a.startTimer()
	defer a.finishTimer(start)

	a.logMode(goal, "ReAct (iterative reasoning + acting)")

	out := a.out
	dbg := a.debug

	conversation := []string{
		fmt.Sprintf("USER: %s", goal),
	}

	for i := 0; ; i++ {
		now := a.clock.Now()

		if err := a.budget.AllowIteration(now); err != nil {
			dbg.Errorf("iteration budget exceeded at iteration %d: %v", i+1, err)
			return "", err
		}

		snap := a.budget.Snapshot(now)
		if snap.Limits.MaxLLMTokens > 0 && snap.LLMTokensUsed >= snap.Limits.MaxLLMTokens {
			return "", BudgetExceededError{
				Kind:    BudgetKindLLMTokens,
				Limit:   snap.Limits.MaxLLMTokens,
				Used:    snap.LLMTokensUsed,
				Message: "llm token budget exceeded",
			}
		}

		if err := a.budget.AllowTool(ToolLLM, now); err != nil {
			dbg.Errorf("budget exceeded at iteration %d: %v", i+1, err)
			return "", err
		}

		prompt := buildReActPrompt(conversation)
		dbg.Debugf("react iteration %d prompt_len=%d", i+1, len(prompt))

		raw, tokens, err := a.llm.Complete(ctx, prompt)
		if err != nil {
			dbg.Errorf("llm error at iteration %d: %v", i+1, err)
			return "", err
		}

		a.budget.ChargeLLMTokens(tokens, now)
		dbg.Debugf("react iteration %d tokens=%d", i+1, tokens)

		action, err := parseReActResponse(raw)
		if err != nil {
			out.Errorf("Failed to parse ReAct response: %v", err)
			dbg.Errorf("parse error at iteration %d: %v\nraw: %s", i+1, err, raw)
			return "", err
		}

		dbg.Debugf("react iteration %d action_type=%s thought=%q", i+1, action.ActionType, action.Thought)

		if action.Thought != "" {
			out.Infof("[Iteration %d] Thought: %s", i+1, action.Thought)
		}

		if action.ActionType == "answer" {
			result := strings.TrimRightFunc(action.FinalAnswer, unicode.IsSpace)
			out.Infof("\nResult: %s\n", result)
			dbg.Debugf("final answer: %q", action.FinalAnswer)
			return result, nil
		}

		if action.ActionType != "tool" {
			err := fmt.Errorf("unknown action_type: %q", action.ActionType)
			dbg.Errorf("unknown action_type at iteration %d: %v", i+1, err)
			return "", err
		}

		step, err := convertReActActionToStep(action)
		if err != nil {
			out.Errorf("Failed to convert action to step: %v", err)
			dbg.Errorf("convert error at iteration %d: %v", i+1, err)
			return "", err
		}

		out.Infof("[Iteration %d] Action: %s %s", i+1, action.Tool, step.Description)

		res, err := a.runner.RunStep(ctx, a.config, step)
		if err != nil {
			if isBudgetStop(err, out) || isPolicyStop(err, out) {
				dbg.Errorf("stop error at iteration %d: %v", i+1, err)
				return "", err
			}
			out.Errorf("Step failed: %s: %v", step.Description, err)
			dbg.Errorf("step failed at iteration %d: %v transcript=%q", i+1, err, res.Transcript)
			return "", err
		}

		out.Infof("[Iteration %d] Observation: %s (took %s)", i+1, truncateForDisplay(res.Output, 100), res.Duration)
		dbg.Debugf("observation (iteration %d): %q", i+1, res.Output)

		conversation = append(conversation, fmt.Sprintf("OBSERVATION: %s", res.Output))
	}
}

func buildReActPrompt(conversation []string) string {
	history := strings.Join(conversation, "\n\n")

	return fmt.Sprintf(`You are a ReAct agent. You will iteratively reason and act to answer the user's question.

You have access to these tools:

1. shell - Execute shell commands
   Fields: "command" (string), "args" (array of strings)

2. llm - Request reasoning or summarization
   Fields: "prompt" (string)

3. file - Read or write files
   Fields: "op" ("read" or "write"), "path" (string), "data" (string for write)

At each step, respond with ONLY valid JSON in this format:

FOR USING A TOOL:
{
  "thought": "your reasoning about what to do next",
  "action_type": "tool",
  "tool": "shell" | "llm" | "file",
  "command": "...",
  "args": [...],
  "prompt": "...",
  "op": "read" | "write",
  "path": "...",
  "data": "..."
}

FOR FINAL ANSWER:
{
  "thought": "your reasoning about the answer",
  "action_type": "answer",
  "final_answer": "your complete answer to the user"
}

CRITICAL RULES:
- Return ONLY raw JSON, no markdown, no code fences
- First character must be '{'
- Last character must be '}'
- Include only fields relevant to your chosen tool
- Keep thoughts concise
- When you have enough information to answer the user's question, use action_type: "answer"

Conversation history:

%s

What's your next step?`, history)
}

func parseReActResponse(raw string) (reActAction, error) {
	raw = cleanReActOutput(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return reActAction{}, errors.New("empty response from LLM")
	}

	var action reActAction
	if err := json.Unmarshal([]byte(raw), &action); err != nil {
		return reActAction{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	action.Thought = strings.TrimSpace(action.Thought)
	action.ActionType = strings.ToLower(strings.TrimSpace(action.ActionType))

	if action.ActionType == "" {
		return reActAction{}, errors.New("missing action_type")
	}

	if action.ActionType == "answer" {
		if strings.TrimSpace(action.FinalAnswer) == "" {
			return reActAction{}, errors.New("action_type=answer but final_answer is empty")
		}
		return action, nil
	}

	if action.ActionType == "tool" {
		action.Tool = strings.ToLower(strings.TrimSpace(action.Tool))
		if action.Tool == "" {
			return reActAction{}, errors.New("action_type=tool but tool field is empty")
		}
		return action, nil
	}

	return reActAction{}, fmt.Errorf("invalid action_type: %q", action.ActionType)
}

func cleanReActOutput(raw string) string {
	raw = strings.TrimSpace(raw)

	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSpace(raw)

		if i := strings.IndexByte(raw, '\n'); i != -1 {
			firstLine := strings.ToLower(strings.TrimSpace(raw[:i]))
			if firstLine == "json" || firstLine == "application/json" {
				raw = raw[i+1:]
			}
		}

		raw = strings.TrimSpace(raw)
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)
	}

	return raw
}

func convertReActActionToStep(action reActAction) (Step, error) {
	switch action.Tool {
	case "shell":
		cmd := strings.TrimSpace(action.Command)
		if cmd == "" {
			return Step{}, errors.New("shell tool requires command")
		}
		return Step{
			Type:        ToolShell,
			Description: fmt.Sprintf("Execute: %s %v", cmd, action.Args),
			Command:     cmd,
			Args:        action.Args,
		}, nil

	case "llm":
		prompt := strings.TrimSpace(action.Prompt)
		if prompt == "" {
			return Step{}, errors.New("llm tool requires prompt")
		}
		return Step{
			Type:        ToolLLM,
			Description: "LLM reasoning",
			Prompt:      prompt,
		}, nil

	case "file":
		op := strings.ToLower(strings.TrimSpace(action.Op))
		path := strings.TrimSpace(action.Path)
		if op == "" {
			return Step{}, errors.New("file tool requires op")
		}
		if path == "" {
			return Step{}, errors.New("file tool requires path")
		}
		return Step{
			Type:        ToolFiles,
			Description: fmt.Sprintf("File %s: %s", op, path),
			Op:          op,
			Path:        path,
			Data:        action.Data,
		}, nil

	default:
		return Step{}, fmt.Errorf("unknown tool: %q", action.Tool)
	}
}

func truncateForDisplay(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
