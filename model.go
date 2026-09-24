package weclawbot

import (
	"bytes"
	"strconv"
	"time"

	"github.com/go-sdk/core/errx"
)

// QRCodeStatus 表示二维码登录状态。
type QRCodeStatus string

const (
	QRCodeStatusWait              QRCodeStatus = "wait"
	QRCodeStatusScanned           QRCodeStatus = "scaned"
	QRCodeStatusConfirmed         QRCodeStatus = "confirmed"
	QRCodeStatusExpired           QRCodeStatus = "expired"
	QRCodeStatusNeedVerifyCode    QRCodeStatus = "need_verifycode"
	QRCodeStatusVerifyCodeBlocked QRCodeStatus = "verify_code_blocked"
	QRCodeStatusScannedRedirect   QRCodeStatus = "scaned_but_redirect"
	QRCodeStatusBoundRedirect     QRCodeStatus = "binded_redirect"
)

// MessageType 表示消息发送方类型。
type MessageType int

const (
	MessageTypeNone MessageType = iota
	MessageTypeUser
	MessageTypeBot
)

// MessageState 表示消息生成状态。
type MessageState int

const (
	MessageStateNew MessageState = iota
	MessageStateGenerating
	MessageStateFinished
)

// MessageItemType 表示消息内容类型。
type MessageItemType int

const (
	MessageItemTypeNone MessageItemType = iota
	MessageItemTypeText
	MessageItemTypeImage
	MessageItemTypeVoice
	MessageItemTypeFile
	MessageItemTypeVideo
	MessageItemTypeToolCallStart  MessageItemType = 11
	MessageItemTypeToolCallResult MessageItemType = 12
)

// MessageID 无损保存微信协议中可能以 JSON 数字或字符串返回的 uint64 ID。
type MessageID string

// UnmarshalJSON 同时接受 JSON 数字和字符串，避免转换为浮点数造成精度损失。
func (id *MessageID) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*id = ""
		return nil
	}
	if data[0] == '"' {
		value, err := strconv.Unquote(string(data))
		if err != nil {
			return errx.Wrap(err, "decode message id")
		}
		*id = MessageID(value)
		return nil
	}
	for _, b := range data {
		if b < '0' || b > '9' {
			return errx.Wrapf(ErrProtocol, "invalid message id %q", data)
		}
	}
	*id = MessageID(data)
	return nil
}

// TextItem 表示文本消息内容。
type TextItem struct {
	Text string `json:"text,omitempty"`
}

// MessageItem 表示单个消息内容项。媒体类型当前仅保留类型信息，不执行下载或解码。
type MessageItem struct {
	Type         MessageItemType `json:"type,omitempty"`
	CreateTimeMS int64           `json:"create_time_ms,omitempty"`
	UpdateTimeMS int64           `json:"update_time_ms,omitempty"`
	Completed    bool            `json:"is_completed,omitempty"`
	MessageID    MessageID       `json:"msg_id,omitempty"`
	Text         *TextItem       `json:"text_item,omitempty"`
}

// Message 表示 getUpdates 返回的微信消息。
type Message struct {
	Sequence     int64         `json:"seq,omitempty"`
	MessageID    MessageID     `json:"message_id,omitempty"`
	FromUserID   string        `json:"from_user_id,omitempty"`
	ToUserID     string        `json:"to_user_id,omitempty"`
	ClientID     string        `json:"client_id,omitempty"`
	CreateTimeMS int64         `json:"create_time_ms,omitempty"`
	UpdateTimeMS int64         `json:"update_time_ms,omitempty"`
	DeleteTimeMS int64         `json:"delete_time_ms,omitempty"`
	SessionID    string        `json:"session_id,omitempty"`
	GroupID      string        `json:"group_id,omitempty"`
	Type         MessageType   `json:"message_type,omitempty"`
	State        MessageState  `json:"message_state,omitempty"`
	Items        []MessageItem `json:"item_list,omitempty"`
	ContextToken string        `json:"context_token,omitempty"`
	RunID        string        `json:"run_id,omitempty"`
}

// GetQRCodeRequest 配置登录二维码请求。
type GetQRCodeRequest struct {
	LocalTokens []string
}

// GetQRCodeResponse 返回二维码值和用于展示的二维码内容。
type GetQRCodeResponse struct {
	QRCode             string `json:"qrcode"`
	QRCodeImageContent string `json:"qrcode_img_content"`
}

// GetQRCodeStatusRequest 配置一次二维码状态长轮询。
type GetQRCodeStatusRequest struct {
	QRCode     string
	VerifyCode string
	BaseURL    string
}

// GetQRCodeStatusResponse 返回当前登录状态及确认后的账号凭证。
type GetQRCodeStatusResponse struct {
	Status       QRCodeStatus `json:"status"`
	BotToken     string       `json:"bot_token,omitempty"`
	BotID        string       `json:"ilink_bot_id,omitempty"`
	BaseURL      string       `json:"baseurl,omitempty"`
	UserID       string       `json:"ilink_user_id,omitempty"`
	RedirectHost string       `json:"redirect_host,omitempty"`
}

// GetUpdatesRequest 配置一次消息长轮询。
type GetUpdatesRequest struct {
	Cursor  string
	Timeout time.Duration
}

// GetUpdatesResponse 返回消息、下次游标和服务端建议的长轮询超时。
type GetUpdatesResponse struct {
	Ret                  *int      `json:"ret,omitempty"`
	ErrorCode            *int      `json:"errcode,omitempty"`
	ErrorMessage         string    `json:"errmsg,omitempty"`
	Messages             []Message `json:"msgs,omitempty"`
	Cursor               string    `json:"get_updates_buf,omitempty"`
	LongPollingTimeoutMS int64     `json:"longpolling_timeout_ms,omitempty"`
}

// LongPollingTimeout 返回服务端建议的下一次消息长轮询超时。
func (r GetUpdatesResponse) LongPollingTimeout() time.Duration {
	if r.LongPollingTimeoutMS <= 0 {
		return 0
	}
	return time.Duration(r.LongPollingTimeoutMS) * time.Millisecond
}

// SendTextRequest 配置一条主动发送或回复的文本消息。
type SendTextRequest struct {
	ToUserID     string
	Text         string
	ContextToken string
	RunID        string
}

// SendTextResponse 返回本地 Client ID 和服务端消息 ID。
type SendTextResponse struct {
	ClientID     string
	MessageID    MessageID
	Ret          *int
	ErrorMessage string
}
