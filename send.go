package weclawbot

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-sdk/core/seq"
)

type sendTextBody struct {
	Message  sendTextMessage `json:"msg"`
	BaseInfo baseInfo        `json:"base_info"`
}

type sendTextMessage struct {
	FromUserID   string                `json:"from_user_id"`
	ToUserID     string                `json:"to_user_id"`
	ClientID     string                `json:"client_id"`
	Type         MessageType           `json:"message_type"`
	State        MessageState          `json:"message_state"`
	ContextToken string                `json:"context_token,omitempty"`
	RunID        string                `json:"run_id,omitempty"`
	Items        []sendTextMessageItem `json:"item_list"`
}

type sendTextMessageItem struct {
	Type MessageItemType `json:"type"`
	Text TextItem        `json:"text_item"`
}

type sendTextWireResponse struct {
	MessageID    MessageID `json:"message_id,omitempty"`
	Ret          *int      `json:"ret,omitempty"`
	ErrorMessage string    `json:"errmsg,omitempty"`
}

// SendText 主动发送或回复一条文本消息。
func (c *Client) SendText(ctx context.Context, request SendTextRequest) (*SendTextResponse, error) {
	if c.token == "" {
		return nil, ErrTokenRequired
	}
	toUserID := strings.TrimSpace(request.ToUserID)
	if toUserID == "" {
		return nil, ErrUserIDRequired
	}
	if strings.TrimSpace(request.Text) == "" {
		return nil, ErrTextRequired
	}
	clientID := seq.UUID()
	body := sendTextBody{
		Message: sendTextMessage{
			FromUserID:   "",
			ToUserID:     toUserID,
			ClientID:     clientID,
			Type:         MessageTypeBot,
			State:        MessageStateFinished,
			ContextToken: strings.TrimSpace(request.ContextToken),
			RunID:        strings.TrimSpace(request.RunID),
			Items: []sendTextMessageItem{
				{
					Type: MessageItemTypeText,
					Text: TextItem{Text: request.Text},
				},
			},
		},
		BaseInfo: c.baseInfo(),
	}
	requestCtx, cancel := requestContext(ctx, c.timeout)
	defer cancel()
	response, err := c.execute(requestCtx, http.MethodPost, c.baseURL, "ilink/bot/sendmessage", headerModeAuthenticatedJSON, body)
	if err != nil {
		return nil, err
	}
	wire := &sendTextWireResponse{}
	if err := decodeResponse(response, wire); err != nil {
		return nil, err
	}
	result := &SendTextResponse{
		ClientID:     clientID,
		MessageID:    wire.MessageID,
		Ret:          wire.Ret,
		ErrorMessage: wire.ErrorMessage,
	}
	if valueIsNonZero(wire.Ret) {
		return result, responseError("send text", wire.Ret, nil)
	}
	return result, nil
}
