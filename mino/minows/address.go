// This file implements the address abstraction for nodes communicating over distributed network using libp2p.

package minows

import (
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

// Address - implements mino.Address
type Address struct {
	location ma.Multiaddr // connection address
	identity peer.ID
}

func NewAddress(location ma.Multiaddr, id peer.ID) (Address, error) {
	return Address{
		location: location,
		identity: id,
	}, nil
}

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
	return fmt.Sprintf("%s/p2p/%s", a.location.String(), a.identity)
}

// ConnectionType implements mino.Address
func (a Address) ConnectionType() mino.AddressConnectionType {
	// TODO implement
	panic("not implemented")
}

// MarshalText implements encoding.TextMarshaler.
func (a Address) MarshalText() ([]byte, error) {
	p2p, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", a.identity))
	if err != nil {
		return nil, xerrors.Errorf("could not marshal identity: %v", err)
	}
	return []byte(a.location.Encapsulate(p2p).String()), nil
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
	full, err := ma.NewMultiaddr(string(text))
	if err != nil {
		// todo log error
		return nil
	}
	location, p2p := ma.SplitLast(full)
	value, err := p2p.ValueForProtocol(ma.P_P2P)
	if err != nil {
		// todo log error
		return nil
	}
	id, err := peer.Decode(value)
	if err != nil {
		// todo log error
		return nil
	}
	return Address{
		location: location,
		identity: id,
	}
}
