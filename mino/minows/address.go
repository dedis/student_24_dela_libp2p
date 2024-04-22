package minows

import (
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const protocolP2P = "/p2p/"

type address struct {
	location ma.Multiaddr // connection address
	identity peer.ID
}

// todo remove, unnecessary, construct via struct literal more convenient
func newAddress(location ma.Multiaddr, id peer.ID) (address, error) {
	// validate
	if location == nil || id.String() == "" {
		return address{}, xerrors.New("address must have location and identity")
	}

	return address{
		location: location,
		identity: id,
	}, nil
}

// todo remove, export identity instead
func (a address) PeerID() peer.ID {
	return a.identity
}

// Equal implements mino.Address.
func (a address) Equal(other mino.Address) bool {
	o, ok := other.(address)
	return ok && a.location.Equal(o.location) && a.identity == o.identity
}

// String implements fmt.Stringer.
func (a address) String() string {
	return fmt.Sprintf("%s%s%s", a.location, protocolP2P, a.identity)
}

// ConnectionType implements mino.Address
func (a address) ConnectionType() mino.AddressConnectionType {
	// TODO implement
	panic("not implemented")
}

// MarshalText implements encoding.TextMarshaler.
func (a address) MarshalText() ([]byte, error) {
	return []byte(fmt.Sprintf("%s%s%s", a.location, protocolP2P, a.identity)),
		nil
}

// addressFactory is a factory to deserialize Minows addresses.
//
// - implements mino.AddressFactory
type addressFactory struct {
	serde.Factory
}

// FromText implements mino.AddressFactory. It returns an instance of an address
// from a byte slice.
// Returns nil if fails
func (f addressFactory) FromText(text []byte) mino.Address {
	loc, id, found := strings.Cut(string(text), protocolP2P)
	if !found {
		// todo log error
		return nil
	}
	location, err := ma.NewMultiaddr(loc)
	if err != nil {
		// todo log error
		return nil
	}
	identity, err := peer.Decode(id)
	if err != nil {
		// todo log error
		return nil
	}
	addr, err := newAddress(location, identity)
	if err != nil {
		// todo log error
		return nil
	}
	return addr
}
