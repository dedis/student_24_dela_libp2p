package minows

import (
	"context"
	"encoding/gob"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
	"io"
)

const loopbackBufferSize = 16

type envelope struct {
	addr mino.Address // todo rename from
	msg  serde.Message
	err  error
}

// session represents a stream session started by rpc.Stream()
// between a minows instance and other players of the RPC.
// A session ends for all participants when the initiator is done and cancels
// the stream context.
// - implements mino.Sender, mino.Receiver
type session struct {
	logger zerolog.Logger

	myAddr   address
	rpc      rpc
	done     chan any
	encoders map[peer.ID]*gob.Encoder // todo rename outs
	loopback chan envelope            // todo rename buffer??
	mailbox  chan envelope            // todo rename in
}

// Send multicasts a message to some players of this session concurrently.
// Send is asynchronous and returns immediately an error channel that
// closes when the message has either been sent or errored to each address.
func (s session) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	send := func(addr mino.Address) error {
		to, ok := addr.(address)
		if !ok {
			return xerrors.Errorf("wrong address type: %T", addr)
		}
		if to.Equal(s.myAddr) {
			// todo refactor with same error below to one
			if s.loopback == nil {
				return xerrors.Errorf("address %v not a player", to)
			}
			select {
			case <-s.done:
				return network.ErrReset
			default:
				s.loopback <- envelope{addr: s.myAddr, msg: msg}
			}
			return nil
		}
		encoder, ok := s.encoders[to.identity]
		if !ok {
			return xerrors.Errorf("address %v not a player", to)
		}
		return s.rpc.send(encoder, msg)
	}

	result := make(chan envelope, len(addrs))
	for _, addr := range addrs {
		go func(dest mino.Address) {
			err := send(dest)
			result <- envelope{addr: dest, err: err}
		}(addr)
	}

	errs := make(chan error, len(addrs))
	go func() {
		defer close(errs)
		for i := 0; i < len(addrs); i++ {
			env := <-result
			if xerrors.Is(env.err, network.ErrReset) {
				errs <- io.ErrClosedPipe
				return
			}
			if env.err != nil {
				errs <- xerrors.Errorf("could not send to %v: %v",
					env.addr, env.err)
				continue
			}
			s.logger.Trace().Stringer("to", env.addr).
				Msgf("sent %v", msg)
		}
	}()
	return errs
}

// Recv receives a message from the players of this session.
// Recv is synchronous and returns when a message is received or the
// context is done.
func (s session) Recv(ctx context.Context) (mino.Address, serde.Message, error) {
	select {
	case <-s.done:
		return nil, nil, io.EOF
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case env := <-s.mailbox:
		s.logger.Trace().Stringer("from", env.addr).
			Msgf("received %v", env.msg)
		return env.addr, env.msg, env.err
	}
}
