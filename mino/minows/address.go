package minows

import (
	"fmt"
	"go.dedis.ch/dela"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"golang.org/x/xerrors"
)

const protocolP2P = "/p2p/"

// address represents a publicly reachable network address that can be used
// to establish communication with a remote player through libp2p and therefore,
// must have both `location` and `identity` components.
type address struct {
	location ma.Multiaddr // required
	identity peer.ID      // required
}

// newAddress creates a new address from a publicly reachable location and a
// peer identity.
func newAddress(location ma.Multiaddr, identity peer.ID) (address, error) {
	if location == nil || identity.String() == "" {
		return address{}, xerrors.New("address must have location and identity")
	}
	return address{
		location: location,
		identity: identity,
	}, nil
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
// Not used by minows
func (a address) ConnectionType() mino.AddressConnectionType {
	return mino.ACTws
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
// Returns nil if fails.
func (f addressFactory) FromText(text []byte) mino.Address {
	str := string(text)
	loc, id, found := strings.Cut(str, protocolP2P)
	if !found {
		dela.Logger.Error().Msgf("%q misses p2p protocol", str)
		return nil
	}
	location, err := ma.NewMultiaddr(loc)
	if err != nil {
		dela.Logger.Error().Msgf("could not parse %q as multiaddress",
			loc)
		return nil
	}
	identity, err := peer.Decode(id)
	if err != nil {
		dela.Logger.Error().Msgf("could not decode %q as peer ID", id)
		return nil
	}
	addr, err := newAddress(location, identity)
	if err != nil {
		dela.Logger.Error().Msgf("could not create address: %v", err)
		return nil
	}
	return addr
}
