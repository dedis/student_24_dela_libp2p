package minows

import (
	"context"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"sync"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const MaxMessageSize = 1e9 // TODO verify

// RPC implements mino.RPC
// TODO unit tests
type rpc struct {
	uri  protocol.ID
	mino *minows
	// TODO handler mino.Handler
	factory serde.Factory
	context serde.Context
}

// Call
// Returns an error if any player address is invalid.
// Otherwise, returns a response channel 1) filled with replies or errors in
// the network from each player 2) closed after each player has
// replied or errored, or the context is done.
func (r rpc) Call(
	ctx context.Context,
	req serde.Message,
	players mino.Players,
) (<-chan mino.Response, error) {
	// TODO assumption: peer store already filled with 'players' Peer IDs & multi-addresses
	// quit unless all player addresses are valid
	addrs, err := toAddresses(players)
	if err != nil {
		return nil, err
	}
	results := openStreams(ctx, r.mino.host, r.uri, addrs)
	// unicast a request-response to each player concurrently
	// as streams are established
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
				reply := receive(res.stream, r.factory, r.context)
				if err != nil {
					responses <- mino.NewResponseWithError(reply.sender, reply.err)
					return
				}
				responses <- mino.NewResponse(reply.sender, reply.message)
			case <-ctx.Done(): // let goroutine exit if context is done
			}
		}()
	}
	go func() {
		wg.Wait()
		close(responses)
	}()
	return responses, nil
}

// Stream
// - context defines when the protocol is done,
// and it should therefore always be canceled at some point. (DELA Doc)
// - When it's done, all the connections are shut down and the resources are
// cleaned up (DELA Doc)
// - orchestrator of a protocol will contact one of the participants which
// will be the root for the routing algorithm (i.e. gateway?).
// It will then relay the messages according to the routing algorithm and
// create relays to other peers when necessary (DELA Doc but ignored)
func (r rpc) Stream(ctx context.Context, players mino.Players) (mino.Sender, mino.Receiver, error) {
	// TODO assumption: peer store already filled with 'players' Peer IDs & multi-addresses
	// quit unless all player addresses are valid
	addrs, err := toAddresses(players)
	if err != nil {
		return nil, nil, err
	}
	results := openStreams(ctx, r.mino.host, r.uri, addrs)
	// wait till all streams are established successfully or quit
	streams := make(map[peer.ID]network.Stream)
	for res := range results {
		if res.err != nil {
			return nil, nil, res.err
		}
		streams[res.remote.identity] = res.stream
	}
	sess, err := r.createSession(ctx, streams)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not start stream session: %v", err)
	}
	return sess, sess, nil
}

func (r rpc) createSession(ctx context.Context,
	streams map[peer.ID]network.Stream) (*session, error) {
	// listen for incoming messages till context is done
	in := make(chan envelope) // unbuffered
	for _, stream := range streams {
		go func(stream network.Stream) {
			for {
				select {
				case in <- receive(stream, r.factory, r.context): // fan-in
				case <-ctx.Done():
					return
				}
			}
		}(stream)
	}
	// close incoming message channel to signal session ended
	go func() {
		<-ctx.Done()
		close(in)
	}()
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
			return nil, xerrors.Errorf("invalid address type: %T", next)
		}
		addrs = append(addrs, addr)
	}
	return addrs, nil
}

type result struct {
	remote address
	stream network.Stream
	err    error
}

func openStreams(ctx context.Context, h host.Host, uri protocol.ID,
	addrs []address) chan result {
	// dial each participant concurrently
	var wg sync.WaitGroup
	results := make(chan result, len(addrs))
	for _, addr := range addrs {
		wg.Add(1)
		go func(addr address) {
			defer wg.Done()
			stream, err := h.NewStream(ctx, addr.identity, uri)
			// collect established stream or error
			if err != nil {
				results <- result{
					remote: addr,
					err:    xerrors.Errorf("could not open stream: %v", err),
				}
				return
			}
			results <- result{remote: addr, stream: stream}
			go func() { // free established stream
				<-ctx.Done()
				stream.Reset()
			}()
		}(addr)
	}
	// close output channels when no more pending streams
	go func() {
		wg.Wait()
		close(results)
	}()
	return results
}

func send(stream network.Stream, msg serde.Message, c serde.Context) error {
	data, err := msg.Serialize(c)
	if err != nil {
		return xerrors.Errorf("could not serialize message: %v", err)
	}
	_, err = stream.Write(data)
	if err != nil {
		return xerrors.Errorf("could not write to stream: %v", err)
	}
	return nil
}

type envelope struct {
	sender  address
	message serde.Message
	err     error
}

func receive(stream network.Stream,
	f serde.Factory, c serde.Context) envelope {
	sender, err := newAddress(
		stream.Conn().RemoteMultiaddr(),
		stream.Conn().RemotePeer())
	if err != nil {
		return envelope{err: xerrors.Errorf(
			"unexpected: could not create sender address: %v",
			err)}
	}
	buffer := make([]byte, MaxMessageSize)
	n, err := stream.Read(buffer)
	if err != nil {
		return envelope{sender: sender, err: xerrors.Errorf(
			"could not read from stream: %v",
			err)}
	}
	msg, err := f.Deserialize(c, buffer[:n])
	if err != nil {
		return envelope{sender: sender, err: xerrors.Errorf(
			"could not deserialize message: %v",
			err)}
	}
	return envelope{sender: sender, message: msg}
}
