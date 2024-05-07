package minows

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

type envelope struct {
	author mino.Address
	msg    serde.Message
	err    error
}

// session represents a stream session started by rpc.Stream()
// between a minows instance and other players of the RPC.
// A session ends for all participants when the initiator is done and cancels
// the stream context.
// - implements mino.Sender, mino.Receiver
type session struct {
	rpc  rpc
	in   chan mino.Response
	outs map[peer.ID]*json.Encoder
}

// Send sends a message to all addresses concurrently.
// Send is asynchronous and returns immediately an error channel that
// closes when the message has either been sent or errored to each address.
func (s session) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	send := func(addr mino.Address) error {
		dest, ok := addr.(address)
		if !ok {
			return xerrors.Errorf("wrong address type: %T", addr)
		}
		out, ok := s.outs[dest.identity]
		if !ok {
			return xerrors.Errorf("address %v not a player", dest)
		}
		return s.rpc.send(out, msg)
	}

	result := make(chan envelope, len(addrs))
	for _, addr := range addrs {
		go func(addr mino.Address) {
			err := send(addr)
			result <- envelope{author: addr, err: err}
		}(addr)
	}

	errs := make(chan error, len(addrs))
	go func() {
		defer close(errs)

		for i := 0; i < len(addrs); i++ {
			env := <-result
			if errors.Is(env.err, network.ErrReset) {
				errs <- xerrors.Errorf("session ended: %v", env.err)
				return
			}
			if env.err != nil {
				errs <- xerrors.Errorf("could not send to %v: %v",
					env.author, env.err)
			}
		}
	}()

	return errs
}

func (s session) Recv(ctx context.Context) (mino.Address, serde.Message, error) {
	select {
	case res, ok := <-s.in:
		if !ok {
			return nil, nil, xerrors.Errorf("session ended: %v",
				network.ErrReset)
		}
		msg, err := res.GetMessageOrError()
		return res.GetFrom(), msg, err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}
