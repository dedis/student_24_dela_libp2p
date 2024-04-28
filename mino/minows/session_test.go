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
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	s, _, tearDown := mustCreateSession(t, r, participant)
	defer tearDown()

	errs := s.Send(fake.Message{}, participant.GetAddress())
	err, open := <-errs
	require.NoError(t, err)
	require.False(t, open)
	// 	todo testHandler assert message received as-is

	// todo send to 2 participants
}

func Test_session_Send_WrongAddressType(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	s, _, tearDown := mustCreateSession(t, r, participant)
	defer tearDown()

	errs := s.Send(fake.Message{}, fake.Address{})
	require.ErrorContains(t, <-errs, "wrong address type")
}

func Test_session_Send_AddressNotPlayer(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	s, _, tearDown := mustCreateSession(t, r, participant)
	defer tearDown()
	addr := mustCreateAddress("/ip4/127.0.0.1/tcp/6002/ws",
		"QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU")

	errs := s.Send(fake.Message{}, addr)
	require.ErrorContains(t, <-errs, "not a player")
}

func Test_session_Send_SessionEnded(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	s, _, tearDown := mustCreateSession(t, r, participant)
	tearDown()

	errs := s.Send(fake.Message{}, participant.GetAddress())
	require.ErrorContains(t, <-errs, "session ended")
	_, open := <-errs
	require.False(t, open)
}

func Test_session_Recv(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	sender, receiver, tearDown := mustCreateSession(t, r, participant)
	defer tearDown()
	errs := sender.Send(fake.Message{}, participant.GetAddress())
	require.NoError(t, <-errs) // sent successfully

	from, msg, err := receiver.Recv(context.Background())
	require.NoError(t, err)
	require.Equal(t, participant.GetAddress(), from)
	require.Equal(t, fake.Message{}, msg)

	// todo send to & receive from 2 participants
}

func Test_session_Recv_SessionEnded(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	sender, receiver, tearDown := mustCreateSession(t, r, participant)
	errs := sender.Send(fake.Message{}, participant.GetAddress())
	require.NoError(t, <-errs) // sent successfully
	tearDown()

	_, _, err := receiver.Recv(context.Background())
	require.ErrorContains(t, err, "session ended")
}

func Test_session_Recv_ContextCancelled(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})
	// todo defer tearDown()
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})
	// todo defer tearDown()
	sender, receiver, tearDown := mustCreateSession(t, r, participant)
	defer tearDown()
	errs := sender.Send(fake.Message{}, participant.GetAddress())
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
	tearDown := func() {
		cancel()
	}
	s, r, err := rpc.Stream(ctx, players)
	require.NoError(t, err)
	return s, r, tearDown
}
