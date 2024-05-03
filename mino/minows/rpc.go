package minows

import (
	"context"
	"errors"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/rs/zerolog"
	"go.dedis.ch/dela"
	"io"
	"sync"

	"github.com/libp2p/go-libp2p/core/protocol"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const MaxMessageSize = 1e9

const PostfixCall = "/call"
const PostfixStream = "/stream"

// RPC
// - implements mino.RPC
type rpc struct {
	logger zerolog.Logger

	uri     string
	mino    *minows
	factory serde.Factory
	context serde.Context
}

type result struct {
	remote address
	stream network.Stream
	err    error
}

// Call is non-blocking and returns before all communications are established.
// Returns an error if no players or any player address is of the wrong type.
// Otherwise, returns a response channel 1) filled with replies or errors args
// the network from each player 2) closed after every player has
// replied or errored, or the context is done.
func (r rpc) Call(ctx context.Context, req serde.Message,
	players mino.Players) (<-chan mino.Response, error) {
	if players == nil || players.Len() == 0 {
		return nil, xerrors.New("no players to call")
	}

	addrs, err := toAddresses(players)
	if err != nil {
		return nil, err
	}

	r.addPeers(addrs)
	results := r.openStreams(ctx, protocol.ID(r.uri+PostfixCall), addrs)

	responses := make(chan mino.Response, len(addrs))
	var wg sync.WaitGroup
	for range addrs {
		wg.Add(1)

		go func() {
			defer wg.Done()

			select {
			case res := <-results: // fan-out
				if res.err != nil {
					responses <- mino.NewResponseWithError(res.remote, res.err)
					return
				}
				reply, err := r.unicast(res.stream, req, r.factory, r.context)
				if err != nil {
					responses <- mino.NewResponseWithError(res.remote, err)
					return
				}
				responses <- mino.NewResponse(res.remote, reply)
			case <-ctx.Done(): // let goroutine exit if context is done
				// before all streams are established
			}
		}()
	}

	go func() {
		wg.Wait()
		close(responses)
	}()

	return responses, nil
}

// Stream is blocking and returns only after all communications are
// established.
// Returns an error if no players, any player address is of the wrong type,
// or communication to any player fails to establish.
// Otherwise, returns a stream session that can be used to send and receive
// messages.
// Note:
// - 'ctx' defines when the stream session ends,
// so should always be canceled at some point (e.g. completed task or errored).
// - When it's done, all the connections are shut down and resources freed.
func (r rpc) Stream(ctx context.Context, players mino.Players) (mino.Sender, mino.Receiver, error) {
	if players == nil || players.Len() == 0 {
		return nil, nil, xerrors.New("no players to stream")
	}

	addrs, err := toAddresses(players)
	if err != nil {
		return nil, nil, err
	}

	r.addPeers(addrs)
	results := r.openStreams(ctx, protocol.ID(r.uri+PostfixStream), addrs)
	streams, err := collectStreams(results)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not establish streams: %v", err)
	}

	sess, err := r.createSession(streams)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not start session: %v", err)
	}

	return sess, sess, nil
}

func collectStreams(results <-chan result) (map[peer.ID]network.Stream, error) {
	streams := make(map[peer.ID]network.Stream)
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		streams[res.remote.identity] = res.stream
	}
	return streams, nil
}

func (r rpc) createCallHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		sender, msg, err := receive(stream, r.factory, r.context)
		if err != nil {
			r.logger.Error().Msgf("could not receive call request: %v", err)
			return
		}

		resp, err := h.Process(mino.Request{Address: sender, Message: msg})
		if err != nil {
			r.logger.Error().Msgf("could not process call request: %v", err)
			return
		}

		err = send(stream, resp, r.context)
		if err != nil {
			r.logger.Error().Msgf("could not send call response: %v", err)
			return
		}
	}
}

func (r rpc) createStreamHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		sess, err := r.createSession(
			map[peer.ID]network.Stream{stream.Conn().RemotePeer(): stream})
		if err != nil {
			dela.Logger.Error().Msgf("could not start stream session: %v", err)
			return
		}

		go func() {
			err = h.Stream(sess, sess)
			if err != nil {
				dela.Logger.Err(err).Msg("could not handle stream")
				return
			}
		}()
	}
}

func (r rpc) addPeers(addrs []address) {
	for _, addr := range addrs {
		r.mino.host.Peerstore().AddAddr(addr.identity, addr.location,
			peerstore.PermanentAddrTTL)
	}
}

// openStreams opens streams to `addrs` concurrently.
// It cancels any pending streams,
// resets & frees any established streams,
// and closes the output channel when `ctx` is done.
func (r rpc) openStreams(ctx context.Context,
	p protocol.ID, addrs []address) chan result {
	r.logger.Debug().Msgf("opening streams to %v...", addrs)

	var wg sync.WaitGroup
	results := make(chan result, len(addrs))
	for _, addr := range addrs {
		wg.Add(1)

		go func(addr address) {
			defer wg.Done()

			stream, err := r.mino.host.NewStream(ctx, addr.identity, p)
			if err != nil {
				r.logger.Debug().Err(err).Msgf("could not open stream to %v", addr)
				results <- result{
					remote: addr,
					err:    xerrors.Errorf("could not open stream: %v", err),
				}
				return
			}

			r.logger.Debug().Msgf("opened stream to %v", addr)
			results <- result{remote: addr, stream: stream}

			go func() { // reset established stream
				<-ctx.Done()
				err := stream.Reset()
				if err != nil {
					r.logger.Error().Err(err).Msg("could not reset stream: %v")
					return
				}
				r.logger.Debug().Msgf("reset stream to %v", addr)
			}()
		}(addr)
	}

	go func() {
		wg.Wait()
		close(results)
		r.logger.Debug().Msg("finished opening streams")
	}()

	return results
}

func (r rpc) unicast(stream network.Stream, req serde.Message,
	f serde.Factory, c serde.Context) (serde.Message, error) {
	err := send(stream, req, r.context)
	if err != nil {
		return nil, err
	}
	_, reply, err := receive(stream, f, c)
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// createSession
// session ends automatically on both initiator & player side by closing
// the incoming message channel when the initiator is done and cancels the
// stream context which resets all streams
func (r rpc) createSession(streams map[peer.ID]network.Stream) (*session, error) {
	in := make(chan envelope) // unbuffered
	for _, stream := range streams {
		go func(stream network.Stream) {
			for {
				sender, msg, err := receive(stream, r.factory, r.context)
				in <- envelope{sender, msg, err} // fan-in
			}
		}(stream)
	}

	return &session{
		streams: streams,
		rpc:     r,
		in:      in,
	}, nil
}

func toAddresses(players mino.Players) ([]address, error) {
	var addrs []address
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

func send(stream network.Stream, msg serde.Message, c serde.Context) error {
	data, err := msg.Serialize(c)
	if err != nil {
		return xerrors.Errorf("could not serialize message: %v", err)
	}

	_, err = stream.Write(data)
	if errors.Is(err, network.ErrReset) || err == io.EOF {
		return err
	}
	if err != nil {
		return xerrors.Errorf("could not write to stream: %v", err)
	}
	return nil
}

func receive(stream network.Stream,
	f serde.Factory, c serde.Context) (address, serde.Message, error) {
	sender, err := newAddress(
		stream.Conn().RemoteMultiaddr(),
		stream.Conn().RemotePeer())
	if err != nil {
		return address{}, nil, xerrors.Errorf(
			"unexpected: could not create sender address: %v",
			err)
	}

	buffer := make([]byte, MaxMessageSize)
	n, err := stream.Read(buffer)
	if errors.Is(err, network.ErrReset) || err == io.EOF {
		return sender, nil, err
	}
	if err != nil {
		return sender, nil, xerrors.Errorf(
			"could not read from stream: %v",
			err)
	}

	msg, err := f.Deserialize(c, buffer[:n])
	if err != nil {
		return sender, nil, xerrors.Errorf(
			"could not deserialize message: %v",
			err)
	}
	return sender, msg, nil
}
