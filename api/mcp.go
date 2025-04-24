package api

type MCPRequest struct {
	Provider string
	Function string
	Version  string
	Params   map[string]interface{}
}

type ProxyConfiguration struct {
	UseApifyProxy bool `json:"useApifyProxy"`
}
