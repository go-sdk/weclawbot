package weclawbot

import "github.com/go-sdk/core/errx"

var (
	// ErrTokenRequired 表示当前操作必须提供二维码登录取得的 Bot Token。
	ErrTokenRequired = errx.New("bot token is required")
	// ErrQRCodeRequired 表示二维码状态查询缺少二维码值。
	ErrQRCodeRequired = errx.New("qrcode is required")
	// ErrQRCodeContentRequired 表示二维码响应缺少可供编码展示的内容。
	ErrQRCodeContentRequired = errx.New("qrcode content is required")
	// ErrWriterRequired 表示二维码输出缺少目标 Writer。
	ErrWriterRequired = errx.New("writer is required")
	// ErrUserIDRequired 表示文本消息缺少目标用户 ID。
	ErrUserIDRequired = errx.New("user id is required")
	// ErrTextRequired 表示文本消息内容为空。
	ErrTextRequired = errx.New("text is required")
	// ErrMessageHandlerRequired 表示持续接收消息时缺少消息处理函数。
	ErrMessageHandlerRequired = errx.New("message handler is required")
	// ErrTooManyLocalTokens 表示二维码请求携带了超过协议上限的本地 Token。
	ErrTooManyLocalTokens = errx.New("too many local tokens")
	// ErrInvalidURL 表示 API 地址或重定向地址不安全或无效。
	ErrInvalidURL = errx.New("invalid api url")
	// ErrRequest 表示请求未获得有效 HTTP 响应，底层错误可能包含敏感查询参数而不会透出。
	ErrRequest = errx.New("wechat request failed")
	// ErrHTTPStatus 表示微信接口返回了非成功 HTTP 状态。
	ErrHTTPStatus = errx.New("unexpected http status")
	// ErrProtocol 表示微信接口响应缺失必要字段或返回业务错误。
	ErrProtocol = errx.New("wechat protocol error")
	// ErrSessionExpired 表示 Bot Token 已失效，需要重新登录。
	ErrSessionExpired = errx.New("wechat session expired")
)
