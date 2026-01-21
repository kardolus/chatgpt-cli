package react

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kardolus/chatgpt-cli/agent/core"
	"github.com/kardolus/chatgpt-cli/agent/tools"
	"github.com/kardolus/chatgpt-cli/agent/types"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type ReActAgent struct {
	*core.BaseAgent
	LLM      tools.LLM
	Runner   core.Runner
	Budget   core.Budget
	effects  types.Effects
	llmCalls int
}

func NewReActAgent(llm tools.LLM, runner core.Runner, budget core.Budget, clock core.Clock, opts ...core.BaseOption) *ReActAgent {
	base := core.NewBaseAgent(clock)
	for _, o := range opts {
		o(base)
	}

	return &ReActAgent{
		BaseAgent: base,
		LLM:       llm,
		Runner:    runner,
		Budget:    budget,
	}
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
	Old         string   `json:"old,omitempty"`
	New         string   `json:"new,omitempty"`
	N           int      `json:"n,omitempty"`
	FinalAnswer string   `json:"final_answer,omitempty"`
}

func (a *ReActAgent) RunAgentGoal(ctx context.Context, goal string) (string, error) {
	start := a.StartTimer()
	defer a.FinishTimer(start)

	a.effects = nil
	a.llmCalls = 0

	guard := newRepetitionGuard(32)

	parseRecoveries := 0
	const maxParseRecoveries = 3

	a.LogMode(goal, "ReAct (iterative reasoning + acting)")

	out := a.Out
	dbg := a.Debug

	conversation := []string{
		fmt.Sprintf("USER: %s", goal),
	}

	for i := 0; ; i++ {
		now := a.Clock.Now()

		if err := a.Budget.AllowIteration(now); err != nil {
			dbg.Errorf("iteration Budget exceeded at iteration %d: %v", i+1, err)
			return "", err
		}

		snap := a.Budget.Snapshot(now)
		if snap.Limits.MaxLLMTokens > 0 && snap.LLMTokensUsed >= snap.Limits.MaxLLMTokens {
			return "", core.BudgetExceededError{
				Kind:    core.BudgetKindLLMTokens,
				Limit:   snap.Limits.MaxLLMTokens,
				Used:    snap.LLMTokensUsed,
				Message: "LLM token Budget exceeded",
			}
		}

		if err := a.Budget.AllowTool(types.ToolLLM, now); err != nil {
			dbg.Errorf("Budget exceeded at iteration %d: %v", i+1, err)
			return "", err
		}

		prompt := buildReActPrompt(conversation, a.promptStateLine())
		dbg.Debugf("react iteration %d prompt_len=%d", i+1, len(prompt))

		a.llmCalls++
		raw, tokens, err := a.LLM.Complete(ctx, prompt)
		if err != nil {
			dbg.Errorf("LLM error at iteration %d: %v", i+1, err)
			return "", err
		}

		a.Budget.ChargeLLMTokens(tokens, now)
		dbg.Debugf("react iteration %d tokens=%d", i+1, tokens)

		action, err := parseReActResponse(raw)
		if err != nil {
			out.Errorf("Failed to parse ReAct response: %v", err)
			dbg.Errorf("parse error at iteration %d: %v\nraw: %s", i+1, err, raw)

			parseRecoveries++
			if parseRecoveries > maxParseRecoveries {
				return "", fmt.Errorf("agent failed to produce valid JSON after %d attempts: %w", maxParseRecoveries, err)
			}

			rawTrim := strings.TrimSpace(raw)
			rawSnippet := rawTrim
			if len(rawSnippet) > 200 {
				rawSnippet = rawSnippet[:200] + "..."
			}

			conversation = append(conversation,
				"ACTION_TAKEN: tool=LLM details=INVALID_RESPONSE",
				fmt.Sprintf("OBSERVATION: ERROR: Your last response violated the ReAct protocol (%s). You MUST reply with EXACTLY ONE JSON object. The first non-whitespace character must be '{' and the last must be '}'. Include \"action_type\". Do not include any prose.", err.Error()),
				fmt.Sprintf("OBSERVATION: ERROR: Raw response (truncated): %q", rawSnippet),
			)
			continue
		}

		parseRecoveries = 0

		if action.ActionType == "tool" {
			sig := signatureForAction(action)

			immediate := guard.isImmediateRepeat(sig)
			seen := guard.count(sig) // how many times seen so far (before recording this attempt)

			if immediate || seen >= 3 {
				// Record the repeated attempt so counts can climb and we can hard-stop.
				guard.observe(sig)

				seenNow := guard.count(sig) // after recording this attempt

				msg := fmt.Sprintf(
					"OBSERVATION: You are repeating the same tool call (%s %q). Do NOT repeat it. "+
						"Choose a different next step (e.g., write the file, narrow the read range, or answer).",
					sig.tool, sig.key,
				)
				conversation = append(conversation, msg)
				dbg.Debugf("repetition guard injected: %s", msg)

				// Hard-stop if it's *really* stuck (e.g., 6+ occurrences in the rolling window)
				if seenNow >= 6 {
					return "", fmt.Errorf("agent appears stuck: repeated tool call too many times: %s %q", sig.tool, sig.key)
				}

				continue
			}

			// Non-repeat path: record it normally
			guard.observe(sig)
		}
		dbg.Debugf("react iteration %d action_type=%s thought=%q", i+1, action.ActionType, action.Thought)

		if action.Thought != "" {
			out.Infof("[Iteration %d] Thought: %s", i+1, action.Thought)
		}

		if action.ActionType == "answer" {
			result := strings.TrimRightFunc(action.FinalAnswer, unicode.IsSpace)
			out.Infof("\nResult: %s\n", result)

			if len(a.effects) > 0 {
				out.Infof("Actions performed: %s", summarizeActionsForUI(a.effects, a.llmCalls))
			}
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
			conversation = append(conversation,
				fmt.Sprintf("ACTION_TAKEN: tool=%s details=INVALID_REQUEST", action.Tool),
				fmt.Sprintf("OBSERVATION: ERROR: %s", err.Error()),
			)
			continue
		}

		out.Infof("[Iteration %d] Action: %s %s", i+1, action.Tool, step.Description)

		res, err := a.Runner.RunStep(ctx, a.Config, step)

		// NOTE: stop errors still bubble out (Budget/policy).
		if err != nil {
			if core.IsBudgetStop(err, out) || core.IsPolicyStop(err, out) {
				dbg.Errorf("stop error at iteration %d: %v", i+1, err)
				return "", err
			}

			// Any other Runner error is treated as fatal (should be rare after Runner change).
			out.Errorf("Step failed: %s: %v", step.Description, err)
			dbg.Errorf("step failed at iteration %d: %v transcript=%q", i+1, err, res.Transcript)
			return "", err
		}

		if res.Outcome == types.OutcomeError {
			mergeEffects(&a.effects, res.Effects)

			out.Errorf("[Iteration %d] Step failed: %s", i+1, step.Description)
			out.Infof("[Iteration %d] Observation: %s (took %s)", i+1, truncateForDisplay(res.Output, 100), res.Duration)

			conversation = append(conversation,
				fmt.Sprintf("ACTION_TAKEN: tool=%s details=%s", action.Tool, step.Description),
				fmt.Sprintf("OBSERVATION: ERROR: %s", res.Output),
				formatEffectsForConversation(res.Effects),
			)

			if types.ToolKind(action.Tool) == types.ToolFiles && (step.Op == "patch" || step.Op == "replace") {
				// This is intentionally strict: stop the model from retrying patch/replace loops.
				conversation = append(conversation,
					fmt.Sprintf(
						"OBSERVATION: FALLBACK REQUIRED: The %s operation failed for %q. "+
							"Do NOT try op=%q or op=patch/replace again for this file. "+
							"Your NEXT step MUST be: {\"action_type\":\"tool\",\"tool\":\"file\",\"op\":\"read\",\"path\":%q}. "+
							"After reading, you MUST construct the FULL updated file contents and use op=\"write\" to overwrite the file.",
						step.Op, step.Path, step.Op, step.Path,
					),
				)
			}

			continue
		}

		mergeEffects(&a.effects, res.Effects)

		out.Infof("[Iteration %d] Observation: %s (took %s)", i+1, truncateForDisplay(res.Output, 100), res.Duration)
		dbg.Debugf("observation (iteration %d): %q", i+1, res.Output)

		conversation = append(conversation,
			fmt.Sprintf("ACTION_TAKEN: tool=%s details=%s", action.Tool, step.Description),
			fmt.Sprintf("OBSERVATION: %s", res.Output),
			formatEffectsForConversation(res.Effects),
		)
	}
}

func buildReActPrompt(conversation []string, stateLine string) string {
	history := strings.Join(conversation, "\n\n")
	stateLine = strings.TrimSpace(stateLine)

	return fmt.Sprintf(`You are a ReAct agent. You will iteratively reason and act to answer the user's question.

You have access to these tools:

1. shell - Execute shell commands
   Fields: "command" (string), "args" (array of strings)

2. llm - Request reasoning or summarization
   Fields: "prompt" (string)

3. file - Read or modify files
   Fields:
   - "op": "read" | "write" | "patch" | "replace"
   - "path": string

   For op="read":
   - returns ENTIRE file contents as text

   For op="write":
   - "data": string REQUIRED
   - OVERWRITES the ENTIRE file with exactly "data"

   For op="patch":
   - "data": string REQUIRED (unified diff)
   - Applies the unified diff to the file (no full rewrite needed if patch applies cleanly)

   For op="replace":
   - "old": string REQUIRED (pattern)
   - "new": string REQUIRED (replacement)
   - "n": int OPTIONAL
     - n <= 0 means replace all occurrences
     - n > 0 means replace first n occurrences

IMPORTANT FILE SEMANTICS:
- file op="read" returns the ENTIRE file contents as text.
- file op="write" OVERWRITES the ENTIRE file with exactly "data".
  It does NOT append. It does NOT merge. It replaces the whole file.
- Therefore: if you want to make a small change to an existing file, you MUST:
  1) read the file,
  2) construct the full updated contents (including unchanged parts),
  3) write the full updated contents back.
- Prefer op="replace" for small mechanical edits (rename, token swap).
- Prefer op="patch" when you have a correct unified diff.
- Fall back to read+write only if patch/replace fails or isn't applicable.

WRITE DEFAULT CONTENT RULE (CRITICAL):
- If the user asks to create a new file but does NOT specify what it should contain,
  you MUST still use file op="write" and you MUST include a non-empty "data" field.
- In that case, use EXACTLY one newline as the default content:
  "data": "\n"
  (This creates an empty-looking file but satisfies the non-empty data requirement.)
- Do NOT ask a follow-up question for content unless the user explicitly requests specific content.

PATCH FORMAT RULES (VERY IMPORTANT):
- For file op="patch", "data" MUST be a valid unified diff.
- The diff MUST use ONLY these line prefixes within hunks:
  - ' ' for context lines
  - '-' for deletions
  - '+' for insertions
  Any other prefix (including no prefix) will FAIL.
- Each hunk MUST start with a header like: @@ -oldStart,oldCount +newStart,newCount @@
- Include enough context lines (' ' lines) so the patch applies cleanly.
- When patching, DO NOT generate prose, explanations, or code fences—only diff text.

- PREFER-REPLACE RULE: If the change can be expressed as a simple string substitution, use op="replace" instead of op="patch".

NO-NEWLINE-AT-EOF RULE (CRITICAL FOR PATCHING):
- If the file content you read DOES NOT end with a newline, the last line is "no newline at end of file".
- If your patch changes or matches that last line, you MUST include the EXACT marker line:
  \ No newline at end of file
  immediately AFTER the affected '-' or '+' line in the diff.
- If your patch does NOT touch the last line, you do NOT need the marker line.

FILE TYPE RULE:
- Determine file type ONLY from the file extension in "path".
- Never infer format from file contents.
- Default to plain text if extension is unknown.
- Only use markdown syntax if:
  - path ends in ".md", OR
  - user explicitly asks for markdown formatting.

At each step, respond with ONLY valid JSON in this format:

FOR USING A TOOL:
{
  "thought": "your reasoning about what to do next",
  "action_type": "tool",
  "tool": "%s" | "%s" | "%s",

  // shell fields:
  "command": "...",
  "args": [...],

  // LLM fields:
  "prompt": "...",

  // file fields:
  "op": "read" | "write" | "patch" | "replace",
  "path": "...",

  // write/patch:
  "data": "...",  // REQUIRED for write and patch; MUST be non-empty for write

  // replace:
  "old": "...",   // REQUIRED for replace
  "new": "...",   // REQUIRED for replace
  "n": 0          // OPTIONAL for replace
}

FOR FINAL ANSWER:
{
  "thought": "your reasoning about the answer",
  "action_type": "answer",
  "final_answer": "your complete answer to the user"
}

CRITICAL RULES:
- Return ONLY raw JSON (no markdown, no code fences, no prose)
- Return EXACTLY ONE JSON object per response (not an array)
- Do NOT output multiple JSON objects back-to-back (no "}{" and no extra text before/after)
- One tool call per response. If multiple steps are needed, choose the NEXT single step only.
- First non-whitespace character must be '{' and the last non-whitespace character must be '}'
- You MUST include "action_type" in every response.
- Do NOT invent alternative schemas (e.g., {"text":...}, {"content":...}, {"result":...} are INVALID).
- Allowed top-level keys are STRICT:
  - For action_type="tool": thought, action_type, tool, command, args, prompt, op, path, data, old, new, n
  - For action_type="answer": thought, action_type, final_answer
  - No other top-level keys are permitted.
- Include only fields relevant to your chosen tool
- Keep "thought" concise
- When you have enough information to answer, respond with action_type="answer" and include "final_answer"

WRITE CONTENT RULE (CRITICAL):
- file op="write" is INVALID without a non-empty "data" field.
- If you cannot produce the full file contents yet, you must NOT call write.
  Instead, gather what you need first, then call write with complete contents.

FILE-DELIVERY RULE (CRITICAL):
- If the user asks you to write, save, put, or output anything into a file, you MUST do a file tool call with op="write" (or patch/replace for an existing file) BEFORE you respond with action_type="answer".
- Do NOT claim you created or wrote a file unless you actually executed a file tool step.
- If the user did not specify a filename, choose a reasonable one (e.g., "output.txt") and write to it.

NEW FILE RULE:
- To create a new file, use file op="write".
- patch/replace are only for modifying an existing file (after reading it, unless the user gave you exact old/new context).

ORDERING RULE:
- If a file tool call is required, it must happen in a step BEFORE any action_type="answer".
- The final answer may only reference files that were actually written or modified.

PROGRESS RULES:
- Never call the exact same tool+args twice in a row.
- After reading a file once, do not reread it unless you explain what NEW information you need.
- Prefer making the smallest safe change, but remember: writes overwrite the entire file.
- If you are stuck, finish with action_type:"answer" explaining what you need next.

COMPLETION RULE (CRITICAL)
- After a tool call succeeds and the user’s goal has been satisfied, your very next response MUST be a final JSON answer in this exact shape:

{
  "thought": "brief reasoning about completion",
  "action_type": "answer",
  "final_answer": "clear confirmation of what was done and any relevant result"
}

- Do NOT say things like:
  - "I’m ready to assist further."
  - "Let me know if you need anything else."
  - Any plain-text response outside JSON.

INVALID EXAMPLE (DO NOT DO THIS)

I’m ready to assist further if you have any new tasks or questions.

This is invalid because:
  - It is not JSON.
  - It does not include action_type.
  - It breaks the ReAct protocol.

State:

%s

Conversation history:

%s

What's your next step?`, types.ToolShell, types.ToolLLM, types.ToolFiles, stateLine, history)
}

func parseReActResponse(raw string) (reActAction, error) {
	raw = cleanReActOutput(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return reActAction{}, errors.New("empty response from LLM")
	}

	one, err := extractFirstJSONObject(raw)
	if err != nil {
		return reActAction{}, fmt.Errorf("failed to locate JSON object: %w", err)
	}

	var action reActAction
	if err := json.Unmarshal([]byte(one), &action); err != nil {
		return reActAction{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	action.Thought = strings.TrimSpace(action.Thought)
	action.ActionType = strings.ToLower(strings.TrimSpace(action.ActionType))
	action.Tool = strings.ToLower(strings.TrimSpace(action.Tool))

	if action.ActionType == "" {
		return reActAction{}, errors.New("missing action_type")
	}

	// Compatibility: allow shorthand like {"action_type":"file", ...}
	// This must work whether "tool" is also set.
	if action.ActionType != "tool" && action.ActionType != "answer" {
		switch action.ActionType {
		case "file", "shell", "llm":
			// If tool is empty OR matches the shorthand, normalize to canonical form.
			// (If tool is set to something else, we'll fall through to invalid action_type.)
			if action.Tool == "" || action.Tool == action.ActionType {
				action.Tool = action.ActionType
				action.ActionType = "tool"
			}
		}
	}

	if action.ActionType == "answer" {
		if strings.TrimSpace(action.FinalAnswer) == "" {
			return reActAction{}, errors.New("action_type=answer but final_answer is empty")
		}
		return action, nil
	}

	if action.ActionType == "tool" {
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

func convertReActActionToStep(action reActAction) (types.Step, error) {
	switch types.ToolKind(action.Tool) {
	case types.ToolShell:
		cmd := strings.TrimSpace(action.Command)
		if cmd == "" {
			return types.Step{}, errors.New("shell tool requires command")
		}
		return types.Step{
			Type:        types.ToolShell,
			Description: fmt.Sprintf("Execute: %s %v", cmd, action.Args),
			Command:     cmd,
			Args:        action.Args,
		}, nil

	case types.ToolLLM:
		prompt := strings.TrimSpace(action.Prompt)
		if prompt == "" {
			return types.Step{}, errors.New("LLM tool requires prompt")
		}
		return types.Step{
			Type:        types.ToolLLM,
			Description: "LLM reasoning",
			Prompt:      prompt,
		}, nil

	case types.ToolFiles:
		op := strings.ToLower(strings.TrimSpace(action.Op))
		path := strings.TrimSpace(action.Path)
		if op == "" {
			return types.Step{}, errors.New("file tool requires op")
		}
		if path == "" {
			return types.Step{}, errors.New("file tool requires path")
		}

		step := types.Step{
			Type:        types.ToolFiles,
			Description: fmt.Sprintf("File %s: %s", op, path),
			Op:          op,
			Path:        path,
			Data:        action.Data,
		}

		switch op {
		case "patch":
			if strings.TrimSpace(action.Data) == "" {
				return types.Step{}, errors.New("file patch requires data (unified diff)")
			}
		case "replace":
			if action.Old == "" {
				return types.Step{}, errors.New("file replace requires old pattern")
			}
			// new can be empty string in principle (delete), so don't forbid it.
			step.Old = action.Old
			step.New = action.New
			step.N = action.N
		case "write":
			if strings.TrimSpace(action.Data) == "" {
				return types.Step{}, errors.New("file write requires data")
			}
		case "read":
			// ok
		default:
			return types.Step{}, fmt.Errorf("unsupported file op: %q", op)
		}

		return step, nil

	default:
		return types.Step{}, fmt.Errorf("unknown tool: %q", action.Tool)
	}
}

func extractFirstJSONObject(s string) (string, error) {
	// Find first '{'
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return "", errors.New("no '{' found")
	}

	inString := false
	escape := false
	depth := 0

	for i := start; i < len(s); i++ {
		ch := s[i]

		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				// Return the first complete top-level object
				return strings.TrimSpace(s[start : i+1]), nil
			}
		}
	}

	return "", errors.New("unterminated JSON object")
}

func truncateForDisplay(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

type actionSig struct {
	tool string
	key  string
}

type repetitionGuard struct {
	last    *actionSig
	counts  map[actionSig]int
	history []actionSig
	limit   int // max history length to keep counts meaningful
}

func newRepetitionGuard(limit int) *repetitionGuard {
	if limit <= 0 {
		limit = 32
	}
	return &repetitionGuard{
		counts: make(map[actionSig]int),
		limit:  limit,
	}
}

func (g *repetitionGuard) observe(sig actionSig) {
	// maintain rolling window counts
	g.history = append(g.history, sig)
	g.counts[sig]++

	if len(g.history) > g.limit {
		evicted := g.history[0]
		g.history = g.history[1:]
		g.counts[evicted]--
		if g.counts[evicted] <= 0 {
			delete(g.counts, evicted)
		}
	}

	// last is always updated
	g.last = &sig
}

func (g *repetitionGuard) isImmediateRepeat(sig actionSig) bool {
	return g.last != nil && g.last.tool == sig.tool && g.last.key == sig.key
}

func (g *repetitionGuard) count(sig actionSig) int {
	return g.counts[sig]
}

func signatureForAction(a reActAction) actionSig {
	tool := strings.ToLower(strings.TrimSpace(a.Tool))

	switch types.ToolKind(tool) {
	case types.ToolFiles:
		op := strings.ToLower(strings.TrimSpace(a.Op))
		path := strings.TrimPrefix(strings.TrimSpace(a.Path), "./")

		if op == "replace" {
			old := a.Old
			newv := a.New
			if len(old) > 40 {
				old = old[:40]
			}
			if len(newv) > 40 {
				newv = newv[:40]
			}
			return actionSig{tool: string(types.ToolFiles), key: fmt.Sprintf("%s:%s old=%q new=%q n=%d", op, path, old, newv, a.N)}
		}

		if op == "patch" {
			diff := strings.TrimSpace(a.Data)
			prefix := diff
			if len(prefix) > 80 {
				prefix = prefix[:80]
			}
			return actionSig{tool: string(types.ToolFiles), key: fmt.Sprintf("%s:%s len=%d:%q", op, path, len(diff), prefix)}
		}

		return actionSig{tool: string(types.ToolFiles), key: op + ":" + path}

	case types.ToolShell:
		cmd := strings.TrimSpace(a.Command)
		args := normalizeArgs(a.Args)
		key := strings.TrimSpace(cmd + " " + strings.Join(args, " "))
		return actionSig{tool: string(types.ToolShell), key: key}

	case types.ToolLLM:
		p := strings.TrimSpace(a.Prompt)
		prefix := p
		if len(prefix) > 80 {
			prefix = prefix[:80]
		}
		return actionSig{tool: string(types.ToolLLM), key: fmt.Sprintf("len=%d:%s", len(p), prefix)}

	default:
		return actionSig{tool: tool, key: ""}
	}
}

func normalizeArgs(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, a := range in {
		s := strings.TrimSpace(a)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func (a *ReActAgent) promptStateLine() string {
	if len(a.effects) == 0 {
		return ""
	}
	return "side_effects_total=" + summarizeEffectsForUI(a.effects)
}

func formatEffectsForConversation(effects types.Effects) string {
	if len(effects) == 0 {
		return "SIDE_EFFECTS: none"
	}

	var b strings.Builder
	b.WriteString("SIDE_EFFECTS:\n")
	for _, e := range effects {
		b.WriteString("- kind=")
		b.WriteString(e.Kind)

		if e.Path != "" {
			b.WriteString(" path=")
			b.WriteString(strconv.Quote(e.Path))
		}

		if e.Bytes != 0 {
			b.WriteString(" bytes=")
			b.WriteString(strconv.Itoa(e.Bytes))
		}

		if len(e.Meta) > 0 {
			raw, err := json.Marshal(e.Meta)
			if err == nil {
				b.WriteString(" meta=")
				b.WriteString(string(raw))
			}
		}

		b.WriteString("\n")
	}

	return strings.TrimRightFunc(b.String(), unicode.IsSpace)
}

func summarizeEffectsForUI(effects types.Effects) string {
	if len(effects) == 0 {
		return "no side effects"
	}

	counts := map[string]int{}
	for _, e := range effects {
		counts[e.Kind]++
	}

	var parts []string
	for k, n := range counts {
		parts = append(parts, fmt.Sprintf("%s x%d", k, n))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func summarizeActionsForUI(effects types.Effects, llmCalls int) string {
	counts := map[string]int{}
	for _, e := range effects {
		counts[e.Kind]++
	}
	if llmCalls > 0 {
		counts["llm.call"] += llmCalls
	}

	if len(counts) == 0 {
		return "none"
	}

	var parts []string
	for k, n := range counts {
		parts = append(parts, fmt.Sprintf("%s x%d", k, n))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

func mergeEffects(dst *types.Effects, src types.Effects) {
	if len(src) == 0 {
		return
	}
	*dst = append(*dst, src...)
}
