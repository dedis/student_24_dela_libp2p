package minows

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAddress_New(t *testing.T) {
	tests := map[string]struct {
		multiaddr string
		pid       string
		err       bool
	}{
		"invalid multiaddr": {
			multiaddr: "example.com",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			err:       true,
		},
		"invalid pid": {
			multiaddr: "/ip4/0.0.0.0/tcp/80",
			pid:       "XmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			err:       true,
		},
		"all interface": {
			multiaddr: "/ip4/0.0.0.0/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			err:       false,
		},
		"localhost": {
			multiaddr: "/ip4/127.0.0.1/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			err:       false,
		},
		"hostname": {
			multiaddr: "/dns4/example.com/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			err:       false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// no exported fields on address type, ignored
			if _, err := NewAddress(tt.multiaddr, tt.pid); tt.err {
				// returns any error when failed to create address
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddress_Equal(t *testing.T) {
	tests := []struct {
		name      string
		multiaddr string
		pid       string
		out       bool
	}{
		{
			name:      "self",
			multiaddr: "/dns4/example.com/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			out:       true,
		},
		{
			name:      "copy",
			multiaddr: "/dns4/example.com/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			out:       true,
		},
		{
			name:      "diff multiaddr",
			multiaddr: "/dns4/example.com/tcp/8080",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			out:       false,
		},
		{
			name:      "diff pid",
			multiaddr: "/dns4/example.com/tcp/80",
			pid:       "QmVt9t5Tk2uEoA4CDbKNCVxqrut8UXmWHXvFZ8wFZ3ghhv",
			out:       false,
		},
	}
	addresses := make([]Address, len(tests))
	for i, tt := range tests {
		addresses[i] = newAddressOrPanic(tt.multiaddr, tt.pid)
		reference := addresses[0]
		current := addresses[i]

		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.out, current.Equal(reference))
		})
	}
}

func TestAddress_String(t *testing.T) {
	tests := map[string]struct {
		multiaddr string
		pid       string
		out       string
	}{
		"all interface": {
			multiaddr: "/ip4/0.0.0.0/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			out:       "/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		},
		"localhost": {
			multiaddr: "/ip4/127.0.0.1/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			out:       "/ip4/127.0.0.1/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		},
		"hostname": {
			multiaddr: "/dns4/example.com/tcp/80",
			pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
			out:       "/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			addr := newAddressOrPanic(tt.multiaddr, tt.pid)

			require.Equal(t, tt.out, addr.String())
		})
	}
}

var testsMarshalText = map[string]struct {
	multiaddr string
	pid       string
	out       []byte
}{
	"all interface": {
		multiaddr: "/ip4/0.0.0.0/tcp/80",
		pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		out:       []byte("/ip4/0.0.0.0/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
	},
	"localhost": {
		multiaddr: "/ip4/127.0.0.1/tcp/80",
		pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		out:       []byte("/ip4/127.0.0.1/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
	},
	"hostname": {
		multiaddr: "/dns4/example.com/tcp/80",
		pid:       "QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU",
		out:       []byte("/dns4/example.com/tcp/80/p2p/QmaD31nEzFGwD8dK96UFWHtTYTqYJgHLMYSFz4W4Hm2WCU"),
	},
}

func TestAddress_MarshalText(t *testing.T) {
	for name, tt := range testsMarshalText {
		t.Run(name, func(t *testing.T) {
			addr := newAddressOrPanic(tt.multiaddr, tt.pid)

			result, err := addr.MarshalText()
			require.NoError(t, err)
			require.Equal(t, tt.out, result)
		})
	}
}

func TestAddressFactory_FromText(t *testing.T) {
	factory := addressFactory{}
	for name, tt := range testsMarshalText {
		t.Run(name, func(t *testing.T) {
			expected := newAddressOrPanic(tt.multiaddr, tt.pid)

			result := factory.FromText(tt.out)
			require.True(t, expected.Equal(result))
		})
	}
}

func newAddressOrPanic(multiaddr, pid string) Address {
	addr, err := NewAddress(multiaddr, pid)
	if err != nil {
		panic(err)
	}
	return addr
}
