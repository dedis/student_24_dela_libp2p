// This file implements the address abstraction for nodes communicating over distributed network using libp2p.

package minows

import (
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

// - implements mino.Address
type address struct {
	multiaddr ma.Multiaddr
}

func NewAddress(multiaddr string) (mino.Address, error) {
	parsed, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		return nil, xerrors.Errorf("could not create address: %v", err)
	}
	return address{multiaddr: parsed}, nil
}

// Equal implements mino.Address.
func (a address) Equal(other mino.Address) bool {
	addr, ok := other.(address)
	return ok && a.multiaddr.Equal(addr.multiaddr)
}

// String implements fmt.Stringer.
func (a address) String() string {
	return a.multiaddr.String()
}

// ConnectionType implements mino.Address
func (a address) ConnectionType() mino.AddressConnectionType {
	return mino.ACTws
}

// MarshalText implements encoding.TextMarshaler.
func (a address) MarshalText() ([]byte, error) {
	return []byte(a.multiaddr.String()), nil
}

// AddressFactory is a factory to deserialize Minows addresses.
//
// - implements mino.AddressFactory
type AddressFactory struct {
	serde.Factory
}

// FromText implements mino.AddressFactory. It returns an instance of an address
// from a byte slice.
func (f AddressFactory) FromText(text []byte) mino.Address {
	multiaddr, err := ma.NewMultiaddr(string(text))
	if err != nil {
		return address{}
	}
	return address{multiaddr: multiaddr}
}
