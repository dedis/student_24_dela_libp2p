package minows

import (
	"context"

	"github.com/libp2p/go-libp2p/core/protocol"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
)

// RPC implements mino.RPC
type RPC struct {
	pid protocol.ID
	// handler mino.Handler
	// factory serde.Factory
}

func (R RPC) Call(
	ctx context.Context,
	req serde.Message,
	players mino.Players,
) (<-chan mino.Response, error) {
	// TODO implement me
	panic("implement me")
}

func (R RPC) Stream(ctx context.Context, players mino.Players) (mino.Sender, mino.Receiver, error) {
	// TODO implement me
	panic("implement me")
}
