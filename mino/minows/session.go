package minows

import (
	"context"
	"encoding/json"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

var errSessionEnded = xerrors.New("session ended")

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
	myAddr   address
	rpc      rpc
	done     chan any
	encoders map[peer.ID]*json.Encoder
	mailbox  chan envelope
	loopback chan envelope
}

// Send multicasts a message to some players of this session concurrently.
// Send is asynchronous and returns immediately an error channel that
// closes when the message has either been sent or errored to each address.
func (s session) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	send := func(addr mino.Address) error {
		dest, ok := addr.(address)
		if !ok {
			return xerrors.Errorf("wrong address type: %T", addr)
		}
		if dest.Equal(s.myAddr) {
			if s.loopback == nil {
				return xerrors.Errorf("address %v not a player", dest)
			}
			go func() {
				s.loopback <- envelope{author: s.myAddr, msg: msg}
			}()
			return nil
		}
		encoder, ok := s.encoders[dest.identity]
		if !ok {
			return xerrors.Errorf("address %v not a player", dest)
		}
		return s.rpc.send(encoder, msg)
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
			select {
			case <-s.done:
				errs <- errSessionEnded
				return
			case env := <-result:
				if env.err != nil {
					errs <- xerrors.Errorf("could not send to %v: %v",
						env.author, env.err)
				}
			}
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
		return nil, nil, errSessionEnded
	case env := <-s.mailbox:
		return env.author, env.msg, env.err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}
