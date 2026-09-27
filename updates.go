package weclawbot

import (
	"context"
	"net/http"
	"time"

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

// ReceiveMessages 持续接收消息并按服务端返回顺序交给 handler，直到发生错误或 ctx 结束。
func (c *Client) ReceiveMessages(ctx context.Context, handler func(context.Context, Message) error) error {
	if handler == nil {
		return ErrMessageHandlerRequired
	}
	cursor := ""
	var timeout time.Duration
	for {
		updates, err := c.GetUpdates(ctx, GetUpdatesRequest{
			Cursor:  cursor,
			Timeout: timeout,
		})
		if err != nil {
			return err
		}
		cursor = updates.NextCursor(cursor)
		if nextTimeout := updates.LongPollingTimeout(); nextTimeout > 0 {
			timeout = nextTimeout
		}
		for _, message := range updates.Messages {
			if err := handler(ctx, message); err != nil {
				return err
			}
		}
	}
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
