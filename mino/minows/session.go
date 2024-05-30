package minows

import (
	"context"
	"encoding/gob"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
	"io"
	"sync"
)

var ErrWrongAddressType = xerrors.New("wrong address type")
var ErrNotPlayer = xerrors.New("not player")

// MaxUnreadAllowed Maximum number of unread messages allowed
// in orchestrator's incoming message buffer before pausing relaying
const MaxUnreadAllowed = 1e3

type Forward struct {
	Packet
	Destination []byte
}

type orchestrator struct {
	logger zerolog.Logger

	myAddr orchestratorAddr
	rpc    rpc
	// Connects to the participants
	outs map[peer.ID]*gob.Encoder
	in   chan Packet
}

func (o orchestrator) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	send := func(addr mino.Address) error {
		var unwrapped address
		switch a := addr.(type) {
		case address:
			unwrapped = a
		case orchestratorAddr:
			unwrapped = a.address
		default:
			return xerrors.Errorf("%v: %T", ErrWrongAddressType, addr)
		}

		encoder, ok := o.outs[unwrapped.identity]
		if !ok {
			return xerrors.Errorf("%v: %v", ErrNotPlayer, addr)
		}
		src, err := o.myAddr.MarshalText()
		if err != nil {
			return xerrors.Errorf("could not marshal address: %v", err)
		}
		payload, err := msg.Serialize(o.rpc.context)
		if err != nil {
			return xerrors.Errorf("could not serialize message: %v", err)
		}

		err = encoder.Encode(&Packet{Source: src, Payload: payload})
		if err != nil {
			return xerrors.Errorf("could not encode packet: %v", err)
		}
		return nil
	}

	errs := doSend(addrs, send, msg, o.logger)
	return errs
}

func (o orchestrator) Recv(ctx context.Context) (mino.Address, serde.Message, error) {
	return doReceive(ctx, o.in, o.rpc.mino.GetAddressFactory(), o.rpc.factory,
		o.rpc.context, o.logger)
}

type participant struct {
	logger zerolog.Logger

	myAddr address
	rpc    rpc
	// Connects to the orchestrator
	out *gob.Encoder
	in  chan Packet
}

func (p participant) Send(msg serde.Message, addrs ...mino.Address) <-chan error {
	send := func(addr mino.Address) error {
		switch addr.(type) {
		case address:
		case orchestratorAddr:
		default:
			return xerrors.Errorf("%v: %T", ErrWrongAddressType, addr)
		}

		src, err := p.myAddr.MarshalText()
		if err != nil {
			return xerrors.Errorf("could not marshal address: %v", err)
		}
		payload, err := msg.Serialize(p.rpc.context)
		if err != nil {
			return xerrors.Errorf("could not serialize message: %v", err)
		}
		dest, err := addr.MarshalText()
		if err != nil {
			return xerrors.Errorf("could not marshal address: %v", err)
		}

		// Send to orchestrator to relay to the destination participant
		forward := Forward{
			Packet:      Packet{Source: src, Payload: payload},
			Destination: dest,
		}
		err = p.out.Encode(&forward)
		if err != nil {
			return xerrors.Errorf("could not encode packet: %v", err)
		}
		return nil
	}

	errs := doSend(addrs, send, msg, p.logger)
	return errs
}

func (p participant) Recv(ctx context.Context) (mino.Address, serde.Message, error) {
	return doReceive(ctx, p.in, p.rpc.mino.GetAddressFactory(),
		p.rpc.factory, p.rpc.context, p.logger)
}

func doSend(addrs []mino.Address, send func(addr mino.Address) error,
	msg serde.Message, logger zerolog.Logger) chan error {
	errs := make(chan error, len(addrs))
	var wg sync.WaitGroup
	wg.Add(len(addrs))
	for _, addr := range addrs {
		go func(addr mino.Address) {
			defer wg.Done()
			err := send(addr)
			if err != nil {
				errs <- xerrors.Errorf("could not send to %v: %v", addr, err)
				logger.Error().Err(err).Msgf("could not send %T to %v", msg, addr)
				return
			}
			logger.Debug().Msgf("sent %T to %v", msg, addr)
		}(addr)
	}

	go func() {
		wg.Wait()
		close(errs)
	}()
	return errs
}

func doReceive(ctx context.Context, in chan Packet,
	af mino.AddressFactory, f serde.Factory, c serde.Context,
	logger zerolog.Logger) (mino.Address, serde.Message, error) {
	unpack := func(packet Packet) (mino.Address, serde.Message, error) {
		src := af.FromText(packet.Source)
		if src == nil {
			return nil, nil, xerrors.New("could not unmarshal address")
		}
		msg, err := f.Deserialize(c, packet.Payload)
		if err != nil {
			return src, nil, xerrors.Errorf("could not deserialize message: %v", err)
		}
		return src, msg, nil
	}

	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case packet, open := <-in:
		if !open {
			return nil, nil, io.EOF
		}
		origin, msg, err := unpack(packet)
		if err != nil {
			logger.Error().Err(err).Msg("could not receive")
			return nil, nil, xerrors.Errorf("could not receive from %v: %v",
				origin, err)
		}
		logger.Debug().Msgf("received %T from %v", msg, origin)
		return origin, msg, nil
	}
}
