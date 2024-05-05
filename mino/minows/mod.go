package minows

import (
	"github.com/rs/zerolog"
	"go.dedis.ch/dela"
	"go.dedis.ch/dela/serde/json"
	"regexp"
	"strings"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

var pattern = regexp.MustCompile("^[a-zA-Z0-9]+$")

// Minows
// - implements mino.Mino
type minows struct {
	logger zerolog.Logger

	myAddr   address
	host     host.Host
	context  serde.Context
	segments []string
	rpcs     map[string]any
}

// NewMinows creates a new Minows instance that starts listening.
// listen: local listening address in multiaddress format,
// e.g. /ip4/0.0.0.0/tcp/80/ws
// public: public dial-able address in multiaddress format,
// e.g. /dns4/p2p-1.c4dt.dela.org/tcp/443/wss
// secret: private key representing this mino instance's identity
func NewMinows(listen, public ma.Multiaddr, secret crypto.PrivKey) (mino.Mino,
	error) {
	h, err := libp2p.New(libp2p.ListenAddrs(listen), libp2p.Identity(secret))
	if err != nil {
		return nil, xerrors.Errorf("could not start host: %v", err)
	}

	if public.Equal(listen) {
		const localhost = "127.0.0.1"
		listening, ok := findAddress(h, ma.P_IP4, localhost)
		if !ok {
			return nil, xerrors.Errorf("no local listening address found")
		}
		public = listening
	}

	myAddr, err := newAddress(public, h.ID())
	if err != nil {
		return nil, xerrors.Errorf("could not create public address: %v", err)
	}

	return &minows{
		logger:   dela.Logger.With().Str("mino", myAddr.String()).Logger(),
		myAddr:   myAddr, // TODO replace localhost port 0 with actual port
		segments: nil,
		host:     h,
		context:  json.NewContext(),
		rpcs:     make(map[string]any),
	}, nil
}

// NewMinowsLocal creates a new Minows instance with the local `listen`
// address as the public dial-able address.
// If `listen` has TCP port 0,
// it will be replaced with the actual port the host is listening on.
// This is useful for testing.
func NewMinowsLocal(listen ma.Multiaddr, secret crypto.PrivKey) (mino.Mino,
	error) {
	return NewMinows(listen, listen, secret)
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
		rpcs:     make(map[string]any),
		context:  m.context,
	}
}

func (m *minows) CreateRPC(name string, h mino.Handler, f serde.Factory) (mino.RPC, error) {
	if len(m.rpcs) == 0 {
		for _, seg := range m.segments {
			if !pattern.MatchString(seg) {
				return nil, xerrors.Errorf("invalid segment: %s", seg)
			}
		}
	}

	if !pattern.MatchString(name) {
		return nil, xerrors.Errorf("invalid name: %s", name)
	}
	_, found := m.rpcs[name]
	if found {
		return nil, xerrors.Errorf("already exists rpc: %s", name)
	}

	uri := strings.Join(append(m.segments, name), "/")

	r := &rpc{
		logger:  m.logger.With().Str("rpc", uri).Logger(),
		uri:     uri,
		mino:    m,
		factory: f,
		context: m.context,
	}

	m.host.SetStreamHandler(protocol.ID(uri+PostfixCall),
		r.createCallHandler(h))
	m.host.SetStreamHandler(protocol.ID(uri+PostfixStream),
		r.createStreamHandler(h))
	m.rpcs[name] = nil

	return r, nil
}

func (m *minows) stop() error {
	return m.host.Close()
}

func findAddress(h host.Host, protocol int, value string) (ma.Multiaddr, bool) {
	for _, addr := range h.Addrs() {
		ip, err := addr.ValueForProtocol(protocol)
		if err == nil && ip == value {
			return addr, true
		}
	}
	return nil, false
}
