package weclawbot

import (
	"context"
	"net/http"

	"github.com/go-sdk/core/errx"
)

type getUpdatesRequest struct {
	Cursor   string   `json:"get_updates_buf"`
	BaseInfo baseInfo `json:"base_info"`
}

// GetUpdates 执行一次消息长轮询。内部超时返回空消息，调用方取消返回错误。
func (c *Client) GetUpdates(ctx context.Context, request GetUpdatesRequest) (*GetUpdatesResponse, error) {
	if c.token == "" {
		return nil, ErrTokenRequired
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = c.longPollTimeout
	}
	requestCtx, cancel := requestContext(ctx, timeout)
	defer cancel()
	response, err := c.execute(requestCtx, http.MethodPost, c.baseURL, "ilink/bot/getupdates", headerModeAuthenticatedJSON, getUpdatesRequest{
		Cursor:   request.Cursor,
		BaseInfo: c.baseInfo(),
	})
	if err != nil {
		if contextExpiredInternally(ctx, requestCtx) {
			zero := 0
			return &GetUpdatesResponse{
				Ret:      &zero,
				Messages: []Message{},
				Cursor:   request.Cursor,
			}, nil
		}
		return nil, err
	}
	result := &GetUpdatesResponse{}
	if err := decodeResponse(response, result); err != nil {
		return nil, err
	}
	if valueIsNonZero(result.Ret) || valueIsNonZero(result.ErrorCode) {
		return result, responseError("get updates", result.Ret, result.ErrorCode)
	}
	return result, nil
}

// NextCursor 返回服务端提供的非空游标；空游标时保留当前值。
func (r GetUpdatesResponse) NextCursor(current string) string {
	if r.Cursor == "" {
		return current
	}
	return r.Cursor
}

// IsSessionExpired 判断 getUpdates 是否返回需要重新登录的会话失效错误。
func IsSessionExpired(err error) bool {
	return errx.Is(err, ErrSessionExpired)
}
