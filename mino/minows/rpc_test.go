package minows

import (
	"context"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"go.dedis.ch/dela/testing/fake"
	"testing"
)

// todo cancel context, then tear down/clean up after each test
// todo parallelize tests: listen on a random port in each test,
//  pass this port to `public` dial address

func Test_rpc_Call(t *testing.T) {
	// todo random port?
	_, r := mustCreateRPCOld("/ip4/127.0.0.1/tcp/5921/ws",
		nil, "test", testHandler{}) // initiator
	participant, _ := mustCreateRPCOld("/ip4/127.0.0.1/tcp/1259/ws",
		nil, "test", testHandler{})
	ctx := context.Background()
	// todo defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses(participant.GetAddress())

	got, err := r.Call(ctx, req, players)
	require.NoError(t, err)
	resp := <-got
	from := resp.GetFrom().(address)
	require.Equal(t, participant.GetAddress(), from)
	msg, err := resp.GetMessageOrError()
	require.NoError(t, err)
	require.Equal(t, fake.Message{}, msg)
	_, ok := <-got
	require.False(t, ok)
}

// todo test no players

func Test_rpc_Call_WrongAddressType(t *testing.T) {
	_, r := mustCreateRPCOld("/ip4/127.0.0.1/tcp/6001/ws",
		nil, "test", testHandler{}) // initiator
	ctx := context.Background()
	req := fake.Message{}
	players := mino.NewAddresses(fake.Address{})

	_, err := r.Call(ctx, req, players)
	require.ErrorContains(t, err, "wrong address type")
}

func Test_rpc_Call_DiffNamespace(t *testing.T) {
	_, r := mustCreateRPCOld("/ip4/127.0.0.1/tcp/6001/ws",
		nil, "test", testHandler{}) // initiator
	participant, _ := mustCreateRPCOld("/ip4/127.0.0.1/tcp/1259/ws",
		[]string{"segment"}, "test", testHandler{})
	ctx := context.Background()
	req := fake.Message{}
	players := mino.NewAddresses(participant.GetAddress())

	got, err := r.Call(ctx, req, players)
	require.NoError(t, err)
	resp := <-got
	from := resp.GetFrom().(address)
	require.Equal(t, participant.GetAddress(), from)
	_, err = resp.GetMessageOrError()
	require.ErrorContains(t, err, "protocols not supported")
	_, ok := <-got
	require.False(t, ok)
}

func Test_rpc_Call_SelfDial(t *testing.T) {
	initiator, r := mustCreateRPCOld("/ip4/127.0.0.1/tcp/5922/ws",
		nil, "test", testHandler{}) // initiator
	ctx := context.Background()
	req := fake.Message{}
	players := mino.NewAddresses(initiator.GetAddress())

	got, err := r.Call(ctx, req, players)
	require.NoError(t, err)
	resp := <-got
	from := resp.GetFrom().(address)
	require.Equal(t, initiator.GetAddress(), from)
	_, err = resp.GetMessageOrError()
	require.ErrorContains(t, err, "dial to self attempted")
	_, ok := <-got
	require.False(t, ok)
}

func Test_rpc_Call_ContextCancelled(t *testing.T) {
	_, r := mustCreateRPCOld("/ip4/127.0.0.1/tcp/6001/ws",
		nil, "test", testHandler{}) // initiator
	participant, _ := mustCreateRPCOld("/ip4/127.0.0.1/tcp/1259/ws",
		nil, "test", testHandler{})
	ctx, cancel := context.WithCancel(context.Background())
	// todo defer cancel()
	req := fake.Message{}
	players := mino.NewAddresses(participant.GetAddress())
	cancel()

	resps, _ := r.Call(ctx, req, players)
	_, ok := <-resps
	require.False(t, ok)
}

func Test_rpc_Stream(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})

	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	mustCreateRPC(participant, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses(participant.GetAddress())

	defer func() { // tear down
		cancel()
		require.NoError(t, initiator.close())
		require.NoError(t, participant.close())
	}()

	sender, receiver, err := r.Stream(ctx, players)
	require.NoError(t, err)
	require.NotNil(t, sender)
	require.NotNil(t, receiver)
}

func Test_rpc_Stream_NoPlayers(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses() // empty

	defer func() { // tear down
		cancel()
		require.NoError(t, initiator.close())
	}()

	_, _, err := r.Stream(ctx, players)
	require.ErrorContains(t, err, "no players")
}

func Test_rpc_Stream_WrongAddressType(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses(fake.Address{})

	defer func() { // tear down
		cancel()
		require.NoError(t, initiator.close())
	}()

	_, _, err := r.Stream(ctx, players)
	require.ErrorContains(t, err, "wrong address type")
}

func Test_rpc_Stream_SelfDial(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	r := mustCreateRPC(initiator, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses(initiator.GetAddress())

	defer func() { // tear down
		cancel()
		require.NoError(t, initiator.close())
	}()

	_, _, err := r.Stream(ctx, players)
	require.ErrorContains(t, err, "dial to self attempted")
}

// todo stream context cancelled
func Test_rpc_Stream_ContextCancelled(t *testing.T) {
	const addrInitiator = "/ip4/127.0.0.1/tcp/6001/ws"
	initiator := mustCreateMinows(addrInitiator, addrInitiator,
		mustCreateSecret())
	// todo defer tearDown()
	r := mustCreateRPC(initiator, "test", testHandler{})
	const addrParticipant = "/ip4/127.0.0.1/tcp/6002/ws"
	participant := mustCreateMinows(addrParticipant, addrParticipant,
		mustCreateSecret())
	// todo defer tearDown()
	mustCreateRPC(participant, "test", testHandler{})

	ctx, cancel := context.WithCancel(context.Background())
	players := mino.NewAddresses(participant.GetAddress())
	cancel()

	_, _, err := r.Stream(ctx, players)
	require.Error(t, err)
}

// todo testHandler: capture received requests to assert on sender & message
//
//	and echo back the message
type testHandler struct{}

func (e testHandler) Process(req mino.Request) (resp serde.Message, err error) {
	return req.Message, nil
}

func (e testHandler) Stream(out mino.Sender, in mino.Receiver) error {
	// TODO implement according to use in tests
	from, msg, err := in.Recv(context.Background())
	if err != nil {
		return err
	}
	err = <-out.Send(msg, from)
	if err != nil {
		return err
	}
	return nil
}

// todo take mino as arg & move to mod_test.go
func mustCreateRPCOld(address string, segments []string, name string,
	h mino.Handler) (*minows, mino.RPC) {
	// start listening
	m := mustCreateMinows(address, address, mustCreateSecret())
	for _, segment := range segments {
		m = m.WithSegment(segment).(*minows)
	}
	// register handler
	r, err := m.CreateRPC(name, h, fake.MessageFactory{})
	if err != nil {
		panic(err)
	}
	return m, r
}
