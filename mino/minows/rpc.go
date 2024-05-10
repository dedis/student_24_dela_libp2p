package minows

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rs/zerolog"
	"go.dedis.ch/dela"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const pathCall = "/call"
const pathStream = "/stream"

// Packet encapsulates a message sent over the network streams.
type Packet struct {
	Payload []byte
}

// RPC
// - implements mino.RPC
type rpc struct {
	logger zerolog.Logger

	myAddr  address
	uri     string
	handler mino.Handler
	mino    *minows // todo remove
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
					responses <- mino.NewResponseWithError(env.author, env.err)
				} else {
					responses <- mino.NewResponse(env.author, env.msg)
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
		r.mino.host.Peerstore().AddAddr(addr.identity, addr.location,
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
	reply, err := r.receive(in)
	if err != nil {
		return nil, xerrors.Errorf("could not receive reply: %v", err)
	}
	return reply, nil
}

func (r rpc) openStream(ctx context.Context, dest address,
	path string) (network.Stream, error) {
	pid := protocol.ID(r.uri + path)
	stream, err := r.mino.host.NewStream(ctx, dest.identity, pid)
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
	decoders, encoders := toCoders(streams)

	result := make(chan envelope)
	done := make(chan any)
	for from, decoder := range decoders {
		go func(from address, decoder *json.Decoder) {
			for {
				msg, err := r.receive(decoder)
				select {
				case <-done:
					return
				case result <- envelope{from, msg, err}:
				}
			}
		}(from, decoder)
	}

	mailbox := make(chan envelope)
	go func() {
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
	}()

	var loopback chan envelope
	if withLoopback {
		loopback = make(chan envelope)
		loop := &session{myAddr: r.myAddr, done: done,
			encoders: make(map[peer.ID]*json.Encoder),
			mailbox:  loopback, loopback: mailbox}
		go func() {
			err := r.handler.Stream(loop, loop)
			if err != nil {
				dela.Logger.Error().Err(err).Msg("could not handle stream")
			}
		}()
	}
	return &session{myAddr: r.myAddr, rpc: r, done: done,
		encoders: encoders, mailbox: mailbox, loopback: loopback}
}

func toCoders(streams []network.Stream) (map[address]*json.Decoder, map[peer.ID]*json.Encoder) {
	decoders := make(map[address]*json.Decoder, len(streams))
	encoders := make(map[peer.ID]*json.Encoder, len(streams))
	for _, stream := range streams {
		remote := getRemoteAddr(stream)
		decoders[remote] = json.NewDecoder(stream)
		encoders[remote.identity] = json.NewEncoder(stream)
	}
	return decoders, encoders
}

func getRemoteAddr(stream network.Stream) address {
	location := stream.Conn().RemoteMultiaddr()
	identity := stream.Conn().RemotePeer()
	remote := address{location, identity}
	return remote
}

func (r rpc) send(out *json.Encoder, msg serde.Message) error {
	payload, err := msg.Serialize(r.context)
	if err != nil {
		return xerrors.Errorf("could not serialize message: %v", err)
	}

	err = out.Encode(&Packet{payload})
	if errors.Is(err, network.ErrReset) {
		return err
	}
	if err != nil {
		return xerrors.Errorf("could not encode packet: %v", err)
	}
	return nil
}

func (r rpc) receive(in *json.Decoder) (serde.Message, error) {
	var packet Packet
	err := in.Decode(&packet)
	if errors.Is(err, network.ErrReset) {
		return nil, err
	}
	if err != nil {
		return nil, xerrors.Errorf("could not decode packet: %v", err)
	}

	msg, err := r.factory.Deserialize(r.context, packet.Payload)
	if err != nil {
		return nil, xerrors.Errorf(
			"could not deserialize message: %v",
			err)
	}
	return msg, nil
}

func (r rpc) createCallHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		in := json.NewDecoder(stream)
		req, err := r.receive(in)
		if err != nil {
			r.logger.Error().Err(err).Msg(
				"could not receive call")
			return
		}

		from := address{stream.Conn().RemoteMultiaddr(),
			stream.Conn().RemotePeer()}
		reply, err := h.Process(mino.Request{Address: from, Message: req})
		if err != nil {
			r.logger.Error().Err(err).Msg("could not process call")
			return
		}

		out := json.NewEncoder(stream)
		err = r.send(out, reply)
		if err != nil {
			r.logger.Error().Err(err).Msg("could not reply to call")
			return
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
				dela.Logger.Error().Err(err).Msg("could not handle stream")
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
