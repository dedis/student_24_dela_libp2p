package minows

import (
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/rs/zerolog"
	"go.dedis.ch/dela"
	"go.dedis.ch/dela/serde/json"
	"regexp"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

// Minows implements mino.Mino
type minows struct {
	logger zerolog.Logger

	myAddr   address // publicly reachable address
	host     host.Host
	context  serde.Context
	segments []string       // namespace
	rpcs     map[string]any // names
}

// newMinows
// listen: local listening address in multiaddress format,
// e.g. /ip4/0.0.0.0/tcp/80/ws
// public: public dial-able address in multiaddress format,
// e.g. /dns4/p2p-1.c4dt.dela.org/tcp/443/wss
func newMinows(listen, public ma.Multiaddr, secret crypto.PrivKey) (*minows,
	error) {
	// create publicly reachable address
	// todo extract function: newAddressKey(location ma.Multiaddr,
	//  secret crypto.PrivKey) (*address, error)
	id, err := peer.IDFromPrivateKey(secret)
	if err != nil {
		return nil, xerrors.Errorf("could not derive identity: %w", err)
	}
	dialable, err := newAddress(public, id)
	if err != nil {
		return nil, xerrors.Errorf("could not create address: %w", err)
	}
	// start listening last after all previous actions succeed
	// TODO extract function startListening(listen ma.Multiaddr, secret crypto.PrivKey) (host.Host, error)
	h, err := libp2p.New(libp2p.ListenAddrs(listen), libp2p.Identity(secret))
	if err != nil {
		return nil, xerrors.Errorf("could not create host: %w", err)
	}
	return &minows{
		logger:   dela.Logger.With().Str("mino", dialable.String()).Logger(),
		myAddr:   dialable,
		segments: nil,
		host:     h,
		context:  json.NewContext(),
		rpcs:     make(map[string]any),
	}, nil
}

func (m *minows) GetAddressFactory() mino.AddressFactory {
	return addressFactory{}
}

func (m *minows) GetAddress() mino.Address {
	return m.myAddr
}

func (m *minows) WithSegment(segment string) mino.Mino {
	if segment == "" {
		return m
	}

	return &minows{
		myAddr:   m.myAddr,
		segments: append(m.segments, segment),
		host:     m.host,
		rpcs:     make(map[string]any), // don't copy existing rpcs
		context:  m.context,
	}
}

func (m *minows) CreateRPC(name string, h mino.Handler, f serde.Factory) (mino.RPC, error) {
	pattern := regexp.MustCompile("^[a-zA-Z0-9]+$")
	// validate namespace
	if len(m.rpcs) == 0 { // namespace not validated yet
		// TODO extract function: validateSegments(segments []string,
		//  pattern *Regexp) error
		for _, seg := range m.segments {
			if !pattern.MatchString(seg) {
				return nil, xerrors.Errorf("invalid segment: %s", seg)
			}
		}
	}
	// validate name
	// TODO extract function: validateName(name string, pattern *Regexp,
	//  existing map[string]any) error
	if !pattern.MatchString(name) {
		return nil, xerrors.Errorf("invalid name: %s", name)
	}
	if _, found := m.rpcs[name]; found {
		return nil, xerrors.Errorf("already exists rpc: %s", name)
	}
	// TODO extract function: createUri(segments []string, name string) string
	uri := strings.Join(append(m.segments, name), "/")
	// create rpc
	r := &rpc{
		logger:  m.logger.With().Str("rpc", uri).Logger(),
		uri:     uri,
		mino:    m,
		factory: f,
		context: m.context,
	}
	// start listening for calls & streams
	// TODO extract method: registerHandler(uri string, h mino.Handler)
	m.host.SetStreamHandler(protocol.ID(uri+PostfixCall),
		r.createCallHandler(h))
	m.host.SetStreamHandler(protocol.ID(uri+PostfixStream),
		r.createStreamHandler(h))
	// TODO need to be thread safe?
	m.rpcs[name] = nil
	// TODO when to return pointer vs struct?
	return r, nil
}

func (m *minows) stop() error {
	return m.host.Close()
}

// todo move to rpc.go
func (r rpc) createCallHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		sender, msg, err := receive(stream, r.factory, r.context)
		if err != nil {
			r.logger.Err(xerrors.Errorf(
				"could not receive call request: %w", err))
			return
		}
		resp, err := h.Process(mino.Request{Address: sender, Message: msg})
		if err != nil {
			r.logger.Err(xerrors.Errorf(
				"could not process call request: %w", err))
			return
		}
		err = send(stream, resp, r.context)
		if err != nil {
			r.logger.Err(xerrors.Errorf(
				"could not send call response: %w", err))
			return
		}
		// initiator resets & frees the stream
	}
}

// todo move to rpc.go
func (r rpc) createStreamHandler(h mino.Handler) network.StreamHandler {
	return func(stream network.Stream) {
		sess, err := r.createSession(
			map[peer.ID]network.Stream{stream.Conn().RemotePeer(): stream})
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not start stream session: %w", err))
			return
		}
		err = h.Stream(sess, sess)
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not handle stream: %w", err))
			return
		}
		// initiator resets & frees the stream
	}
}
