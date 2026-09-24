package weclawbot

import (
	"bytes"
	"io"
	"os"
	"strings"

	"github.com/go-sdk/core/errx"
	"github.com/mdp/qrterminal/v3"
)

// WriteQRCode 将二维码以适合终端显示的半高字符写入目标 Writer。
func (r GetQRCodeResponse) WriteQRCode(writer io.Writer) error {
	if writer == nil {
		return ErrWriterRequired
	}
	content := strings.TrimSpace(r.QRCodeImageContent)
	if content == "" {
		return ErrQRCodeContentRequired
	}
	var buffer bytes.Buffer
	qrterminal.GenerateWithConfig(content, qrterminal.Config{
		Level:      qrterminal.L,
		Writer:     &buffer,
		HalfBlocks: true,
		QuietZone:  2,
	})
	if _, err := io.Copy(writer, &buffer); err != nil {
		return errx.Wrap(err, "write qrcode")
	}
	return nil
}

// PrintQRCode 将二维码输出到标准输出。
func (r GetQRCodeResponse) PrintQRCode() error {
	return r.WriteQRCode(os.Stdout)
}
