package weclawbot

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-sdk/core/codec/json"
	"github.com/go-sdk/core/errx"
	"github.com/go-sdk/core/restx"
)

const (
	headerAuthorizationType = "AuthorizationType"
	headerWechatUIN         = "X-WECHAT-UIN"
	headerAppID             = "iLink-App-Id"
	headerClientVersion     = "iLink-App-ClientVersion"
	headerRouteTag          = "SKRouteTag"
)

type baseInfo struct {
	ChannelVersion string `json:"channel_version"`
	BotAgent       string `json:"bot_agent"`
}

type headerMode int

const (
	headerModeApplication headerMode = iota
	headerModeAnonymousJSON
	headerModeAuthenticatedJSON
)

func (c *Client) baseInfo() baseInfo {
	return baseInfo{
		ChannelVersion: c.channelVersion,
		BotAgent:       c.botAgent,
	}
}

func (c *Client) headers(mode headerMode) (map[string]string, error) {
	headers := map[string]string{
		headerAppID:         "bot",
		headerClientVersion: clientVersion(c.channelVersion),
	}
	if c.routeTag != "" {
		headers[headerRouteTag] = c.routeTag
	}
	if mode == headerModeApplication {
		return headers, nil
	}
	uin, err := randomWechatUIN()
	if err != nil {
		return nil, err
	}
	headers["Content-Type"] = "application/json"
	headers[headerAuthorizationType] = "ilink_bot_token"
	headers[headerWechatUIN] = uin
	if mode == headerModeAuthenticatedJSON {
		if c.token == "" {
			return nil, ErrTokenRequired
		}
		headers["Authorization"] = "Bearer " + c.token
	}
	return headers, nil
}

func randomWechatUIN() (string, error) {
	var data [4]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", errx.Wrap(err, "generate wechat uin")
	}
	value := strconv.FormatUint(uint64(binary.BigEndian.Uint32(data[:])), 10)
	return base64.StdEncoding.EncodeToString([]byte(value)), nil
}

func resolveURL(baseURL, endpoint string, allowHTTP bool) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return "", ErrInvalidURL
	}
	if parsed.Scheme != "https" && (!allowHTTP || parsed.Scheme != "http") {
		return "", ErrInvalidURL
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimSuffix(parsed.Path, "/") + "/"
	relative, err := url.Parse(strings.TrimPrefix(endpoint, "/"))
	if err != nil {
		return "", errx.Wrap(err, "parse api endpoint")
	}
	return parsed.ResolveReference(relative).String(), nil
}

func redirectBaseURL(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "/?#@") {
		return "", ErrInvalidURL
	}
	parsed, err := url.Parse("https://" + host)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil {
		return "", ErrInvalidURL
	}
	return parsed.String(), nil
}

func requestContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}

func (c *Client) execute(ctx context.Context, method, baseURL, endpoint string, mode headerMode, body any) (*restx.Response, error) {
	requestURL, err := resolveURL(baseURL, endpoint, c.allowHTTP)
	if err != nil {
		return nil, err
	}
	headers, err := c.headers(mode)
	if err != nil {
		return nil, err
	}
	request := c.cli.R().SetContext(ctx).SetHeaders(headers)
	if body != nil {
		request.SetBody(body)
	}
	var response *restx.Response
	switch method {
	case http.MethodGet:
		response, err = request.Get(requestURL)
	case http.MethodPost:
		response, err = request.Post(requestURL)
	default:
		return nil, errx.Wrapf(ErrProtocol, "unsupported method %q", method)
	}
	if err != nil {
		if ctx.Err() != nil {
			return response, ctx.Err()
		}
		return response, ErrRequest
	}
	if response.StatusCode() < http.StatusOK || response.StatusCode() >= http.StatusMultipleChoices {
		return response, errx.Wrapf(ErrHTTPStatus, "status %d", response.StatusCode())
	}
	return response, nil
}

func decodeResponse(response *restx.Response, target any) error {
	if err := json.Unmarshal(response.Body(), target); err != nil {
		return errx.Wrap(err, "decode wechat response")
	}
	return nil
}

func responseError(operation string, ret, errorCode *int) error {
	if valueIs(ret, -14) || valueIs(errorCode, -14) {
		return errx.Wrapf(ErrSessionExpired, "%s ret=%s errcode=%s", operation, intValue(ret), intValue(errorCode))
	}
	return errx.Wrapf(ErrProtocol, "%s ret=%s errcode=%s", operation, intValue(ret), intValue(errorCode))
}

func valueIs(value *int, target int) bool {
	return value != nil && *value == target
}

func valueIsNonZero(value *int) bool {
	return value != nil && *value != 0
}

func intValue(value *int) string {
	if value == nil {
		return "missing"
	}
	return strconv.Itoa(*value)
}
