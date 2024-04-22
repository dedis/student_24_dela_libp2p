package minows

import (
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	AddrAllInterface = "/ip4/0.0.0.0/tcp/80"
	AddrLocalhost    = "/ip4/127.0.0.1/tcp/80"
	AddrHostname     = "/dns4/example.com/tcp/80"
)
const (
	PID1 = "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"
	PID2 = "QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv"
)

func TestNewAddress(t *testing.T) {
	tests := map[string]struct {
		location ma.Multiaddr
		identity peer.ID
		err      bool
	}{
		"missing location": {
			location: nil,
			identity: mustCreatePeerID(PID1),
			err:      true,
		},
		"missing identity": {
			location: mustCreateMultiaddress(AddrAllInterface),
			identity: "",
			err:      true,
		},
		"all interface": {
			location: mustCreateMultiaddress(AddrAllInterface),
			identity: mustCreatePeerID(PID1),
		},
		"localhost": {
			location: mustCreateMultiaddress(AddrLocalhost),
			identity: mustCreatePeerID(
				PID2),
		},
		"hostname": {
			location: mustCreateMultiaddress(AddrHostname),
			identity: mustCreatePeerID(
				PID1),
		},
	}

	t.Parallel() // run this test function in parallel to other test functions
	for name, tt := range tests {
		tt := tt // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel() // run this test case in parallel to other test cases
			// no exported fields on Address type, ignored
			if _, err := newAddress(tt.location, tt.identity); tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddress_Equal(t *testing.T) {
	reference := mustCreateAddress(AddrHostname, PID1)
	tests := map[string]struct {
		self  address
		other mino.Address
		out   bool
	}{
		"self": {
			self:  reference,
			other: reference,
			out:   true,
		},
		"copy": {
			self:  reference,
			other: mustCreateAddress(AddrHostname, PID1),
			out:   true,
		},
		"diff location": {
			self:  reference,
			other: mustCreateAddress(AddrLocalhost, PID1),
			out:   false,
		},
		"diff identity": {
			self:  reference,
			other: mustCreateAddress(AddrHostname, PID2),
			out:   false,
		},
		"nil": {
			self:  reference,
			other: nil,
			out:   false,
		},
	}
	t.Parallel()
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := tt.self.Equal(tt.other)

			require.Equal(t, tt.out, result)
		})
	}
}

var sharedTests = map[string]struct {
	addr        address
	string      string
	marshalText []byte
}{
	"all interface": {
		addr:        mustCreateAddress(AddrAllInterface, PID1),
		string:      "/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		marshalText: []byte("/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
	},
	"localhost": {
		addr:        mustCreateAddress(AddrLocalhost, PID2),
		string:      "/ip4/127.0.0.1/tcp/80/p2p/QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv",
		marshalText: []byte("/ip4/127.0.0.1/tcp/80/p2p/QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv"),
	},
	"hostname": {
		addr:        mustCreateAddress(AddrHostname, PID1),
		string:      "/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		marshalText: []byte("/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
	},
}

func TestAddress_String(t *testing.T) {
	t.Parallel()
	for name, tt := range sharedTests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := tt.addr.String()

			require.Equal(t, tt.string, result)
		})
	}
}

func TestAddress_MarshalText(t *testing.T) {
	t.Parallel()
	for name, tt := range sharedTests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.addr.MarshalText()

			require.NoError(t, err)
			require.Equal(t, tt.marshalText, result)
		})
	}
}

func TestAddressFactory_FromText(t *testing.T) {
	t.Parallel()
	factory := addressFactory{}
	for name, tt := range sharedTests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := factory.FromText(tt.marshalText)

			require.Equal(t, tt.addr, result)
		})
	}
}

func TestAddressFactory_FromText_invalid(t *testing.T) {
	tests := map[string]struct {
		in []byte
	}{
		"invalid text": {
			in: []byte("invalid"),
		},
		"missing location": {
			in: []byte("/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
		},
		"missing identity": {
			in: []byte(AddrLocalhost),
		},
	}

	t.Parallel()
	factory := addressFactory{}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := factory.FromText(tt.in)

			require.Nil(t, result)
		})
	}
}

func mustCreateAddress(location string, identity string) address {
	addr, err := newAddress(mustCreateMultiaddress(location), mustCreatePeerID(identity))
	if err != nil {
		panic(err)
	}
	return addr
}

func mustCreateMultiaddress(address string) ma.Multiaddr {
	multiaddr, err := ma.NewMultiaddr(address)
	if err != nil {
		panic(err)
	}
	return multiaddr
}

func mustCreatePeerID(id string) peer.ID {
	pid, err := peer.Decode(id)
	if err != nil {
		panic(err)
	}
	return pid
}
