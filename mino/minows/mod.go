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
	"github.com/libp2p/go-libp2p/core/peer"
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

// newMinows creates a new Minows instance.
// listen: local listening address in multiaddress format,
// e.g. /ip4/0.0.0.0/tcp/80/ws
// public: public dial-able address in multiaddress format,
// e.g. /dns4/p2p-1.c4dt.dela.org/tcp/443/wss
// secret: private key representing this mino instance's identity
func newMinows(listen, public ma.Multiaddr, secret crypto.PrivKey) (*minows,
	error) {
	id, err := peer.IDFromPrivateKey(secret)
	if err != nil {
		return nil, xerrors.Errorf("could not derive identity: %w", err)
	}
	myAddr, err := newAddress(public, id)
	if err != nil {
		return nil, xerrors.Errorf("could not create address: %v", err)
	}

	h, err := libp2p.New(libp2p.ListenAddrs(listen), libp2p.Identity(secret))
	if err != nil {
		return nil, xerrors.Errorf("could not create host: %v", err)
	}

	return &minows{
		logger:   dela.Logger.With().Str("mino", myAddr.String()).Logger(),
		myAddr:   myAddr,
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
