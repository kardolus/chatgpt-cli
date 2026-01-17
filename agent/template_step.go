package agent

import (
	"bytes"
	"fmt"
	"text/template"
)

// ApplyTemplate applies Go templates in Step string fields using ExecContext.
// If a field contains no "{{", it’s returned unchanged.
func ApplyTemplate(step Step, ctx ExecContext) (Step, error) {
	var err error

	step.Description, err = renderMaybe(step.Description, ctx)
	if err != nil {
		return Step{}, fmt.Errorf("render Description: %w", err)
	}

	switch step.Type {
	case ToolShell:
		step.Command, err = renderMaybe(step.Command, ctx)
		if err != nil {
			return Step{}, fmt.Errorf("render Command: %w", err)
		}
		for i := range step.Args {
			step.Args[i], err = renderMaybe(step.Args[i], ctx)
			if err != nil {
				return Step{}, fmt.Errorf("render Args[%d]: %w", i, err)
			}
		}

	case ToolLLM:
		step.Prompt, err = renderMaybe(step.Prompt, ctx)
		if err != nil {
			return Step{}, fmt.Errorf("render Prompt: %w", err)
		}

	case ToolFiles:
		step.Op, err = renderMaybe(step.Op, ctx)
		if err != nil {
			return Step{}, fmt.Errorf("render Op: %w", err)
		}
		step.Path, err = renderMaybe(step.Path, ctx)
		if err != nil {
			return Step{}, fmt.Errorf("render Path: %w", err)
		}
		step.Data, err = renderMaybe(step.Data, ctx)
		if err != nil {
			return Step{}, fmt.Errorf("render Data: %w", err)
		}
	}

	return step, nil
}

func renderMaybe(s string, ctx ExecContext) (string, error) {
	if !containsTemplateMarker(s) {
		return s, nil
	}

	tpl, err := template.New("step").Option("missingkey=error").Parse(s)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func containsTemplateMarker(s string) bool {
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '{' && s[i+1] == '{' {
			return true
		}
	}
	return false
}
