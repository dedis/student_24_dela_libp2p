// This file implements the address abstraction for nodes communicating over distributed network using libp2p.

package minows

import (
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
)

// - implements mino.Address
type address struct {
	multiaddr ma.Multiaddr
}

// Equal implements mino.Address.
func (a address) Equal(other mino.Address) bool {
	addr, ok := other.(address)
	// TODO: handle multiaddr possibly nil
	return ok && a.multiaddr.Equal(addr.multiaddr)
}

// String implements fmt.Stringer.
func (a address) String() string {
	return a.multiaddr.String()
}

// ConnectionType implements mino.Address
func (a address) ConnectionType() mino.AddressConnectionType {
	return mino.ACTws // TODO ACTwss
}

// MarshalText implements encoding.TextMarshaler.
func (a address) MarshalText() ([]byte, error) {
	return []byte(a.multiaddr.String()), nil
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
	multiaddr, err := ma.NewMultiaddr(string(text))
	if err != nil {
		return address{}
	}
	return address{multiaddr: multiaddr}
}
