package minows

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/testing/fake"
	"testing"
	"time"
)

func Test_session_Send(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator, stop := mustCreateMinows(t, addrInitiator, addrInitiator)
	defer stop()
	r := mustCreateRPC(t, initiator, "test", testHandler{})

	const addrPlayer1 = "/ip4/127.0.0.1/tcp/6002/ws"
	player1, stop := mustCreateMinows(t, addrPlayer1, addrPlayer1)
	defer stop()
	mustCreateRPC(t, player1, "test", testHandler{})

	const addrPlayer2 = "/ip4/127.0.0.1/tcp/6003/ws"
	player2, stop := mustCreateMinows(t, addrPlayer2, addrPlayer2)
	defer stop()
	mustCreateRPC(t, player2, "test", testHandler{})

	s, _, stop := mustCreateSession(t, r, player1, player2)
	defer stop()

	errs := s.Send(fake.Message{}, player1.GetAddress())
	err, open := <-errs
	require.NoError(t, err)
	require.False(t, open)

	errs = s.Send(fake.Message{}, player1.GetAddress(), player2.GetAddress())
	err, open = <-errs
	require.NoError(t, err)
	require.False(t, open)
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

	const addrPlayer1 = "/ip4/127.0.0.1/tcp/6002/ws"
	player1, stop := mustCreateMinows(t, addrPlayer1, addrPlayer1)
	defer stop()
	mustCreateRPC(t, player1, "test", testHandler{})

	const addrPlayer2 = "/ip4/127.0.0.1/tcp/6003/ws"
	player2, stop := mustCreateMinows(t, addrPlayer2, addrPlayer2)
	defer stop()
	mustCreateRPC(t, player2, "test", testHandler{})

	sender, receiver, stop := mustCreateSession(t, r, player1, player2)
	defer stop()

	errs := sender.Send(fake.Message{}, player1.GetAddress())
	require.NoError(t, <-errs)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	from, msg, err := receiver.Recv(ctx)
	require.NoError(t, err)
	require.Equal(t, player1.GetAddress(), from)
	require.Equal(t, fake.Message{}, msg)

	errs = sender.Send(fake.Message{}, player1.GetAddress(),
		player2.GetAddress())
	require.NoError(t, <-errs)

	from1, msg, err := receiver.Recv(ctx)
	require.NoError(t, err)
	require.True(t, from1.Equal(player1.GetAddress()) || from1.Equal(player2.
		GetAddress()))
	require.Equal(t, fake.Message{}, msg)

	from2, msg, err := receiver.Recv(ctx)
	require.NoError(t, err)
	require.True(t, from2.Equal(player1.GetAddress()) || from2.Equal(player2.
		GetAddress()))
	require.Equal(t, fake.Message{}, msg)
	require.NotEqual(t, from1, from2)
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
	require.NoError(t, <-errs)
	stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err := receiver.Recv(ctx)
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
	require.NoError(t, <-errs)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := receiver.Recv(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func mustCreateSession(t *testing.T, rpc mino.RPC,
	minos ...mino.Mino) (mino.Sender, mino.Receiver, func()) {
	ctx, cancel := context.WithCancel(context.Background())
	var addrs []mino.Address
	for _, m := range minos {
		addrs = append(addrs, m.GetAddress())
	}
	players := mino.NewAddresses(addrs...)
	stop := func() {
		cancel()
	}
	s, r, err := rpc.Stream(ctx, players)
	require.NoError(t, err)
	return s, r, stop
}
