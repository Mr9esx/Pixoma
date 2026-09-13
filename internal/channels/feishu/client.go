package feishu

import (
	"bytes"
	"context"
	"fmt"
	"io"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Image is an outbound image blob ready to be uploaded and sent.
type Image struct {
	Data    []byte
	MIME    string
	Caption string
}

// IMClient sends messages and downloads inbound media for a Feishu app.
type IMClient interface {
	SendText(ctx context.Context, chatID, text string) error
	SendImage(ctx context.Context, chatID string, img Image) error
	DownloadImage(ctx context.Context, messageID, imageKey string) ([]byte, error)
}

// larkIMClient adapts the lark oapi-sdk-go client to IMClient.
type larkIMClient struct {
	cli *lark.Client
}

// NewIMClient builds an IMClient bound to the Feishu app credentials.
func NewIMClient(appID, appSecret string) IMClient {
	return &larkIMClient{cli: lark.NewClient(appID, appSecret)}
}

func (c *larkIMClient) SendText(ctx context.Context, chatID, text string) error {
	body := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(chatID).
		MsgType("text").
		Content(fmt.Sprintf(`{"text":%q}`, text)).
		Build()
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(body).
		Build()
	_, err := c.cli.Im.V1.Message.Create(ctx, req)
	return err
}

func (c *larkIMClient) SendImage(ctx context.Context, chatID string, img Image) error {
	up, err := c.cli.Im.V1.Image.Create(ctx, larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType("message").
			Image(bytes.NewReader(img.Data)).
			Build()).
		Build())
	if err != nil {
		return err
	}
	if !up.Success() {
		return fmt.Errorf("feishu image upload: %w", up.CodeError)
	}
	key := ""
	if up.Data != nil && up.Data.ImageKey != nil {
		key = *up.Data.ImageKey
	}
	if key == "" {
		return fmt.Errorf("feishu image upload: empty image_key")
	}
	msgBody := larkim.NewCreateMessageReqBodyBuilder().
		ReceiveId(chatID).
		MsgType("image").
		Content(fmt.Sprintf(`{"image_key":%q}`, key)).
		Build()
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(msgBody).
		Build()
	_, err = c.cli.Im.V1.Message.Create(ctx, req)
	return err
}

func (c *larkIMClient) DownloadImage(ctx context.Context, messageID, imageKey string) ([]byte, error) {
	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(messageID).
		FileKey(imageKey).
		Type("image").
		Build()
	resp, err := c.cli.Im.V1.MessageResource.Get(ctx, req)
	if err != nil {
		return nil, err
	}
	if !resp.Success() {
		return nil, fmt.Errorf("feishu image download: %w", resp.CodeError)
	}
	return io.ReadAll(resp.File)
}
