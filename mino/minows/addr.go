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

// Address should not be instantiated via struct literal
type Address struct {
	// TODO may export both fields to populate peer store after parsing
	//  addresses of all peers
	location ma.Multiaddr // connection address
	identity peer.ID
}

func NewAddress(location ma.Multiaddr, id peer.ID) (Address, error) {
	// validate
	if location == nil || id.String() == "" {
		return Address{}, xerrors.New("address must have location and identity")
	}

	return Address{
		location: location,
		identity: id,
	}, nil
}

// todo remove, export identity instead
func (a Address) PeerID() peer.ID {
	return a.identity
}

// Equal implements mino.Address.
func (a Address) Equal(other mino.Address) bool {
	o, ok := other.(Address)
	return ok && a.location.Equal(o.location) && a.identity == o.identity
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return fmt.Sprintf("%s%s%s", a.location, protocolP2P, a.identity)
}

// ConnectionType implements mino.Address
func (a Address) ConnectionType() mino.AddressConnectionType {
	// TODO implement
	panic("not implemented")
}

// MarshalText implements encoding.TextMarshaler.
func (a Address) MarshalText() ([]byte, error) {
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
func (f addressFactory) FromText(text []byte) mino.Address {
	loc, id, found := strings.Cut(string(text), protocolP2P)
	if !found {
		// todo log error
		return Address{}
	}
	location, err := ma.NewMultiaddr(loc)
	if err != nil {
		// todo log error
		return Address{}
	}
	identity, err := peer.Decode(id)
	if err != nil {
		// todo log error
		return Address{}
	}
	addr, err := NewAddress(location, identity)
	if err != nil {
		// todo log error
		return Address{}
	}
	return addr
}
