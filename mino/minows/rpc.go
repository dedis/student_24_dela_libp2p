package minows

import (
	"context"
	"encoding/json"
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
	ctx, cancel := context.WithCancel(ctx)
	for _, dest := range addrs {
		if r.myAddr.Equal(dest) {
			request := mino.Request{Address: r.myAddr, Message: req}
			reply, err := r.handler.Process(request)
			result <- envelope{r.myAddr, reply, err}
		} else {
			go func(dest address) {
				reply, err := r.unicast(ctx, dest, req)
				result <- envelope{dest, reply, err}
			}(dest)
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
					responses <- mino.NewResponseWithError(env.addr, env.err)
				} else {
					responses <- mino.NewResponse(env.addr, env.msg)
				}
			}
		}
		cancel()
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

	out := json.NewEncoder(stream)
	err = r.send(out, req)
	if err != nil {
		return nil, xerrors.Errorf("could not send request: %v", err)
	}

	in := json.NewDecoder(stream)
	_, reply, err := r.receive(in)
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
	streams []network.Stream, withLoopback bool) *session {
	result := make(chan envelope)
	done := make(chan any)
	listen := func(decoder *json.Decoder, id peer.ID) {
		for {
			from, msg, err := r.receive(decoder)
			author := address{identity: id, location: from}
			select {
			case <-done:
				return
			case result <- envelope{author, msg, err}:
			}
		}
	}

	encoders := make(map[peer.ID]*json.Encoder)
	for _, stream := range streams {
		id := stream.Conn().RemotePeer()
		encoder := json.NewEncoder(stream)
		encoders[id] = encoder
		decoder := json.NewDecoder(stream)
		go listen(decoder, id)
	}

	mailbox := make(chan envelope)
	deliver := func() {
		for {
			select {
			// Initiator ended session by canceling context
			case <-ctx.Done():
				close(done)
				return
			case env := <-result:
				// Cancelling context resets stream and ends
				// session for players too
				if xerrors.Is(env.err, network.ErrReset) {
					close(done)
					return
				}
				mailbox <- env
			}
		}
	}
	go deliver()

	var loopback chan envelope
	if withLoopback {
		loopback = make(chan envelope)
		loop := &session{
			logger: r.logger.With().Stringer("loopback", xid.New()).Logger(),
			myAddr: r.myAddr, done: done,
			encoders: make(map[peer.ID]*json.Encoder),
			mailbox:  loopback, loopback: mailbox,
		}
		go func() {
			err := r.handler.Stream(loop, loop)
			if err != nil {
				r.logger.Error().Err(err).Msg("could not handle stream")
			}
		}()
	}
	return &session{
		logger: r.logger.With().Stringer("session", xid.New()).Logger(),
		myAddr: r.myAddr, rpc: r, done: done,
		encoders: encoders, mailbox: mailbox, loopback: loopback,
	}
}

func (r rpc) send(out *json.Encoder, msg serde.Message) error {
	from := r.myAddr.location.Bytes()

	var payload []byte
	if msg != nil {
		bytes, err := msg.Serialize(r.context)
		if err != nil {
			return xerrors.Errorf("could not serialize message: %v", err)
		}
		payload = bytes
	}

	err := out.Encode(&Packet{Source: from, Payload: payload})
	if errors.Is(err, network.ErrReset) {
		return err
	}
	if err != nil {
		return xerrors.Errorf("could not encode packet: %v", err)
	}
	return nil
}

func (r rpc) receive(in *json.Decoder) (ma.Multiaddr, serde.Message, error) {
	var packet Packet
	err := in.Decode(&packet)
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
			in := json.NewDecoder(stream)
			from, req, err := r.receive(in)
			if err != nil {
				return xerrors.Errorf("could not receive: %v", err)
			}

			id := stream.Conn().RemotePeer()
			author := address{identity: id, location: from}
			reply, err := h.Process(mino.Request{Address: author, Message: req})
			if err != nil {
				return xerrors.Errorf("could not process: %v", err)
			}

			out := json.NewEncoder(stream)
			err = r.send(out, reply)
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
		next := iter.GetNext()
		addr, ok := next.(address)
		if !ok {
			return nil, xerrors.Errorf("wrong address type: %T", next)
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
