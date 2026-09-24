package weclawbot

import (
	"strings"
	"time"

	"github.com/go-sdk/core/restx"
)

const (
	defaultBaseURL         = "https://ilinkai.weixin.qq.com"
	defaultRequestTimeout  = 15 * time.Second
	defaultLongPollTimeout = 35 * time.Second
)

type Option func(*Client)

// Client 是微信 ClawBot HTTP JSON 协议客户端。
type Client struct {
	cli *restx.Client

	baseURL         string
	token           string
	routeTag        string
	channelVersion  string
	botAgent        string
	timeout         time.Duration
	longPollTimeout time.Duration
	allowHTTP       bool
}

// New 创建使用统一 restx 默认配置的客户端。
func New(opts ...Option) *Client {
	c := &Client{
		cli:             restx.New(),
		baseURL:         defaultBaseURL,
		channelVersion:  moduleVersion(),
		timeout:         defaultRequestTimeout,
		longPollTimeout: defaultLongPollTimeout,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	if c.botAgent == "" {
		c.botAgent = defaultBotAgent(c.channelVersion)
	}
	return c
}

// WithToken 配置二维码登录后取得的 Bot Token。
func WithToken(token string) Option {
	return func(c *Client) {
		c.token = strings.TrimSpace(token)
	}
}

// WithClient 注入自定义 restx 客户端，主要用于调整传输配置或本地测试。
func WithClient(cli *restx.Client) Option {
	return func(c *Client) {
		if cli != nil {
			c.cli = cli
		}
	}
}

// WithRouteTag 配置可选的微信路由标签。
func WithRouteTag(routeTag string) Option {
	return func(c *Client) {
		c.routeTag = strings.TrimSpace(routeTag)
	}
}

// WithChannelVersion 覆盖 base_info 和客户端请求头使用的版本号。
func WithChannelVersion(version string) Option {
	return func(c *Client) {
		version = normalizeVersion(version)
		if version != "" {
			c.channelVersion = version
		}
	}
}

// WithBotAgent 配置 base_info 中仅用于观测归因的客户端标识。
func WithBotAgent(agent string) Option {
	return func(c *Client) {
		c.botAgent = sanitizeBotAgent(agent, defaultBotAgent(c.channelVersion))
	}
}

// WithTimeout 配置普通 API 请求超时。
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithLongPollTimeout 配置二维码状态和消息查询的单次长轮询超时。
func WithLongPollTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.longPollTimeout = timeout
		}
	}
}

// WithBaseURL 配置登录确认后返回的账号 API 地址，默认只接受 HTTPS。
func WithBaseURL(u string) Option {
	return func(c *Client) {
		u = strings.TrimSuffix(strings.TrimSpace(u), "/")
		if u != "" {
			c.baseURL = u
		}
	}
}

// WithInsecureHTTP 显式允许使用明文 HTTP API 地址，仅应用于可信的开发或测试环境。
func WithInsecureHTTP() Option {
	return func(c *Client) {
		c.allowHTTP = true
	}
}
