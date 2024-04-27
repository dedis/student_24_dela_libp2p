package minows

import (
	"github.com/libp2p/go-libp2p/core/network"
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
	myAddr    address
	namespace []string
	host      host.Host
	context   serde.Context
	rpcs      map[string]rpc
}

// NewMinows
// listen: local listening address in multiaddress format,
// e.g. /ip4/0.0.0.0/tcp/80
// public: public dial-able address in multiaddress format,
// e.g. /dns4/p2p-1.c4dt.dela.org/tcp/443/wss
func NewMinows(listen, public ma.Multiaddr, secret crypto.PrivKey) (*minows, error) {
	id, err := peer.IDFromPrivateKey(secret)
	if err != nil {
		return nil, xerrors.Errorf("could not get Peer ID: %v", err)
	}
	myAddr, err := newAddress(public, id)
	if err != nil {
		return nil, xerrors.Errorf("could not create address: %v", err)
	}
	// create host & start listening
	h, err := libp2p.New(libp2p.ListenAddrs(listen), libp2p.Identity(secret))
	if err != nil {
		return nil, xerrors.Errorf("could not create host: %v", err)
	}
	// TODO populate peer store with multiaddr & peer IDs of other peers
	return &minows{
		myAddr:    myAddr,
		namespace: nil,
		host:      h,
		context:   json.NewContext(),
		rpcs:      make(map[string]rpc),
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

	// does not copy existing rpcs when creating mino for new namespace
	return &minows{
		myAddr:    m.myAddr,
		namespace: append(m.namespace, segment),
		host:      m.host,
		rpcs:      make(map[string]rpc),
		context:   m.context,
	}
}

func (m *minows) CreateRPC(name string, h mino.Handler, f serde.Factory) (mino.RPC, error) {
	pattern := regexp.MustCompile("^[a-zA-Z0-9]+$")
	// validate namespace if no RPC created yet
	if len(m.rpcs) == 0 {
		for _, seg := range m.namespace {
			if !pattern.MatchString(seg) {
				return nil, xerrors.Errorf("invalid segment: %s", seg)
			}
		}
	}
	// validate name
	if !pattern.MatchString(name) {
		return nil, xerrors.Errorf("invalid name: %s", name)
	}
	if _, found := m.rpcs[name]; found {
		return nil, xerrors.Errorf("already exists rpc: %s", name)
	}
	// create rpc
	uri := strings.Join(append(m.namespace, name), "/")
	r := &rpc{
		uri:     uri,
		mino:    m,
		factory: f,
		context: m.context,
	}
	// TODO need to be thread safe?
	m.rpcs[name] = *r
	// start listening
	m.host.SetStreamHandler(protocol.ID(uri+"/call"),
		callHandler(r, h))
	m.host.SetStreamHandler(protocol.ID(uri+"/stream"),
		streamHandler(r, h))
	// TODO when to return pointer vs struct?
	return r, nil
}

func (m *minows) close() error {
	return m.host.Close()
}

func callHandler(r *rpc, h mino.Handler) func(stream network.Stream) {
	return func(stream network.Stream) {
		sender, msg, err := receive(stream, r.factory, r.context)
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not receive call request: %v", err))
			return
		}
		resp, err := h.Process(mino.Request{Address: sender, Message: msg})
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not process call request: %v", err))
			return
		}
		err = send(stream, resp, r.context)
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not send call response: %v", err))
			return
		}
		// initiator resets & frees the stream
	}
}

func streamHandler(r *rpc, h mino.Handler) func(stream network.Stream) {
	return func(stream network.Stream) {
		sess, err := r.createSession(
			map[peer.ID]network.Stream{stream.Conn().RemotePeer(): stream})
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not start stream session: %v", err))
			return
		}
		err = h.Stream(sess, sess)
		if err != nil {
			dela.Logger.Err(xerrors.Errorf(
				"could not handle stream: %v", err))
			return
		}
		// initiator resets & frees the stream
	}
}
