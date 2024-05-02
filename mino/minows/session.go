package minows

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
	"io"
	"sync"
)

// session represents a stream session opened by RPC.Stream()
// between a Minows instance and other players of the RPC .
// - implements mino.Sender, mino.Receiver
type session struct {
	streams map[peer.ID]network.Stream // read-only after initialization
	rpc     rpc
	in      chan envelope
}

type envelope struct {
	sender  address
	message serde.Message
	err     error
}

func (s session) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	var wg sync.WaitGroup
	errs := make(chan error, len(addrs))
	// send message to all addresses concurrently
	// some may fail while some succeed
	for _, next := range addrs {
		addr, ok := next.(address)
		if !ok {
			errs <- xerrors.Errorf("wrong address type: %T", next)
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
			err := send(stream, msg, s.rpc.context)
			// all streams are reset when session ends
			if errors.Is(err, network.ErrReset) || errors.Is(err,
				io.ErrClosedPipe) {
				errs <- xerrors.Errorf("session ended: %w", err)
			} else if err != nil {
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

func (s session) Recv(ctx context.Context) (mino.Address, serde.Message, error) {
	select {
	case env := <-s.in:
		// all streams are reset when session ends
		if errors.Is(env.err, network.ErrReset) || errors.Is(env.err, io.EOF) {
			return nil, nil, xerrors.Errorf("session ended: %w", env.err)
		}
		return env.sender, env.message, env.err
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}
}
