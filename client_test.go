package weclawbot

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-sdk/core/errx"
	"github.com/go-sdk/core/restx"
	"github.com/go-sdk/core/testx"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestWriteQRCode(t *testing.T) {
	var output bytes.Buffer
	result := GetQRCodeResponse{QRCodeImageContent: "https://example.com/qr"}
	testx.NoError(t, result.WriteQRCode(&output))
	testx.NotEmpty(t, output.String())

	testx.ErrorIs(t, (GetQRCodeResponse{}).WriteQRCode(&output), ErrQRCodeContentRequired)
	testx.ErrorIs(t, result.WriteQRCode(nil), ErrWriterRequired)
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func newTestClient(handler roundTripFunc, options ...Option) *Client {
	httpClient := restx.New().SetTransport(handler)
	options = append([]Option{WithClient(httpClient), WithChannelVersion("2.4.9")}, options...)
	return New(options...)
}

func TestGetQRCode(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		testx.Equal(t, http.MethodPost, request.Method)
		testx.Equal(t, "ilinkai.weixin.qq.com", request.URL.Host)
		testx.Equal(t, "/ilink/bot/get_bot_qrcode", request.URL.Path)
		testx.Equal(t, "3", request.URL.Query().Get("bot_type"))
		testx.Equal(t, "ilink_bot_token", request.Header.Get(headerAuthorizationType))
		testx.NotEmpty(t, request.Header.Get(headerWechatUIN))
		testx.Empty(t, request.Header.Get("Authorization"))
		body, err := io.ReadAll(request.Body)
		testx.NoError(t, err)
		testx.Equal(t, `{"local_token_list":[]}`, string(body))
		return response(http.StatusOK, `{"qrcode":"qr-value","qrcode_img_content":"https://example.com/qr"}`), nil
	})

	result, err := client.GetQRCode(context.Background(), GetQRCodeRequest{})
	testx.NoError(t, err)
	testx.Equal(t, "qr-value", result.QRCode)
	testx.Equal(t, "https://example.com/qr", result.QRCodeImageContent)
}

func TestGetQRCodeStatus(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		testx.Equal(t, http.MethodGet, request.Method)
		testx.Equal(t, "poll.example.com", request.URL.Host)
		testx.Equal(t, "qr value", request.URL.Query().Get("qrcode"))
		testx.Equal(t, "123456", request.URL.Query().Get("verify_code"))
		testx.Empty(t, request.Header.Get(headerAuthorizationType))
		testx.Empty(t, request.Header.Get(headerWechatUIN))
		return response(http.StatusOK, `{"status":"scaned_but_redirect","redirect_host":"redirect.example.com"}`), nil
	})

	result, err := client.GetQRCodeStatus(context.Background(), GetQRCodeStatusRequest{
		QRCode:     "qr value",
		VerifyCode: "123456",
		BaseURL:    "https://poll.example.com",
	})
	testx.NoError(t, err)
	testx.Equal(t, QRCodeStatusScannedRedirect, result.Status)
	redirect, err := result.RedirectBaseURL()
	testx.NoError(t, err)
	testx.Equal(t, "https://redirect.example.com", redirect)
}

func TestGetUpdates(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		testx.Equal(t, http.MethodPost, request.Method)
		testx.Equal(t, "Bearer bot-token", request.Header.Get("Authorization"))
		body, err := io.ReadAll(request.Body)
		testx.NoError(t, err)
		testx.Contains(t, string(body), `"get_updates_buf":"cursor-1"`)
		testx.Contains(t, string(body), `"channel_version":"2.4.9"`)
		return response(http.StatusOK, `{"ret":0,"msgs":[{"message_id":18446744073709551615,"from_user_id":"user-1","context_token":"context-1","item_list":[{"type":1,"text_item":{"text":"hello"}}]}],"get_updates_buf":"cursor-2","longpolling_timeout_ms":35000}`), nil
	}, WithBaseURL("https://api.example.com"), WithToken("bot-token"))

	result, err := client.GetUpdates(context.Background(), GetUpdatesRequest{Cursor: "cursor-1"})
	testx.NoError(t, err)
	testx.Len(t, result.Messages, 1)
	testx.Equal(t, MessageID("18446744073709551615"), result.Messages[0].MessageID)
	testx.Equal(t, "user-1", result.Messages[0].FromUserID)
	testx.Equal(t, "context-1", result.Messages[0].ContextToken)
	testx.Equal(t, "cursor-2", result.NextCursor("cursor-1"))
}

func TestGetUpdatesSessionExpired(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		return response(http.StatusOK, `{"ret":-14,"errcode":-14,"errmsg":"expired"}`), nil
	}, WithBaseURL("https://api.example.com"), WithToken("bot-token"))

	result, err := client.GetUpdates(context.Background(), GetUpdatesRequest{})
	testx.NotNil(t, result)
	testx.ErrorIs(t, err, ErrSessionExpired)
	testx.True(t, IsSessionExpired(err))
}

func TestReceiveMessages(t *testing.T) {
	requests := 0
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		requests++
		body, err := io.ReadAll(request.Body)
		testx.NoError(t, err)
		switch requests {
		case 1:
			testx.Contains(t, string(body), `"get_updates_buf":""`)
			return response(http.StatusOK, `{"ret":0,"msgs":[],"get_updates_buf":"cursor-2","longpolling_timeout_ms":500}`), nil
		case 2:
			testx.Contains(t, string(body), `"get_updates_buf":"cursor-2"`)
			deadline, ok := request.Context().Deadline()
			testx.True(t, ok)
			testx.True(t, time.Until(deadline) <= time.Second)
			return response(http.StatusOK, `{"ret":0,"msgs":[{"message_id":"1"},{"message_id":"2"}],"get_updates_buf":"cursor-3"}`), nil
		default:
			t.Fatal("unexpected getUpdates request")
			return nil, nil
		}
	}, WithBaseURL("https://api.example.com"), WithToken("bot-token"))

	stop := errx.New("stop receiving")
	received := []MessageID{}
	err := client.ReceiveMessages(context.Background(), func(_ context.Context, message Message) error {
		received = append(received, message.MessageID)
		if message.MessageID == "2" {
			return stop
		}
		return nil
	})
	testx.ErrorIs(t, err, stop)
	testx.Equal(t, []MessageID{"1", "2"}, received)
	testx.Equal(t, 2, requests)
}

func TestReceiveMessagesCanceled(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		return nil, request.Context().Err()
	}, WithBaseURL("https://api.example.com"), WithToken("bot-token"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.ReceiveMessages(ctx, func(context.Context, Message) error {
		t.Fatal("message handler must not be called")
		return nil
	})
	testx.ErrorIs(t, err, context.Canceled)
}

func TestHTTPBaseURLRequiresExplicitOption(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		t.Fatal("HTTP request must not be sent without WithInsecureHTTP")
		return nil, nil
	}, WithBaseURL("http://api.example.com"), WithToken("bot-token"))

	_, err := client.GetUpdates(context.Background(), GetUpdatesRequest{})
	testx.ErrorIs(t, err, ErrInvalidURL)

	client = newTestClient(func(request *http.Request) (*http.Response, error) {
		testx.Equal(t, "http", request.URL.Scheme)
		return response(http.StatusOK, `{"ret":0,"msgs":[]}`), nil
	}, WithBaseURL("http://api.example.com"), WithToken("bot-token"), WithInsecureHTTP())

	_, err = client.GetUpdates(context.Background(), GetUpdatesRequest{})
	testx.NoError(t, err)
}

func TestSendText(t *testing.T) {
	client := newTestClient(func(request *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(request.Body)
		testx.NoError(t, err)
		text := string(body)
		testx.Contains(t, text, `"from_user_id":""`)
		testx.Contains(t, text, `"to_user_id":"user-1"`)
		testx.Contains(t, text, `"message_type":2`)
		testx.Contains(t, text, `"message_state":2`)
		testx.Contains(t, text, `"context_token":"context-1"`)
		testx.Contains(t, text, `"text":"notice"`)
		return response(http.StatusOK, `{"message_id":"18446744073709551615","ret":0}`), nil
	}, WithBaseURL("https://api.example.com"), WithToken("bot-token"))

	result, err := client.SendText(context.Background(), SendTextRequest{
		ToUserID:     "user-1",
		Text:         "notice",
		ContextToken: "context-1",
	})
	testx.NoError(t, err)
	testx.NotEmpty(t, result.ClientID)
	testx.Equal(t, MessageID("18446744073709551615"), result.MessageID)
}

func TestValidation(t *testing.T) {
	client := New()
	_, err := client.GetQRCodeStatus(context.Background(), GetQRCodeStatusRequest{})
	testx.ErrorIs(t, err, ErrQRCodeRequired)

	_, err = client.GetUpdates(context.Background(), GetUpdatesRequest{})
	testx.ErrorIs(t, err, ErrTokenRequired)

	_, err = New(WithToken("token")).SendText(context.Background(), SendTextRequest{})
	testx.ErrorIs(t, err, ErrUserIDRequired)
	testx.False(t, errx.Is(err, ErrTextRequired))

	err = New(WithToken("token")).ReceiveMessages(context.Background(), nil)
	testx.ErrorIs(t, err, ErrMessageHandlerRequired)
}
