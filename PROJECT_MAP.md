# 项目地图

## 项目定位

`github.com/go-sdk/weclawbot` 是微信 ClawBot HTTP JSON 协议客户端，提供二维码登录、消息长轮询和文本消息发送能力。SDK 不管理账号文件；`ReceiveMessages` 在调用期间管理长轮询和游标，调用方负责保存凭证和消息上下文。

## 目录结构

```text
weclawbot/
├── .github/workflows/golang.yml     持续集成与 Tag Release
├── AGENTS.md                        仓库协作、安全和验证规范
├── client.go                        Client、默认值和 Options
├── error.go                         可由 errx.Is 判断的公共哨兵错误
├── login.go                         获取二维码和查询登录状态
├── model.go                         协议请求、响应和消息模型
├── protocol.go                      公共请求头、URL、JSON 和错误处理
├── qrcode.go                        Console 二维码输出
├── send.go                          文本消息构造和发送
├── updates.go                       消息长轮询和游标处理
├── version.go                       SDK 版本和客户端版本编码
├── client_test.go                   本地 HTTP 传输替身行为测试
├── network_test.go                  显式启用的真实微信闭环测试
├── send_network_test.go             环境变量驱动的真实文本发送测试
├── doc.go                           根包说明
├── README.md                        安装、流程和公共 API 说明
├── Makefile                         依赖整理、检查和测试入口
└── go.mod                           Go 模块和依赖定义
```

## 登录链路

```text
Client.GetQRCode
    -> POST 固定入口 /ilink/bot/get_bot_qrcode?bot_type=3
    -> 返回 qrcode 和 qrcode_img_content

Client.GetQRCodeStatus
    -> GET /ilink/bot/get_qrcode_status
    -> wait / scaned / need_verifycode / expired
    -> scaned_but_redirect 时校验 redirect_host 并切换轮询地址
    -> confirmed 时返回 bot_token、ilink_bot_id、baseurl、ilink_user_id
```

SDK 每次只执行一次状态长轮询，不自动读取终端输入、不自动刷新二维码，也不保存登录结果。调用方可以通过 `PrintQRCode` 输出 Console 二维码，根据状态决定下一次请求，并通过 `VerifyCode` 回传手机显示的验证码。

## 消息链路

```text
Client.GetUpdates
    -> POST /ilink/bot/getupdates
    -> 回传上次 get_updates_buf
    -> 返回消息、下一游标和建议超时
    -> 从消息取得 from_user_id 和 context_token

Client.ReceiveMessages
    -> 循环调用 Client.GetUpdates
    -> 自动维护非空游标和服务端建议超时
    -> 按顺序调用消息处理函数
    -> 处理错误、查询错误或 Context 结束时退出

Client.SendText
    -> core/seq.UUID 生成 client_id
    -> 构造 Bot 完成态文本消息
    -> POST /ilink/bot/sendmessage
    -> 返回本地 client_id 和服务端 message_id
```

`GetUpdatesResponse.NextCursor` 只接受服务端非空游标。`ReceiveMessages` 只在当前调用期间保存游标，不持久化状态，也不隐藏查询或消息处理错误。`ret` 或 `errcode` 为 `-14` 时返回 `ErrSessionExpired`，由调用方决定重新登录或退避。

## 公共请求行为

- `core/restx` 提供统一 HTTP Transport、代理、连接池、Cookie Jar 和日志。
- `core/codec/json` 负责编解码协议 JSON。
- `core/errx` 负责创建、包装和判断错误。
- `core/seq` 生成文本消息的 UUID v7 Client ID。
- 普通请求默认 15 秒超时，二维码状态和消息查询默认 35 秒。
- API 地址默认只允许 HTTPS；可信开发或测试环境必须显式配置 `WithInsecureHTTP()` 才允许 HTTP。
- 内部长轮询超时转换为空结果；调用方取消仍原样返回取消错误。
- 非 2xx HTTP 状态与业务返回码分别处理，错误不包含凭证或完整响应体。
- Client 保持 `core/restx` 的调试配置；调用方启用调试模式时必须自行控制包含 Header、二维码查询参数和消息上下文的输出范围。

## 支持边界

当前支持二维码登录、验证码回传、IDC 重定向、消息长轮询、文本消息接收和文本发送。媒体消息只保留消息项类型，不实现 CDN 上传下载、AES-128-ECB、缩略图或媒体发送。

## 验证边界

- `make lint` 先整理依赖，再运行 golangci-lint。
- `make test` 使用竞态检测运行本地确定性测试，不访问真实微信接口。
- `make test-network` 访问真实微信接口，需要人工扫码和发送消息，成功后会回复一条测试文本。
- `make test-send` 从 `WECHAT_*` 环境变量读取已有凭证和会话参数，只执行一次真实文本发送。
- `go build ./...` 只验证当前包可编译，不能证明二维码登录、Token、长轮询或消息发送在真实服务端可用。
