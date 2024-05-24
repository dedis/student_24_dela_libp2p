package minows

import (
	"context"
	"encoding/gob"
	"errors"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/rs/xid"
	"github.com/rs/zerolog"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const pathCall = "/call"
const pathStream = "/stream"

// Packet encapsulates a message sent over the network streams.
type Packet struct {
	Source  []byte
	Payload []byte
}

// RPC
// - implements mino.RPC
type rpc struct {
	logger zerolog.Logger

	myAddr  address
	uri     string
	host    host.Host
	handler mino.Handler
	mino    *minows
	factory serde.Factory
	context serde.Context
}

// Call sends a request to all players concurrently and fills the response
// channel with replies or errors from the network.
// Call is asynchronous and returns immediately either an error or a response
// channel that is closed when each player has replied,
// or the context is done.
func (r rpc) Call(ctx context.Context, req serde.Message,
	players mino.Players) (<-chan mino.Response, error) {
	if players == nil || players.Len() == 0 {
		return nil, xerrors.New("no players")
	}

	addrs, err := toAddresses(players)
	if err != nil {
		return nil, err
	}

	r.addPeers(addrs)

	result := make(chan envelope, len(addrs))
	for _, addr := range addrs {
		if r.myAddr.Equal(addr) {
			request := mino.Request{Address: r.myAddr, Message: req}
			reply, err := r.handler.Process(request)
			result <- envelope{r.myAddr, reply, err}
		} else {
			go func(addr address) {
				reply, err := r.unicast(ctx, addr, req)
				result <- envelope{addr, reply, err}
			}(addr)
		}
	}

	responses := make(chan mino.Response, len(addrs))
	go func() {
		defer close(responses)
		for i := 0; i < len(addrs); i++ {
			select {
			case <-ctx.Done():
				return
			case env := <-result:
				if env.err != nil {
					responses <- mino.NewResponseWithError(env.from, env.err)
				} else {
					responses <- mino.NewResponse(env.from, env.msg)
				}
			}
		}
	}()
	return responses, nil
}

// Stream starts a persistent bidirectional stream session with the players.
// Stream is synchronous and returns after communications to all players
// are established.
// When the context is done, the stream session ends and all streams are reset.
func (r rpc) Stream(ctx context.Context, players mino.Players) (mino.Sender, mino.Receiver, error) {
	if players == nil || players.Len() == 0 {
		return nil, nil, xerrors.New("no players")
	}

	addrs, err := toAddresses(players)
	if err != nil {
		return nil, nil, err
	}

	r.addPeers(addrs)

	result := make(chan network.Stream, len(addrs))
	errs := make(chan error, len(addrs))
	selfDial := false
	remote := 0
	for _, addr := range addrs {
		if addr.Equal(r.myAddr) {
			selfDial = true
			continue
		}
		go func(addr address) {
			stream, err := r.openStream(ctx, addr, pathStream)
			if err != nil {
				errs <- err
				return
			}
			result <- stream
		}(addr)
		remote++
	}

	streams := make([]network.Stream, 0, remote)
	for i := 0; i < remote; i++ {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case err = <-errs:
			return nil, nil, err
		case stream := <-result:
			streams = append(streams, stream)
		}
	}

	sess := r.createSession(ctx, streams, selfDial)
	return sess, sess, nil
}

func (r rpc) addPeers(addrs []address) {
	for _, addr := range addrs {
		r.host.Peerstore().AddAddr(addr.identity, addr.location,
			peerstore.PermanentAddrTTL)
	}
}

func (r rpc) unicast(ctx context.Context, dest address, req serde.Message) (
	serde.Message, error) {
	stream, err := r.openStream(ctx, dest, pathCall)
	if err != nil {
		return nil, xerrors.Errorf("could not open stream: %v", err)
	}

	dec := gob.NewEncoder(stream)
	err = r.send(dec, req)
	if err != nil {
		return nil, xerrors.Errorf("could not send request: %v", err)
	}

	enc := gob.NewDecoder(stream)
	_, reply, err := r.receive(enc)
	if err != nil {
		return nil, xerrors.Errorf("could not receive reply: %v", err)
	}
	return reply, nil
}

func (r rpc) openStream(ctx context.Context, dest address,
	path string) (network.Stream, error) {
	pid := protocol.ID(r.uri + path)
	stream, err := r.host.NewStream(ctx, dest.identity, pid)
	if err != nil {
		return nil, xerrors.Errorf("could not open stream: %v", err)
	}

	go func() {
		<-ctx.Done()
		err := stream.Reset()
		if err != nil {
			r.logger.Error().Err(err).Msg("could not reset stream")
		}
	}()

	return stream, nil
}

func (r rpc) createSession(ctx context.Context,
	streams []network.Stream, loopback bool) *session {
	done := make(chan any)
	buffer := make(chan envelope, loopbackBufferSize)
	in := make(chan envelope)
	if loopback {
		lb := r.setupLoopback(done, buffer)
		go listenLoopback(done, lb.buffer, in)
	}

	result := make(chan envelope)
	encoders := make(map[peer.ID]*gob.Encoder)
	for _, stream := range streams {
		id := stream.Conn().RemotePeer()
		encoder := gob.NewEncoder(stream)
		encoders[id] = encoder
		decoder := gob.NewDecoder(stream)
		go func() {
			for {
				from, msg, err := r.receive(decoder)
				author := address{location: from, identity: id}
				select {
				case <-done:
					return
				case result <- envelope{author, msg, err}:
				}
			}
		}()
	}

	go func() {
		for env := range result {
			// Cancelling context resets stream and ends session for
			// participants
			if xerrors.Is(env.err, network.ErrReset) {
				close(done)
				return
			}
			select {
			// Initiator ended session by canceling context
			case <-ctx.Done():
				close(done)
				return
			case in <- env:
			}
		}
	}()

	return &session{
		logger: r.logger.With().Stringer("session", xid.New()).Logger(),
		myAddr: r.myAddr,
		rpc:    r,
		done:   done,
		outs:   encoders,
		in:     in,
		buffer: buffer,
	}
}

func (r rpc) setupLoopback(done chan any, buffer chan envelope) *session {
	loopback := &session{
		logger: r.logger.With().Stringer("session", xid.New()).Logger(),
		myAddr: r.myAddr,
		done:   done,
		outs:   make(map[peer.ID]*gob.Encoder),
		in:     make(chan envelope),
		buffer: make(chan envelope, loopbackBufferSize),
	}
	go listenLoopback(done, buffer, loopback.in)
	go func() {
		err := r.handler.Stream(loopback, loopback)
		if err != nil {
			r.logger.Error().Err(err).Msg("could not handle stream")
		}
	}()
	return loopback
}

func listenLoopback(done chan any, buffer chan envelope, in chan envelope) {
	for {
		select {
		case <-done:
			return
		case env := <-buffer:
			select {
			case <-done:
				return
			case in <- env:
			}
		}
	}
}

func (r rpc) send(enc *gob.Encoder, msg serde.Message) error {
	from := r.myAddr.location.Bytes()

	var payload []byte
	if msg != nil {
		bytes, err := msg.Serialize(r.context)
		if err != nil {
			return xerrors.Errorf("could not serialize message: %v", err)
		}
		payload = bytes
	}

	err := enc.Encode(&Packet{Source: from, Payload: payload})
	if errors.Is(err, network.ErrReset) {
		return err
	}
	if err != nil {
		return xerrors.Errorf("could not encode packet: %v", err)
	}
	return nil
}

func (r rpc) receive(dec *gob.Decoder) (ma.Multiaddr, serde.Message, error) {
	var packet Packet
	err := dec.Decode(&packet)
	if errors.Is(err, network.ErrReset) {
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, xerrors.Errorf("could not decode packet: %v", err)
	}

	from, err := ma.NewMultiaddrBytes(packet.Source)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not unmarshal address: %v",
			packet.Source)
	}

	if packet.Payload == nil {
		return from, nil, nil
	}
	msg, err := r.factory.Deserialize(r.context, packet.Payload)
	if err != nil {
		return from, nil, xerrors.Errorf(
			"could not deserialize message: %v",
			err)
	}
	return from, msg, nil
}

func (r rpc) createCallHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		handle := func() error {
			dec := gob.NewDecoder(stream)
			from, req, err := r.receive(dec)
			if err != nil {
				return xerrors.Errorf("could not receive: %v", err)
			}

			id := stream.Conn().RemotePeer()
			author := address{location: from, identity: id}
			reply, err := h.Process(mino.Request{Address: author, Message: req})
			if err != nil {
				return xerrors.Errorf("could not process: %v", err)
			}

			enc := gob.NewEncoder(stream)
			err = r.send(enc, reply)
			if err != nil {
				return xerrors.Errorf("could not reply: %v", err)
			}
			return nil
		}
		err := handle()
		if err != nil {
			r.logger.Error().Err(err).Msg("could not handle call")
		}
	}
}

func (r rpc) createStreamHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		sess := r.createSession(context.Background(),
			[]network.Stream{stream}, false)

		go func() {
			err := h.Stream(sess, sess)
			if err != nil {
				r.logger.Error().Err(err).Msg("could not handle stream")
			}
		}()
	}
}

func toAddresses(players mino.Players) ([]address, error) {
	addrs := make([]address, 0, players.Len())
	iter := players.AddressIterator()
	for iter.HasNext() {
		player := iter.GetNext()
		addr, ok := player.(address)
		if !ok {
			return nil, xerrors.Errorf("wrong address type: %T", player)
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
