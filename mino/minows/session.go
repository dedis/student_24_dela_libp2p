package minows

import (
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
	"sync"
)

// session represents a stream session opened by RPC.Stream()
// between a Minows instance and other players of the RPC .
// - implements mino.Sender, mino.Receiver
type session struct {
	streams map[peer.ID]network.Stream // read-only after initialization
	rpc     rpc
	in      chan envelope // todo unbuffered, check no blocking
}

func (s session) Recv(ctx context.Context) (mino.Address, serde.Message, error) {
	select {
	case env, ok := <-s.in:
		if !ok { // session ended
			return nil, nil, xerrors.New("session ended")
		}
		return env.sender, env.message, env.err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}

func (s session) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	var wg sync.WaitGroup
	errs := make(chan error, len(addrs))
	for _, next := range addrs {
		addr, ok := next.(address)
		if !ok {
			errs <- xerrors.Errorf("invalid address type: %T", next)
			continue
		}
		stream, ok := s.streams[addr.identity]
		if !ok {
			errs <- xerrors.Errorf("address %v not a player", addr)
			continue
		}
		wg.Add(1)
		go func(stream network.Stream) {
			defer wg.Done()
			err := send(stream, msg, s.rpc.msgContext)
			if err != nil {
				errs <- err
			}
		}(stream)
	}
	// close error channel when all messages are sent
	go func() {
		wg.Wait()
		close(errs)
	}()
	return errs
}

// todo move to rpc.go
type envelope struct {
	sender  address
	message serde.Message
	err     error
}
