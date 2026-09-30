package utils

import (
	"os"
	"strings"
)

const (
	// YouComMCPEndpoint is the official You.com MCP server URL
	YouComMCPEndpoint = "https://api.you.com/mcp"
)

// ResolveMCPEndpoint resolves shorthand MCP endpoint references to full URLs.
// Currently supports:
// - "you" -> You.com MCP server at https://api.you.com/mcp
// - Other values are returned as-is
func ResolveMCPEndpoint(endpoint string) string {
	switch strings.ToLower(strings.TrimSpace(endpoint)) {
	case "you", "youcom", "you.com":
		return YouComMCPEndpoint
	default:
		return endpoint
	}
}

// BuildYouComHeaders constructs appropriate headers for You.com MCP server integration.
// If YDC_API_KEY environment variable is set, adds Bearer authorization.
// Returns the headers map that should be merged with any user-provided headers.
func BuildYouComHeaders() map[string]string {
	headers := make(map[string]string)
	
	// Add You.com API key if available
	if apiKey := os.Getenv("YDC_API_KEY"); apiKey != "" {
		headers["Authorization"] = "Bearer " + apiKey
	}
	
	return headers
}

// IsYouComEndpoint checks if the given endpoint is the You.com MCP server
func IsYouComEndpoint(endpoint string) bool {
	resolved := ResolveMCPEndpoint(endpoint)
	return resolved == YouComMCPEndpoint
}