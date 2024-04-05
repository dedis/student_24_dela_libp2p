package minows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/testing/fake"
)

func TestIntegration_Scenario_Call(t *testing.T) {
	const N = 3
	minos := make([]mino.Mino, N)
	rpcs := make([]mino.RPC, N)
	for i := 0; i < N; i++ {
		minos[i], rpcs[i] = newMinowsAndRPC()
	}

	authority := fake.NewAuthorityFromMino(fake.NewSigner, minos...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out, err := rpcs[0].Call(ctx, fake.Message{}, authority)
	require.NoError(t, err)

	for {
		select {
		case resp, open := <-out:
			if !open {
				// TODO verify no error
				return
			}
			// TODO verify responses
			msg, err := resp.GetMessageOrError()
			require.NoError(t, err)
			require.Equal(t, fake.Message{}, msg)
		case <-time.After(5 * time.Second): // hard time-out
			t.Fatal("timeout waiting for closure")
		}
	}
}

func newMinowsAndRPC() (mino.Mino, mino.RPC) {
	m, err := NewMinows("ip4/127.0.0.1/tcp/9998/ws", "ip4/127.0.0.1/tcp/9998/ws",
		"QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU")
	if err != nil {
		panic(err)
	}
	rpc, err := m.CreateRPC("test", mino.UnsupportedHandler{}, fake.MessageFactory{})
	if err != nil {
		panic(err)
	}
	return m, rpc
}
