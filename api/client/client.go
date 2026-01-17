package client

import (
	"github.com/kardolus/chatgpt-cli/api/http"
	"github.com/kardolus/chatgpt-cli/config"
	"github.com/kardolus/chatgpt-cli/history"
	"github.com/kardolus/chatgpt-cli/internal/fsio"
	"time"
)

const (
	AssistantRole      = "assistant"
	ErrHistoryTracking = "history tracking needs to be enabled to use this feature"
	UserRole           = "user"
)

type Timer interface {
	Now() time.Time
}

type RealTime struct {
}

func (r *RealTime) Now() time.Time {
	return time.Now()
}

type Client struct {
	Config       config.Config
	History      []history.History
	Caller       http.Caller
	historyStore history.Store
	transport    MCPTransport
	timer        Timer
	reader       fsio.Reader
	writer       fsio.Writer
}

func New(callerFactory http.CallerFactory, hs history.Store, t Timer, r fsio.Reader, w fsio.Writer, cfg config.Config) *Client {
	caller := callerFactory(cfg)

	return &Client{
		Config:       cfg,
		Caller:       caller,
		historyStore: hs,
		timer:        t,
		reader:       r,
		writer:       w,
	}
}

func (c *Client) WithContextWindow(window int) *Client {
	c.Config.ContextWindow = window
	return c
}

func (c *Client) WithServiceURL(url string) *Client {
	c.Config.URL = url
	return c
}

func (c *Client) WithTransport(transport MCPTransport) *Client {
	c.transport = transport
	return c
}
