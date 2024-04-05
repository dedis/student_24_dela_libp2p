// This file implements the address abstraction for nodes communicating over distributed network using libp2p.

package minows

import (
	"fmt"

	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

// Address - implements mino.Address
type Address struct {
	location ma.Multiaddr
	identity ma.Multiaddr
}

func NewAddress(multiaddr, pid string) (Address, error) {
	location, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		return Address{}, xerrors.Errorf("could not parse multiaddr: %v", err)
	}
	identity, err := ma.NewMultiaddr(fmt.Sprintf("/p2p/%s", pid))
	if err != nil {
		return Address{}, xerrors.Errorf("could not parse peer identity: %v", err)
	}
	return Address{
		location: location,
		identity: identity,
	}, nil
}

// Equal implements mino.Address.
func (a Address) Equal(other mino.Address) bool {
	o, ok := other.(Address)
	return ok && a.location.Equal(o.location) && a.identity.Equal(o.identity)
}

// String implements fmt.Stringer.
func (a Address) String() string {
	return a.location.Encapsulate(a.identity).String()
}

// ConnectionType implements mino.Address
func (a Address) ConnectionType() mino.AddressConnectionType {
	// TODO implement
	panic("not implemented")
}

// MarshalText implements encoding.TextMarshaler.
func (a Address) MarshalText() ([]byte, error) {
	return []byte(a.location.Encapsulate(a.identity).String()), nil
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
		// TODO log error
		return Address{}
	}
	location, identity := ma.SplitLast(full)
	return Address{
		location: location,
		identity: identity,
	}
}
