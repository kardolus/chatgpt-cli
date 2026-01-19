package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// matches: (index .Results 0) or (index .Results 12)
var reResultsIndex = regexp.MustCompile(`\( *index +\.Results +([0-9]+) *\)`)

//go:generate mockgen -destination=plannermocks_test.go -package=agent_test github.com/kardolus/chatgpt-cli/agent Planner
type Planner interface {
	Plan(ctx context.Context, goal string) (Plan, error)
}

type LoggingPlanner struct {
	inner Planner
	log   *zap.SugaredLogger

	// artifacts (overwritten every run)
	dir            string
	rawPath        string
	normalizedPath string
}

func NewLoggingPlanner(inner Planner, logs *Logs) *LoggingPlanner {
	// default: no-op logger, no files
	lp := &LoggingPlanner{
		inner: inner,
		log:   zap.NewNop().Sugar(),
	}

	if logs == nil {
		return lp
	}
	if logs.DebugLogger != nil {
		lp.log = logs.DebugLogger
	}
	if logs.Dir != "" {
		lp.dir = logs.Dir
		lp.rawPath = filepath.Join(logs.Dir, "plan.json")
		lp.normalizedPath = filepath.Join(logs.Dir, "plan.normalized.json")
	}
	return lp
}

func (p *LoggingPlanner) Plan(ctx context.Context, goal string) (Plan, error) {
	g := strings.TrimSpace(goal)
	p.log.Debugf("planner: start goal_len=%d", len(g))

	plan, err := p.inner.Plan(ctx, goal)
	if err != nil {
		p.log.Debugf("planner: error=%v", err)
		return Plan{}, err
	}

	// Write normalized plan
	p.writeNormalized(plan)

	p.log.Debugf("planner: ok steps=%d", len(plan.Steps))
	return plan, nil
}

func (p *LoggingPlanner) WriteRaw(raw string) {
	if p.rawPath == "" {
		return
	}
	_ = os.WriteFile(p.rawPath, []byte(raw), 0o644) // best-effort
}

func (p *LoggingPlanner) writeNormalized(plan Plan) {
	if p.normalizedPath == "" {
		return
	}
	b, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		p.log.Debugf("planner: failed to marshal normalized plan: %v", err)
		return
	}
	_ = os.WriteFile(p.normalizedPath, b, 0o644) // best-effort
}

type DefaultPlanner struct {
	llm    LLM
	budget Budget
	clock  Clock

	onRaw func(raw string) // optional
}

func NewDefaultPlanner(llm LLM, budget Budget, clock Clock, opts ...PlannerOption) *DefaultPlanner {
	p := &DefaultPlanner{llm: llm, budget: budget, clock: clock}
	for _, o := range opts {
		o(p)
	}
	return p
}

type PlannerOption func(*DefaultPlanner)

func WithPlannerRawSink(fn func(string)) PlannerOption {
	return func(p *DefaultPlanner) {
		p.onRaw = fn
	}
}

func (p *DefaultPlanner) Plan(ctx context.Context, goal string) (Plan, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return Plan{}, errors.New("missing goal")
	}

	now := p.clock.Now()

	// Count this as an LLM tool call in budget.
	if err := p.budget.AllowTool(ToolLLM, now); err != nil {
		return Plan{}, err
	}

	prompt := buildPlanningPrompt(goal)

	raw, tokens, err := p.llm.Complete(ctx, prompt)
	if err != nil {
		return Plan{}, err
	}

	if p.onRaw != nil {
		p.onRaw(raw)
	}

	p.budget.ChargeLLMTokens(tokens, now)

	plan, err := parsePlanJSON(raw, goal)
	if err != nil {
		return Plan{}, err
	}

	if err := validatePlan(plan); err != nil {
		return Plan{}, err
	}

	return plan, nil
}

func buildPlanningPrompt(goal string) string {
	return fmt.Sprintf(`
You are a planning module for a CLI agent. Convert the user's goal into an explicit plan.

CRITICAL OUTPUT RULES:
- Return ONLY raw JSON.
- Do NOT use markdown.
- Do NOT use code fences.
- Do NOT add prose, comments, or explanations.
- The FIRST non-whitespace character MUST be '{'.
- The LAST non-whitespace character MUST be '}'.
- If you cannot produce valid JSON, return: {"goal": "...", "steps": []}

Return JSON matching this schema:

{
  "goal": "string",
  "steps": [
    {
      "type": "%s" | "%s" | "%s",
      "description": "string",

      // %s-only:
      "command": "string",
      "args": ["string", "..."],

      // %s-only:
      "prompt": "string",

      // %s-only:
      "op": "read" | "write",
      "path": "string",
      "data": "string"
    }
  ]
}

Core rules:
- Keep steps minimal.
- Prefer %s steps for concrete actions.
- Use %s steps for reasoning/summarization based on prior results.
- Use %s steps only for explicit reads/writes.
- Every step must have a non-empty description.
- You MAY include Go template expressions like {{ ... }} in any string field; they will be rendered later.

FILE TOOL SEMANTICS (IMPORTANT):
- "op":"read" returns the full current file content as Output.
- "op":"write" OVERWRITES THE ENTIRE FILE CONTENT with "data".
- There is NO append mode and NO in-place edit mode.
- Therefore, for "modify a line or two", plan MUST do:
  1) file read the current content
  2) llm produce the FULL updated content (include unchanged parts)
  3) file write the FULL updated content back

FILE WRITE OUTPUT RULE (CRITICAL):
When you choose tool="file" with op="write", the value of "data" must be the EXACT file contents to write.

- Do NOT wrap "data" in markdown fences.
	- Do NOT add leading/trailing backticks.
	- Do NOT add any prose before/after the file content.
	- For non-.md files, "data" must be plain raw text/code only.

MARKDOWN IS ONLY ALLOWED INSIDE "data" WHEN:
- path ends with ".md" AND the user asked for markdown formatting changes.
Otherwise, preserve the existing file’s formatting and do not introduce markdown syntax.

Prohibited patterns (unless the user explicitly wants to replace the whole file with only that snippet):
- Writing only a "diff", "patch", or partial snippet to a file.
- Writing only "the new paragraph" or "the new attempt" without including the existing content.

Template rules:
- Templates use Go text/template syntax.
- They are rendered at runtime with missingkey=error, so ALL referenced keys must exist.
- You can reference prior step outputs via:
  - {{ (index .Results 0).Output }}
  - {{ (index .Results 1).Output }}
- Prefer using .Output unless you explicitly need raw stdout/stderr.

Examples:

1) Shell + summarize:
{
  "type": "%s",
  "description": "Get git status",
  "command": "git",
  "args": ["status", "--porcelain"]
},
{
  "type": "%s",
  "description": "Summarize changes",
  "prompt": "Summarize these changes:\n{{ (index .Results 0).Output }}"
}

2) Edit a file safely (read -> generate full new content -> write full file):
{
  "type": "%s",
  "description": "Read the existing report",
  "op": "read",
  "path": "report.txt"
},
{
  "type": "%s",
  "description": "Produce the full updated report text (preserve existing content, apply requested changes)",
  "prompt": "Here is the current file content:\n---\n{{ (index .Results 0).Output }}\n---\nRewrite the ENTIRE file content with the requested changes applied. Return ONLY the full new file content."
},
{
  "type": "%s",
  "description": "Overwrite report with updated content",
  "op": "write",
  "path": "report.txt",
  "data": "{{ (index .Results 1).Output }}"
}

User goal:
%q

SELF-CHECK BEFORE RESPONDING:
- Does output start with '{' and end with '}'?
- Is it valid JSON?
- Does it contain NO markdown or backticks?
If any answer is "no", fix it before returning.
`,
		ToolShell, ToolLLM, ToolFiles,
		ToolShell,
		ToolLLM,
		ToolFiles,
		ToolShell,
		ToolLLM,
		ToolFiles,
		ToolShell,
		ToolLLM,
		ToolFiles,
		ToolLLM,
		ToolFiles,
		goal,
	)
}

type planJSON struct {
	Goal  string     `json:"goal"`
	Steps []stepJSON `json:"steps"`
}

type stepJSON struct {
	Type        string `json:"type"`
	Description string `json:"description"`

	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`

	Prompt string `json:"prompt,omitempty"`

	Op   string `json:"op,omitempty"`
	Path string `json:"path,omitempty"`
	Data string `json:"data,omitempty"`
}

func parsePlanJSON(raw string, fallbackGoal string) (Plan, error) {
	raw = cleanPlannerOutput(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Plan{}, errors.New("planner returned empty response")
	}

	var pj planJSON
	if err := json.Unmarshal([]byte(raw), &pj); err != nil {
		return Plan{}, fmt.Errorf("failed to parse planner JSON: %w", err)
	}

	goal := strings.TrimSpace(pj.Goal)
	if goal == "" {
		goal = fallbackGoal
	}

	out := Plan{Goal: goal}
	out.Steps = make([]Step, 0, len(pj.Steps))

	for _, s := range pj.Steps {
		step, err := convertStepJSON(s)
		if err != nil {
			return Plan{}, err
		}
		out.Steps = append(out.Steps, step)
	}

	return out, nil
}

func cleanPlannerOutput(raw string) string {
	raw = strings.TrimSpace(raw)

	// If wrapped in ```...```
	if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSpace(raw)

		// Remove optional language tag (only if the first line IS a language tag)
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

func convertStepJSON(s stepJSON) (Step, error) {
	t := strings.TrimSpace(strings.ToLower(s.Type))
	desc := strings.TrimSpace(s.Description)
	if desc == "" {
		return Step{}, errors.New("planner step missing description")
	}

	switch t {
	case string(ToolShell): // "shell"
		cmd := strings.TrimSpace(s.Command)
		if cmd == "" {
			return Step{}, errors.New("shell step missing command")
		}
		return Step{
			Type:        ToolShell,
			Description: desc,
			Command:     cmd,
			Args:        s.Args,
		}, nil

	case string(ToolLLM): // "llm"
		prompt := strings.TrimSpace(s.Prompt)
		if prompt == "" {
			return Step{}, errors.New("llm step missing prompt")
		}
		return Step{
			Type:        ToolLLM,
			Description: desc,
			Prompt:      prompt,
		}, nil

	case string(ToolFiles): // "file"
		op := strings.TrimSpace(strings.ToLower(s.Op))
		path := strings.TrimSpace(s.Path)
		if op == "" {
			return Step{}, errors.New("file step missing op")
		}
		if path == "" {
			return Step{}, errors.New("file step missing path")
		}
		return Step{
			Type:        ToolFiles,
			Description: desc,
			Op:          op,
			Path:        path,
			Data:        s.Data,
		}, nil

	default:
		return Step{}, fmt.Errorf("unknown step type: %q", s.Type)
	}
}

func validatePlan(p Plan) error {
	if strings.TrimSpace(p.Goal) == "" {
		return errors.New("plan missing goal")
	}
	if len(p.Steps) == 0 {
		return errors.New("plan has no steps")
	}

	for i, s := range p.Steps {
		if strings.TrimSpace(s.Description) == "" {
			return fmt.Errorf("step %d missing description", i)
		}
		switch s.Type {
		case ToolShell:
			if strings.TrimSpace(s.Command) == "" {
				return fmt.Errorf("step %d shell missing command", i)
			}
		case ToolLLM:
			if strings.TrimSpace(s.Prompt) == "" {
				return fmt.Errorf("step %d llm missing prompt", i)
			}
		case ToolFiles:
			if strings.TrimSpace(s.Op) == "" {
				return fmt.Errorf("step %d files missing op", i)
			}
			if strings.TrimSpace(s.Path) == "" {
				return fmt.Errorf("step %d files missing path", i)
			}
		default:
			return fmt.Errorf("step %d has unknown type %q", i, s.Type)
		}
	}

	if err := validateTemplates(p); err != nil {
		return err
	}

	return nil
}

func validateTemplates(p Plan) error {
	for i := range p.Steps {
		s := p.Steps[i]

		if err := validateTemplateField(i, "description", s.Description); err != nil {
			return err
		}

		switch s.Type {
		case ToolShell:
			if err := validateTemplateField(i, "command", s.Command); err != nil {
				return err
			}
			for ai, a := range s.Args {
				if err := validateTemplateField(i, fmt.Sprintf("args[%d]", ai), a); err != nil {
					return err
				}
			}

		case ToolLLM:
			if err := validateTemplateField(i, "prompt", s.Prompt); err != nil {
				return err
			}

		case ToolFiles:
			if err := validateTemplateField(i, "op", s.Op); err != nil {
				return err
			}
			if err := validateTemplateField(i, "path", s.Path); err != nil {
				return err
			}
			if err := validateTemplateField(i, "data", s.Data); err != nil {
				return err
			}

		default:
			return fmt.Errorf("step %d has unknown type %q", i, s.Type)
		}
	}
	return nil
}

func validateTemplateField(stepIndex int, field string, s string) error {
	if !strings.Contains(s, "{{") {
		return nil
	}

	_, err := template.New("validate").Option("missingkey=error").Parse(s)
	if err != nil {
		return fmt.Errorf("step %d %s: invalid template: %w", stepIndex, field, err)
	}

	matches := reResultsIndex.FindAllStringSubmatch(s, -1)

	if strings.Contains(s, "index .Results") && len(matches) == 0 {
		return fmt.Errorf(
			"step %d %s: template uses index .Results but not with a literal index",
			stepIndex, field,
		)
	}

	for _, m := range matches {
		n, convErr := strconv.Atoi(m[1])
		if convErr != nil {
			return fmt.Errorf("step %d %s: invalid Results index %q", stepIndex, field, m[1])
		}
		if n >= stepIndex {
			return fmt.Errorf(
				"step %d %s: template references .Results[%d] but only prior results are available (max index %d)",
				stepIndex, field, n, stepIndex-1,
			)
		}
	}

	return nil
}

type NaivePlanner struct{}

func (p *NaivePlanner) Plan(ctx context.Context, goal string) (Plan, error) {
	// Stub: good enough for wiring + tests.
	// Later: call a.client to generate this.
	return Plan{
		Goal: goal,
		Steps: []Step{
			{
				Type:        ToolShell,
				Description: "Show repo status",
				Command:     "git",
				Args:        []string{"status", "--porcelain"},
			},
			{
				Type:        ToolShell,
				Description: "Run tests",
				Command:     "go",
				Args:        []string{"test", "./..."},
			},
		},
	}, nil
}
