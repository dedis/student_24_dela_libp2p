package minows

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/testing/fake"
	"testing"
)

func Test_session_Send(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	s, _, stop := mustCreateSession(t, r, player)
	defer stop()

	errs := s.Send(fake.Message{}, player.GetAddress())
	err, open := <-errs
	require.NoError(t, err)
	require.False(t, open)
	// TODO testHandler assert message received as-is

	// TODO send to 2 participants
}

func Test_session_Send_WrongAddressType(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	s, _, stop := mustCreateSession(t, r, player)
	defer stop()

	errs := s.Send(fake.Message{}, fake.Address{})
	require.ErrorContains(t, <-errs, "wrong address type")
}

func Test_session_Send_AddressNotPlayer(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	s, _, stop := mustCreateSession(t, r, player)
	defer stop()
	notPlayer := mustCreateAddress(t, "/ip4/127.0.0.1/tcp/6003/ws",
		"QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU")

	errs := s.Send(fake.Message{}, notPlayer)
	require.ErrorContains(t, <-errs, "not a player")
}

func Test_session_Send_SessionEnded(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	s, _, stop := mustCreateSession(t, r, player)
	stop()

	errs := s.Send(fake.Message{}, player.GetAddress())
	require.ErrorContains(t, <-errs, "session ended")
	_, open := <-errs
	require.False(t, open)
}

func Test_session_Recv(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	sender, receiver, stop := mustCreateSession(t, r, player)
	defer stop()
	errs := sender.Send(fake.Message{}, player.GetAddress())
	require.NoError(t, <-errs) // sent successfully

	from, msg, err := receiver.Recv(context.Background())
	require.NoError(t, err)
	require.Equal(t, player.GetAddress(), from)
	require.Equal(t, fake.Message{}, msg)

	// TODO send to & receive from 2 participants
}

func Test_session_Recv_SessionEnded(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	sender, receiver, stop := mustCreateSession(t, r, player)
	errs := sender.Send(fake.Message{}, player.GetAddress())
	require.NoError(t, <-errs) // sent successfully
	stop()

	_, _, err := receiver.Recv(context.Background())
	require.ErrorContains(t, err, "session ended")
}

func Test_session_Recv_ContextCancelled(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer = "/ip4/127.0.0.1/tcp/6002/ws"
	player, stop := mustCreateMinows(t, addrPlayer, addrPlayer)
	defer stop()
	mustCreateRPC(t, player, "test", testHandler{})

	sender, receiver, stop := mustCreateSession(t, r, player)
	defer stop()
	errs := sender.Send(fake.Message{}, player.GetAddress())
	require.NoError(t, <-errs) // sent successfully

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := receiver.Recv(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func mustCreateSession(t *testing.T, rpc mino.RPC,
	player *minows) (mino.Sender, mino.Receiver, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses(player.GetAddress())
	stop := func() {
		cancel()
	}
	s, r, err := rpc.Stream(ctx, players)
	require.NoError(t, err)
	return s, r, stop
}
