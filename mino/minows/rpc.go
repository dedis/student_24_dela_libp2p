package minows

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const MaxMessageSize = 1e9 // TODO verify

// RPC implements mino.RPC
// TODO unit tests
type RPC struct {
	uri  protocol.ID
	host host.Host
	// handler mino.Handler
	factory serde.Factory // TODO assign in CreateRPC()
	context serde.Context // TODO assign somewhere
}

func (r RPC) Call(
	ctx context.Context,
	req serde.Message,
	players mino.Players,
) (<-chan mino.Response, error) {
	// TODO assumption: peer store already filled with 'players' Peer IDs & multi-addresses

	// dial participants iteratively in parallel
	var wg sync.WaitGroup
	iter := players.AddressIterator()

	responses := make(chan mino.Response, players.Len())

	for iter.HasNext() {
		next := iter.GetNext()
		target, ok := next.(Address)
		if !ok {
			return nil, xerrors.Errorf("invalid address type: %T", next)
		}
		wg.Add(1)
		go func(target Address) {
			defer wg.Done()

			reply, err := r.call(ctx, req, target)
			if err != nil {
				responses <- mino.NewResponseWithError(target, err)
				return
			}
			responses <- mino.NewResponse(target, reply)
		}(target)
	}
	// wait for responses & close response channel
	go func() {
		wg.Wait()
		close(responses)
	}()
	return responses, nil
}

func (r RPC) Stream(ctx context.Context, players mino.Players) (mino.Sender, mino.Receiver, error) {
	// TODO implement me
	panic("implement me")
}

func (r RPC) call(ctx context.Context, req serde.Message, target Address) (serde.Message, error) {
	// open connection stream
	stream, err := r.host.NewStream(ctx, target.PeerID(), r.uri)
	if err != nil {
		return nil, xerrors.Errorf("could not open stream: %v", err)
	}
	defer stream.Reset() // must discard stream
	// send request & wait for response
	// TODO verify
	out, err := req.Serialize(r.context)
	if err != nil {
		return nil, xerrors.Errorf("could not serialize request: %v", err)
	}
	_, err = stream.Write(out) // blocking
	if err != nil {
		return nil, xerrors.Errorf("could not send request: %v", err)
	}
	// TODO verify
	var in []byte
	_, err = stream.Read(in)
	if err != nil {
		return nil, xerrors.Errorf("could not receive reply: %v", err)
	}
	reply, err := r.factory.Deserialize(r.context, in)
	if err != nil {
		return nil, xerrors.Errorf("could not deserialize reply: %v", err)
	}
	return reply, nil
}
