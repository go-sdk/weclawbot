package weclawbot

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-sdk/core/testx"
)

const sendNetworkTestEnvironment = "WECHAT_SEND_TESTS"

// TestNetworkWechatSendTextFromEnv 使用环境变量提供的账号和会话参数发送真实文本消息。
func TestNetworkWechatSendTextFromEnv(t *testing.T) {
	if os.Getenv(sendNetworkTestEnvironment) != "1" {
		t.Skip(sendNetworkTestEnvironment + " is not enabled")
	}
	botToken := requiredNetworkEnvironment(t, "WECHAT_BOT_TOKEN")
	toUserID := requiredNetworkEnvironment(t, "WECHAT_TO_USER_ID")
	text := strings.TrimSpace(os.Getenv("WECHAT_TEXT"))
	if text == "" {
		text = "测试消息 " + time.Now().Format(time.RFC3339)
	}
	baseURL := strings.TrimSpace(os.Getenv("WECHAT_BASE_URL"))
	contextToken := strings.TrimSpace(os.Getenv("WECHAT_CONTEXT_TOKEN"))
	runID := strings.TrimSpace(os.Getenv("WECHAT_RUN_ID"))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	options := []Option{WithToken(botToken)}
	if baseURL != "" {
		options = append(options, WithBaseURL(baseURL))
	}
	client := New(options...)
	result, err := client.SendText(ctx, SendTextRequest{
		ToUserID:     toUserID,
		Text:         text,
		ContextToken: contextToken,
		RunID:        runID,
	})
	testx.NoError(t, err)
	testx.NotEmpty(t, result.ClientID)
	t.Logf("message sent: client_id=%s message_id=%s", result.ClientID, result.MessageID)
}

func requiredNetworkEnvironment(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s is required", name)
	}
	return value
}
