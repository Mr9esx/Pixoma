package feishu

import (
	"context"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
)

// defaultLongConn builds the Feishu official WebSocket connection with
// automatic reconnect and forwards inbound message events to onMsg.
func defaultLongConn(appID, appSecret string, onMsg func(context.Context, *larkim.P2MessageReceiveV1)) (LongConn, error) {
	dis := newMessageDispatcher(onMsg)
	cli := ws.NewClient(appID, appSecret, ws.WithEventHandler(dis), ws.WithAutoReconnect(true))
	return &larkLongConn{cli: cli}, nil
}

type larkLongConn struct {
	cli *ws.Client
}

func (c *larkLongConn) Start(ctx context.Context) error {
	return c.cli.Start(ctx)
}

func newMessageDispatcher(onMsg func(context.Context, *larkim.P2MessageReceiveV1)) *dispatcher.EventDispatcher {
	dis := dispatcher.NewEventDispatcher("", "")
	dis.OnP2MessageReceiveV1(func(ctx context.Context, ev *larkim.P2MessageReceiveV1) error {
		onMsg(ctx, ev)
		return nil
	})
	return dis
}
