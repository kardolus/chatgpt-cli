package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kardolus/chatgpt-cli/agent/types"
	"gopkg.in/yaml.v3"
)

// BoolToOnOff renders a boolean as the "ON"/"OFF" strings used in the CLI's
// config display.
func BoolToOnOff(b bool) string {
	if b {
		return "ON"
	}
	return "OFF"
}

// ToAliasFlagName converts a viper config key (e.g. "context_window" or a
// dotted "agent.mode") into its dashed CLI alias flag name.
func ToAliasFlagName(viperKey string) string {
	s := strings.ReplaceAll(viperKey, ".", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// ResolveAgentMode normalizes the agent mode from the flag (falling back to the
// config value, then "react"). Aliases plan_execute/plan-execute map to "plan".
func ResolveAgentMode(flagMode, cfgMode string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(flagMode))
	if mode == "" {
		mode = strings.ToLower(strings.TrimSpace(cfgMode))
	}
	if mode == "" {
		mode = "react"
	}

	switch mode {
	case "react":
		return "react", nil
	case "plan":
		return "plan", nil
	case "plan_execute", "plan-execute":
		return "plan", nil
	default:
		return "", fmt.Errorf("unknown agent mode %q (expected react|plan)", mode)
	}
}

// BuildAgentGoal joins any piped chat context and CLI args into a single agent
// goal, erroring when nothing was provided.
func BuildAgentGoal(chatContext string, args []string) (string, error) {
	var parts []string
	if s := strings.TrimSpace(chatContext); s != "" {
		parts = append(parts, s)
	}
	if len(args) > 0 {
		parts = append(parts, strings.Join(args, " "))
	}

	goal := strings.TrimSpace(strings.Join(parts, "\n"))
	if goal == "" {
		return "", errors.New("missing agent goal (provide args or pipe)")
	}
	return goal, nil
}

// ParseToolKinds parses the agent.allowed_tools config entries (shell|llm|files)
// into deduplicated ToolKinds, erroring on an unknown or empty set.
func ParseToolKinds(in []string) ([]types.ToolKind, error) {
	out := make([]types.ToolKind, 0, len(in))
	seen := map[types.ToolKind]bool{}

	for _, raw := range in {
		s := strings.ToLower(strings.TrimSpace(raw))
		if s == "" {
			continue
		}

		var k types.ToolKind
		switch s {
		case "shell":
			k = types.ToolShell
		case "llm":
			k = types.ToolLLM
		case "files", "file":
			k = types.ToolFiles
		default:
			return nil, fmt.Errorf("unknown agent.allowed_tools entry %q (expected shell|llm|files)", raw)
		}

		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}

	// An empty allow-list is treated as an error rather than silently allowing
	// every tool.
	if len(out) == 0 {
		return nil, errors.New("agent.allowed_tools is empty (expected at least one of shell|llm|files)")
	}

	return out, nil
}

// generalFlags are the non-config CLI flags (actions and one-shot options) that
// are neither `set-*` setters nor config aliases.
var generalFlags = map[string]bool{
	"query":           true,
	"interactive":     true,
	"config":          true,
	"version":         true,
	"new-thread":      true,
	"list-models":     true,
	"list-threads":    true,
	"clear-history":   true,
	"delete-thread":   true,
	"show-history":    true,
	"prompt":          true,
	"agent":           true,
	"set-completions": true,
	"help":            true,
	"role-file":       true,
	"image":           true,
	"audio":           true,
	"speak":           true,
	"draw":            true,
	"output":          true,
	"transcribe":      true,
	"mcp":             true,
	"mcp-header":      true,
	"mcp-param":       true,
	"mcp-params":      true,
	"mcp-tool":        true,
	"target":          true,
}

// IsNonConfigSetter reports whether a `set-*`-style flag does not persist a
// config value (only "set-completions" today).
func IsNonConfigSetter(name string) bool {
	return name == "set-completions"
}

// IsGeneralFlag reports whether name is a general (non-config) CLI flag.
func IsGeneralFlag(name string) bool {
	return generalFlags[name]
}

// IsConfigAlias reports whether name is a config-value alias flag (i.e. neither
// a `set-*` setter nor a general flag).
func IsConfigAlias(name string) bool {
	return !strings.HasPrefix(name, "set-") && !IsGeneralFlag(name)
}

// MergeMaps copies all entries of m2 into m1 (m2 wins on conflict) and returns
// m1.
func MergeMaps(m1, m2 map[string]interface{}) map[string]interface{} {
	for k, v := range m2 {
		m1[k] = v
	}
	return m1
}

// FileExists reports whether filename exists and is statable.
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// ReadConfigWithComments reads a YAML config file into a *yaml.Node, preserving
// comments and formatting for in-place edits.
func ReadConfigWithComments(configPath string) (*yaml.Node, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var rootNode yaml.Node
	if err := yaml.Unmarshal(data, &rootNode); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}
	return &rootNode, nil
}

// SaveConfigWithComments marshals a *yaml.Node back to the config file (0600).
func SaveConfigWithComments(configPath string, node *yaml.Node) error {
	out, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	return os.WriteFile(configPath, out, 0o600)
}

// UpdateConfigNode applies changes onto a YAML document node in place: existing
// keys are updated and missing keys appended, preserving surrounding comments.
//
// Values are written with their default string form (fmt.Sprintf("%v", ...))
// and left untagged, so a bool true is emitted as `true` and an int 100 as
// `100`; YAML re-parses these back to their scalar types on the next read. Only
// flat, scalar top-level keys are supported (the CLI's `set-*` writes), not
// nested/structured values.
func UpdateConfigNode(node *yaml.Node, changes map[string]interface{}) error {
	// If the node is not a document or has no content, create an empty mapping.
	if node.Kind != yaml.DocumentNode || len(node.Content) == 0 {
		node.Kind = yaml.DocumentNode
		node.Content = []*yaml.Node{
			{
				Kind:    yaml.MappingNode,
				Content: []*yaml.Node{},
			},
		}
	}

	mapNode := node.Content[0]
	if mapNode.Kind != yaml.MappingNode {
		return errors.New("expected a mapping node at the root of the YAML document")
	}

	// Update existing keys.
	for i := 0; i < len(mapNode.Content); i += 2 {
		keyNode := mapNode.Content[i]
		valueNode := mapNode.Content[i+1]

		if newValue, ok := changes[keyNode.Value]; ok {
			valueNode.Value = fmt.Sprintf("%v", newValue)
		}
	}

	// Append keys that don't exist yet.
	for key, value := range changes {
		if !keyExistsInNode(mapNode, key) {
			mapNode.Content = append(mapNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: key,
			}, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%v", value),
			})
		}
	}

	return nil
}

func keyExistsInNode(mapNode *yaml.Node, key string) bool {
	for i := 0; i < len(mapNode.Content); i += 2 {
		if mapNode.Content[i].Value == key {
			return true
		}
	}
	return false
}
