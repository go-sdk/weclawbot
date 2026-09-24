# weclawbot

`weclawbot` 是微信 ClawBot HTTP JSON 协议的 Go SDK，提供二维码登录、消息长轮询和文本消息发送能力。

协议参考 [Tencent/openclaw-weixin](https://github.com/Tencent/openclaw-weixin/blob/main/docs/protocol_zh_CN.md)。当前实现基于提交 `24de5c9eb0dd5e595d7e2d090ed8a3f82870d42c` 对应的协议和客户端行为。

## 环境要求

- Go 1.27 或更高版本

## 安装

```bash
go get github.com/go-sdk/weclawbot
```

## 登录流程

先创建匿名客户端并获取二维码：

```go
client := weclawbot.New()

qr, err := client.GetQRCode(ctx, weclawbot.GetQRCodeRequest{})
if err != nil {
	return err
}

fmt.Println(qr.QRCodeImageContent)
```

也可以直接将二维码输出到 Console：

```go
if err := qr.PrintQRCode(); err != nil {
	return err
}
```

`WriteQRCode(writer)` 可以输出到其他 `io.Writer`。二维码编码内容来自服务端返回的
`qrcode_img_content`；二维码本身属于登录凭证，只应输出到可信终端。

调用方展示 `QRCodeImageContent`，随后循环查询状态：

```go
baseURL := ""
verifyCode := ""

for {
	status, err := client.GetQRCodeStatus(ctx, weclawbot.GetQRCodeStatusRequest{
		QRCode:     qr.QRCode,
		VerifyCode: verifyCode,
		BaseURL:    baseURL,
	})
	if err != nil {
		return err
	}

	switch status.Status {
	case weclawbot.QRCodeStatusWait, weclawbot.QRCodeStatusScanned:
		continue
	case weclawbot.QRCodeStatusNeedVerifyCode:
		verifyCode = readVerifyCodeFromUser()
	case weclawbot.QRCodeStatusScannedRedirect:
		baseURL, err = status.RedirectBaseURL()
		if err != nil {
			return err
		}
	case weclawbot.QRCodeStatusConfirmed:
		botClient := weclawbot.New(
			weclawbot.WithBaseURL(status.BaseURL),
			weclawbot.WithToken(status.BotToken),
		)
		return runMessages(ctx, botClient)
	case weclawbot.QRCodeStatusExpired,
		weclawbot.QRCodeStatusVerifyCodeBlocked,
		weclawbot.QRCodeStatusBoundRedirect:
		return errLoginStopped
	}
}
```

SDK 不读取终端输入、不自动刷新二维码，也不保存 `BotToken`。调用方应根据业务界面取得验证码，并安全保存确认响应中的 Token、Bot ID、API 地址和用户 ID。

## 接收消息

`GetUpdates` 每次执行一次长轮询。首次请求使用空游标，之后回传服务端返回的非空游标：

```go
cursor := ""

for {
	updates, err := botClient.GetUpdates(ctx, weclawbot.GetUpdatesRequest{
		Cursor: cursor,
	})
	if err != nil {
		if weclawbot.IsSessionExpired(err) {
			return relogin()
		}
		return err
	}

	cursor = updates.NextCursor(cursor)
	for _, message := range updates.Messages {
		fmt.Println(message.FromUserID, message.ContextToken)
	}
}
```

服务端可能通过 `longpolling_timeout_ms` 建议下一次超时，可使用 `updates.LongPollingTimeout()` 读取。单次内部长轮询超时返回空消息；调用方主动取消仍返回 `context.Canceled` 或 `context.DeadlineExceeded`。

消息中的 `message_id` 和 `msg_id` 使用 `MessageID` 字符串类型，无损兼容服务端返回的 JSON 数字或字符串。

## 发送文本

回复入站消息时，应把发送者 ID 和会话上下文原样带回：

```go
result, err := botClient.SendText(ctx, weclawbot.SendTextRequest{
	ToUserID:     message.FromUserID,
	ContextToken: message.ContextToken,
	RunID:        message.RunID,
	Text:         "收到",
})
if err != nil {
	return err
}

fmt.Println(result.ClientID, result.MessageID)
```

`context_token` 在 SDK 中保持可选，以兼容腾讯当前客户端行为；但协议没有保证服务端一定接受缺少上下文的主动发送，生产环境应通过真实账号验证。

## Client 配置

```go
client := weclawbot.New(
	weclawbot.WithBaseURL("https://ilinkai.weixin.qq.com"),
	weclawbot.WithToken(token),
	weclawbot.WithRouteTag(routeTag),
	weclawbot.WithChannelVersion("1.0.0"),
	weclawbot.WithBotAgent("ExampleBot/1.0.0"),
	weclawbot.WithTimeout(15*time.Second),
	weclawbot.WithLongPollTimeout(35*time.Second),
)
```

未覆盖版本时，SDK 尝试从 Go 构建信息读取自身模块版本，开发态回退为 `0.0.0`。应用请求头中的版本按协议编码为 `0x00MMNNPP` 的十进制字符串。

API 地址默认只允许 HTTPS。仅在可信的开发或测试环境需要连接明文 HTTP 服务时，才显式传入 `weclawbot.WithInsecureHTTP()`；HTTP 会明文传输 Bot Token、用户 ID、消息内容和上下文凭证，不应在生产环境启用。

## 错误处理

错误使用 `core/errx` 判断：

```go
switch {
case errx.Is(err, weclawbot.ErrSessionExpired):
	// Token 已失效，需要重新登录。
case errx.Is(err, weclawbot.ErrRequest):
	// DNS、连接、TLS 或其他请求级失败。
case errx.Is(err, weclawbot.ErrHTTPStatus):
	// 服务端返回非 2xx 状态。
case errx.Is(err, weclawbot.ErrProtocol):
	// 响应缺少必要字段或返回非零业务码。
}
```

SDK 自身的错误信息不会包含 Bot Token、二维码值、验证码、`context_token` 或完整响应体。
Client 保持 `core/restx` 的调试行为；启用调试模式可能输出请求 Header、查询参数或请求体，生产环境应根据敏感信息策略控制调试开关。

## 当前边界

- 支持获取二维码、验证码回传、IDC 重定向和登录确认。
- 支持 `getUpdates` 游标、服务端建议超时和 `-14` 会话失效。
- 支持接收文本消息和发送文本消息。
- 不持久化账号、Token、游标或上下文。
- 不实现图片、语音、文件和视频的上传、下载、解密或发送。
- 不实现 `getConfig`、`sendTyping`、`notifyStart` 或 `notifyStop`。

## 开发

```bash
make lint
make test
```

测试使用本地 HTTP 传输替身，不访问真实微信接口。静态检查和本地测试不能证明真实账号登录、服务端路由或主动发送行为。

显式运行真实微信闭环测试：

```bash
make test-network
```

该测试会访问真实微信接口，在 Console 输出二维码并等待人工扫码；登录成功后需要向 Bot 发送一条消息，测试会使用收到的 `from_user_id/context_token` 回复 `wechat network test: received`。普通 `make test` 不会执行该真实调用流程。

如果已经保存了发送所需参数，可以单独执行环境变量驱动的真实发送测试：

```bash
export WECHAT_BOT_TOKEN="<Bot Token>"
export WECHAT_TO_USER_ID="<目标用户 ID>"
export WECHAT_TEXT="<可选，消息内容>"
export WECHAT_BASE_URL="https://<可选，登录返回的 API 地址>"
export WECHAT_CONTEXT_TOKEN="<可选，最新 Context Token>"
export WECHAT_RUN_ID="<可选 Run ID>"

make test-send
```

`WECHAT_BOT_TOKEN` 和 `WECHAT_TO_USER_ID` 为必填。`WECHAT_BASE_URL` 为空时使用 `https://ilinkai.weixin.qq.com`；`WECHAT_TEXT` 为空时发送 `测试消息 ` 加当前 RFC3339 时间，例如 `测试消息 2026-09-24T14:30:25+08:00`；`WECHAT_CONTEXT_TOKEN` 和 `WECHAT_RUN_ID` 可以为空。测试只记录返回的 Client ID 和 Message ID，不输出 Token、用户 ID 或 Context Token。该命令会向真实微信用户发送消息，普通 `make test` 不会执行。
