package weclawbot

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const networkTestEnvironment = "WECHAT_NETWORK_TESTS"

// TestNetworkWechatTextFlow 通过真实微信接口验证扫码登录、收取消息和文本回复。
// 测试需要人工扫码并向 Bot 发送一条消息，只在显式启用时运行。
func TestNetworkWechatTextFlow(t *testing.T) {
	if os.Getenv(networkTestEnvironment) != "1" {
		t.Skip(networkTestEnvironment + " is not enabled")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	loginClient := New()
	qr := requestNetworkQRCode(t, ctx, loginClient)
	pollBaseURL := ""
	verifyCode := ""
	reader := bufio.NewReader(os.Stdin)

	var login *GetQRCodeStatusResponse
	for login == nil {
		status, err := loginClient.GetQRCodeStatus(ctx, GetQRCodeStatusRequest{
			QRCode:     qr.QRCode,
			VerifyCode: verifyCode,
			BaseURL:    pollBaseURL,
		})
		if err != nil {
			t.Fatalf("get qrcode status: %v", err)
		}
		switch status.Status {
		case QRCodeStatusWait, QRCodeStatusScanned:
		case QRCodeStatusNeedVerifyCode:
			if _, err := fmt.Fprint(os.Stdout, "请输入手机微信显示的验证码："); err != nil {
				t.Fatalf("write verify code prompt: %v", err)
			}
			value, readErr := reader.ReadString('\n')
			if readErr != nil {
				t.Fatalf("read verify code: %v", readErr)
			}
			verifyCode = strings.TrimSpace(value)
		case QRCodeStatusScannedRedirect:
			pollBaseURL, err = status.RedirectBaseURL()
			if err != nil {
				t.Fatalf("resolve redirect base url: %v", err)
			}
		case QRCodeStatusExpired:
			qr = requestNetworkQRCode(t, ctx, loginClient)
			pollBaseURL = ""
			verifyCode = ""
		case QRCodeStatusConfirmed:
			login = status
		case QRCodeStatusVerifyCodeBlocked:
			t.Fatal("verify code is blocked; rerun the test to request a new qrcode")
		case QRCodeStatusBoundRedirect:
			t.Fatal("the bot is already bound and the API did not return new credentials")
		default:
			t.Fatalf("unsupported qrcode status: %s", status.Status)
		}
	}

	botClient := New(
		WithBaseURL(login.BaseURL),
		WithToken(login.BotToken),
	)
	if _, err := fmt.Fprintln(os.Stdout, "登录成功，请在微信中向 Bot 发送一条文本消息。"); err != nil {
		t.Fatalf("write login result: %v", err)
	}

	cursor := ""
	for {
		updates, err := botClient.GetUpdates(ctx, GetUpdatesRequest{Cursor: cursor})
		if err != nil {
			t.Fatalf("get updates: %v", err)
		}
		cursor = updates.NextCursor(cursor)
		for _, message := range updates.Messages {
			if message.FromUserID == "" || message.ContextToken == "" {
				continue
			}
			_, err = botClient.SendText(ctx, SendTextRequest{
				ToUserID:     message.FromUserID,
				Text:         "wechat network test: received",
				ContextToken: message.ContextToken,
				RunID:        message.RunID,
			})
			if err != nil {
				t.Fatalf("send text: %v", err)
			}
			return
		}
	}
}

func requestNetworkQRCode(t *testing.T, ctx context.Context, client *Client) *GetQRCodeResponse {
	t.Helper()
	qr, err := client.GetQRCode(ctx, GetQRCodeRequest{})
	if err != nil {
		t.Fatalf("get qrcode: %v", err)
	}
	if err := qr.PrintQRCode(); err != nil {
		t.Fatalf("print qrcode: %v", err)
	}
	return qr
}
