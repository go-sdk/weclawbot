package weclawbot

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-sdk/core/errx"
)

const defaultBotType = "3"

type getQRCodeRequest struct {
	LocalTokens []string `json:"local_token_list"`
}

// GetQRCode 获取微信登录二维码。该请求始终使用微信固定登录入口。
func (c *Client) GetQRCode(ctx context.Context, request GetQRCodeRequest) (*GetQRCodeResponse, error) {
	tokens := make([]string, 0, len(request.LocalTokens))
	for _, token := range request.LocalTokens {
		if token = strings.TrimSpace(token); token != "" {
			tokens = append(tokens, token)
		}
	}
	if len(tokens) > 10 {
		return nil, errx.Wrapf(ErrTooManyLocalTokens, "got %d, maximum 10", len(tokens))
	}
	requestCtx, cancel := requestContext(ctx, c.timeout)
	defer cancel()
	response, err := c.execute(requestCtx, http.MethodPost, defaultBaseURL, "ilink/bot/get_bot_qrcode?bot_type="+defaultBotType, headerModeAnonymousJSON, getQRCodeRequest{LocalTokens: tokens})
	if err != nil {
		return nil, err
	}
	result := &GetQRCodeResponse{}
	if err := decodeResponse(response, result); err != nil {
		return nil, err
	}
	if result.QRCode == "" || result.QRCodeImageContent == "" {
		return result, errx.Wrap(ErrProtocol, "get qrcode response is incomplete")
	}
	return result, nil
}

// GetQRCodeStatus 执行一次二维码状态长轮询。内部超时返回 wait，调用方取消返回错误。
func (c *Client) GetQRCodeStatus(ctx context.Context, request GetQRCodeStatusRequest) (*GetQRCodeStatusResponse, error) {
	qrcode := strings.TrimSpace(request.QRCode)
	if qrcode == "" {
		return nil, ErrQRCodeRequired
	}
	baseURL := strings.TrimSpace(request.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	query := url.Values{"qrcode": []string{qrcode}}
	if verifyCode := strings.TrimSpace(request.VerifyCode); verifyCode != "" {
		query.Set("verify_code", verifyCode)
	}
	requestCtx, cancel := requestContext(ctx, c.longPollTimeout)
	defer cancel()
	response, err := c.execute(requestCtx, http.MethodGet, baseURL, "ilink/bot/get_qrcode_status?"+query.Encode(), headerModeApplication, nil)
	if err != nil {
		if contextExpiredInternally(ctx, requestCtx) {
			return &GetQRCodeStatusResponse{Status: QRCodeStatusWait}, nil
		}
		return nil, err
	}
	result := &GetQRCodeStatusResponse{}
	if err := decodeResponse(response, result); err != nil {
		return nil, err
	}
	if result.Status == "" {
		return result, errx.Wrap(ErrProtocol, "qrcode status is missing")
	}
	if result.Status == QRCodeStatusConfirmed && result.BotID == "" {
		return result, errx.Wrap(ErrProtocol, "confirmed login is missing bot id")
	}
	if result.Status == QRCodeStatusConfirmed && result.BotToken == "" {
		return result, errx.Wrap(ErrProtocol, "confirmed login is missing bot token")
	}
	return result, nil
}

// RedirectBaseURL 校验并返回 scaned_but_redirect 状态提供的新轮询地址。
func (r GetQRCodeStatusResponse) RedirectBaseURL() (string, error) {
	return redirectBaseURL(r.RedirectHost)
}

func contextExpiredInternally(parent, request context.Context) bool {
	return (parent == nil || parent.Err() == nil) && errx.Is(request.Err(), context.DeadlineExceeded)
}
