package client

import (
	"github.com/kardolus/chatgpt-cli/internal"
	"go.uber.org/zap"
	"strings"
)

func (c *Client) printRequestDebugInfo(endpoint string, body []byte, headers map[string]string) {
	sugar := zap.S()
	sugar.Debugf("\nGenerated cURL command:\n")

	method := "POST"
	if body == nil {
		method = "GET"
	}
	sugar.Debugf("curl --location --insecure --request %s '%s' \\", method, endpoint)

	if len(headers) > 0 {
		for k, v := range headers {
			sugar.Debugf("  --header '%s: %s' \\", k, v)
		}
	} else {
		sugar.Debugf("  --header \"%s: %s${%s_API_KEY}\" \\", c.Config.AuthHeader, c.Config.AuthTokenPrefix, strings.ToUpper(c.Config.Name))
		sugar.Debugf("  --header '%s: %s' \\", internal.HeaderContentTypeKey, internal.HeaderContentTypeValue)
		sugar.Debugf("  --header '%s: %s' \\", internal.HeaderUserAgentKey, c.Config.UserAgent)

		// Include custom headers from config
		for k, v := range c.Config.CustomHeaders {
			sugar.Debugf("  --header '%s: %s' \\", k, v)
		}
	}

	if body != nil {
		bodyString := strings.ReplaceAll(string(body), "'", "'\"'\"'")
		sugar.Debugf("  --data-raw '%s'", bodyString)
	}
}

func (c *Client) printResponseDebugInfo(raw []byte) {
	sugar := zap.S()
	sugar.Debugf("\nResponse\n")
	sugar.Debugf("%s\n", raw)
}
