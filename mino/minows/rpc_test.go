package minows

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"go.dedis.ch/dela/testing/fake"
	"testing"
)

func Test_rpc_Call(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses(player.GetAddress())

	responses, err := r.Call(ctx, req, players)
	require.NoError(t, err)
	resp := <-responses
	from := resp.GetFrom().(address)
	require.Equal(t, player.GetAddress(), from)
	msg, err := resp.GetMessageOrError()
	require.NoError(t, err)
	require.Equal(t, fake.Message{}, msg)
	_, ok := <-responses
	require.False(t, ok)
}

func Test_rpc_Call_NoPlayers(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses()

	_, err := r.Call(ctx, req, players)
	require.ErrorContains(t, err, "no players")
}

func Test_rpc_Call_WrongAddressType(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses(fake.Address{})

	_, err := r.Call(ctx, req, players)
	require.ErrorContains(t, err, "wrong address type")
}

func Test_rpc_Call_DiffNamespace(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player.WithSegment("segment"), "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses(player.GetAddress())

	responses, err := r.Call(ctx, req, players)
	require.NoError(t, err)
	resp := <-responses
	from := resp.GetFrom().(address)
	require.Equal(t, player.GetAddress(), from)
	_, err = resp.GetMessageOrError()
	require.ErrorContains(t, err, "protocols not supported")
	_, ok := <-responses
	require.False(t, ok)
}

func Test_rpc_Call_SelfDial(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses(initiator.GetAddress())

	responses, err := r.Call(ctx, req, players)
	require.NoError(t, err)
	resp := <-responses
	from := resp.GetFrom().(address)
	require.Equal(t, initiator.GetAddress(), from)
	_, err = resp.GetMessageOrError()
	require.ErrorContains(t, err, "dial to self attempted")
	_, ok := <-responses
	require.False(t, ok)
}

func Test_rpc_Call_ContextCancelled(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	req := fake.Message{}
	players := mino.NewAddresses(player.GetAddress())

	cancel()
	responses, _ := r.Call(ctx, req, players)
	// One response may still come through if task completes in another
	// goroutine due to context switching,
	// ang Go randomly executes a select case
	<-responses
	_, ok := <-responses
	require.False(t, ok)
}

func Test_rpc_Stream(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	players := mino.NewAddresses(player.GetAddress())

	sender, receiver, err := r.Stream(ctx, players)
	require.NoError(t, err)
	require.NotNil(t, sender)
	require.NotNil(t, receiver)
}

func Test_rpc_Stream_NoPlayers(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	players := mino.NewAddresses()

	_, _, err := r.Stream(ctx, players)
	require.ErrorContains(t, err, "no players")
}

func Test_rpc_Stream_WrongAddressType(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	players := mino.NewAddresses(fake.Address{})

	_, _, err := r.Stream(ctx, players)
	require.ErrorContains(t, err, "wrong address type")
}

func Test_rpc_Stream_SelfDial(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	players := mino.NewAddresses(initiator.GetAddress())

	_, _, err := r.Stream(ctx, players)
	require.ErrorContains(t, err, "dial to self attempted")
}

func Test_rpc_Stream_ContextCancelled(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})
	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses(player.GetAddress())

	cancel()

	_, _, err := r.Stream(ctx, players)
	require.Error(t, err)
}

// testHandler implements mino.Handler
// Captures received requests for test assertions and
// echos back the same message
type testHandler struct{}

func (e testHandler) Process(req mino.Request) (resp serde.Message, err error) {
	return req.Message, nil
}

func (e testHandler) Stream(out mino.Sender, in mino.Receiver) error {
	for {
		from, msg, err := in.Recv(context.Background())
		if err != nil {
			return err
		}
		err = <-out.Send(msg, from)
		if err != nil {
			return err
		}
	}
}

func mustCreateRPC(t *testing.T, m mino.Mino, name string,
	h mino.Handler) mino.RPC {
	r, err := m.CreateRPC(name, h, fake.MessageFactory{})
	require.NoError(t, err)
	return r
}
