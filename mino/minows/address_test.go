package minows

import (
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/mino"
	"testing"

	"github.com/stretchr/testify/require"
)

// todo make local to test functions
const (
	AddrAllInterface = "/ip4/0.0.0.0/tcp/80"
	AddrLocalhost    = "/ip4/127.0.0.1/tcp/80"
	AddrHostname     = "/dns4/example.com/tcp/80"
)
const (
	PID1 = "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"
	PID2 = "QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv"
)

func Test_newAddress(t *testing.T) {
	type args struct {
		location ma.Multiaddr
		identity peer.ID
	}
	tests := map[string]struct {
		args args
	}{
		"all interface": {
			args: args{mustCreateMultiaddress(AddrAllInterface), mustCreatePeerID(PID1)},
		},
		"localhost": {
			args: args{mustCreateMultiaddress(AddrLocalhost), mustCreatePeerID(PID2)},
		},
		"hostname": {
			args: args{mustCreateMultiaddress(AddrHostname), mustCreatePeerID(PID1)},
		},
	}
	t.Parallel() // run this test function args parallel to other test functions
	for name, tt := range tests {
		tt := tt // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel() // run this test case args parallel to other test cases
			// no exported a on 'a' type, ignored
			_, err := newAddress(tt.args.location, tt.args.identity)
			require.NoError(t, err)
		})
	}
}

func Test_newAddress_Invalid(t *testing.T) {
	tests := map[string]struct {
		location ma.Multiaddr
		identity peer.ID
	}{
		"missing location": {
			location: nil,
			identity: mustCreatePeerID(PID1),
		},
		"missing identity": {
			location: mustCreateMultiaddress(AddrAllInterface),
			identity: "",
		},
	}

	t.Parallel() // run this test function in parallel to other test functions
	for name, tt := range tests {
		tt := tt // capture range variable
		t.Run(name, func(t *testing.T) {
			t.Parallel() // run this test case in parallel to other test cases
			// no exported fields on Address type, ignored
			_, err := newAddress(tt.location, tt.identity)
			require.Error(t, err)
		})
	}
}

func Test_address_Equal(t *testing.T) {
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

func Test_address_String(t *testing.T) {
	tests := map[string]struct {
		a    address
		want string
	}{
		"all interface": {
			a:    mustCreateAddress(AddrAllInterface, PID1),
			want: "/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		},
		"localhost": {
			a:    mustCreateAddress(AddrLocalhost, PID2),
			want: "/ip4/127.0.0.1/tcp/80/p2p/QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv",
		},
		"hostname": {
			a:    mustCreateAddress(AddrHostname, PID1),
			want: "/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		},
	}
	t.Parallel()
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := tt.a.String()

			require.Equal(t, tt.want, result)
		})
	}
}

func Test_address_MarshalText(t *testing.T) {
	tests := map[string]struct {
		a    address
		want []byte
	}{
		"all interface": {
			a:    mustCreateAddress(AddrAllInterface, PID1),
			want: []byte("/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
		},
		"localhost": {
			a:    mustCreateAddress(AddrLocalhost, PID2),
			want: []byte("/ip4/127.0.0.1/tcp/80/p2p/QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv"),
		},
		"hostname": {
			a:    mustCreateAddress(AddrHostname, PID1),
			want: []byte("/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
		},
	}
	t.Parallel()
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := tt.a.MarshalText()

			require.NoError(t, err)
			require.Equal(t, tt.want, result)
		})
	}
}

func Test_addressFactory_FromText(t *testing.T) {
	tests := map[string]struct {
		args []byte
		want address
	}{
		"all interface": {
			args: []byte("/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
			want: mustCreateAddress(AddrAllInterface, PID1),
		},
		"localhost": {
			args: []byte("/ip4/127.0.0.1/tcp/80/p2p/QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv"),
			want: mustCreateAddress(AddrLocalhost, PID2),
		},
		"hostname": {
			args: []byte("/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
			want: mustCreateAddress(AddrHostname, PID1),
		},
	}
	t.Parallel()
	factory := addressFactory{}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := factory.FromText(tt.args)

			require.Equal(t, tt.want, result)
		})
	}
}

func Test_addressFactory_FromText_Invalid(t *testing.T) {
	tests := map[string]struct {
		args []byte
	}{
		"invalid text": {
			args: []byte("invalid"),
		},
		"missing location": {
			args: []byte("/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
		},
		"missing identity": {
			args: []byte(AddrLocalhost),
		},
	}

	t.Parallel()
	factory := addressFactory{}
	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := factory.FromText(tt.args)

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
