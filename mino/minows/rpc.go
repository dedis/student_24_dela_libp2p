package minows

import (
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/rs/zerolog"
	"sync"

	"github.com/libp2p/go-libp2p/core/protocol"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const MaxMessageSize = 1e9 // TODO verify
const PostfixCall = "/call"
const PostfixStream = "/stream"

// RPC implements mino.RPC
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
func (r rpc) Call(
	ctx context.Context,
	req serde.Message,
	players mino.Players,
) (<-chan mino.Response, error) {
	if players == nil || players.Len() == 0 {
		return nil, xerrors.New("no players to call")
	}
	// quit unless all player addresses are valid
	addrs, err := toAddresses(players)
	if err != nil {
		return nil, err
	}
	// fill peer store
	for _, addr := range addrs {
		r.mino.host.Peerstore().AddAddr(addr.identity, addr.location,
			peerstore.PermanentAddrTTL)
	}
	// establish streams to all players concurrently
	results := r.openStreams(ctx, protocol.ID(r.uri+PostfixCall), addrs)
	// unicast to each player concurrently as streams establish
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
				err := send(res.stream, req, r.context)
				if err != nil {
					responses <- mino.NewResponseWithError(res.remote, err)
					return
				}
				sender, msg, err := receive(res.stream, r.factory, r.context)
				if err != nil {
					responses <- mino.NewResponseWithError(sender, err)
					return
				}
				responses <- mino.NewResponse(sender, msg)
			case <-ctx.Done(): // let goroutine exit if context is done
				// before all streams are established
				// todo put "context cancelled" error in responses?
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
	// quit unless all player addresses are valid
	addrs, err := toAddresses(players)
	if err != nil {
		return nil, nil, err
	}
	// fill peer store
	for _, addr := range addrs {
		r.mino.host.Peerstore().AddAddr(addr.identity, addr.location,
			peerstore.PermanentAddrTTL)
	}
	// establish streams to all players concurrently
	results := r.openStreams(ctx, protocol.ID(r.uri+PostfixStream), addrs)
	// wait till all streams are established successfully or quit
	streams := make(map[peer.ID]network.Stream)
	for res := range results {
		if res.err != nil {
			return nil, nil, res.err
		}
		streams[res.remote.identity] = res.stream
	}
	sess, err := r.createSession(streams)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not start stream session: %w", err)
	}
	return sess, sess, nil
}

// when context is done: 1) cancel pending streams 2) reset
// & free established streams 3) close output channel
func (r rpc) openStreams(ctx context.Context,
	p protocol.ID, addrs []address) chan result {
	r.logger.Debug().Msgf("opening streams to %v...", addrs)
	// dial each player concurrently
	var wg sync.WaitGroup
	results := make(chan result, len(addrs))
	for _, addr := range addrs {
		wg.Add(1)
		go func(addr address) {
			defer wg.Done()
			// free stream when ctx is done
			stream, err := r.mino.host.NewStream(ctx, addr.identity, p)
			// collect established stream or error
			if err != nil {
				r.logger.Debug().Err(err).Msgf("could not open stream to %v", addr)
				results <- result{
					remote: addr,
					err:    xerrors.Errorf("could not open stream: %w", err),
				}
				return
			}
			r.logger.Debug().Msgf("opened stream to %v", addr)
			results <- result{remote: addr, stream: stream}
			go func() { // reset established stream
				<-ctx.Done()
				stream.Reset() // todo log error
				r.logger.Debug().Msgf("reset stream to %v", addr)
			}()
		}(addr)
	}
	// close output channels when no more pending streams
	go func() {
		wg.Wait()
		close(results)
		r.logger.Debug().Msg("finished opening streams")
	}()
	return results
}

// createSession
// session ends automatically on both initiator & player side by closing
// the incoming message channel when the initiator is done and cancels the
// stream context which resets all streams
func (r rpc) createSession(streams map[peer.ID]network.Stream) (*session, error) {
	// listen for incoming messages till streams are reset
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
		return xerrors.Errorf("could not serialize message: %w", err)
	}
	_, err = stream.Write(data)
	if err != nil {
		return xerrors.Errorf("could not write to stream: %w", err)
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
			"unexpected: could not create sender address: %w",
			err)
	}
	buffer := make([]byte, MaxMessageSize)
	n, err := stream.Read(buffer)
	if err != nil {
		return sender, nil, xerrors.Errorf(
			"could not read from stream: %w",
			err)
	}
	msg, err := f.Deserialize(c, buffer[:n])
	if err != nil {
		return sender, nil, xerrors.Errorf(
			"could not deserialize message: %w",
			err)
	}
	return sender, msg, nil
}
