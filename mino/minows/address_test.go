package minows

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
)

func TestAddress_New(t *testing.T) {
	// TODO consolidate New() and export methods such as String(),
	//  MarshalText() to use the same set of 'tests' table
	tests := map[string]struct {
		in  string
		err bool // expect generic error (no specific error type/message)
	}{
		"url (not multiaddr)": {in: "example.com", err: true},
		"all interface":       {in: "/ip4/0.0.0.0/tcp/80", err: false},
		"localhost":           {in: "/ip4/127.0.0.1/tcp/80", err: false},
		"hostname":            {in: "/dns4/example.com/tcp/80", err: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// no exported fields on address type, ignore return value
			_, err := NewAddress(tt.in)
			if tt.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddress_Equal(t *testing.T) {
	addr, _ := NewAddress("/ip4/0.0.0.0/tcp/80")
	copied, _ := NewAddress("/ip4/0.0.0.0/tcp/80")
	other, _ := NewAddress("/ip4/127.0.0.1/tcp/80")

	require.True(t, addr.Equal(addr))
	require.True(t, addr.Equal(copied))
	require.False(t, addr.Equal(other))
	require.False(t, addr.Equal(fakeAddress{}))
}

func TestAddress_String(t *testing.T) {
	tests := map[string]struct {
		in string
	}{
		"all interface": {in: "/ip4/0.0.0.0/tcp/80"},
		"localhost":     {in: "/ip4/127.0.0.1/tcp/80"},
		"hostname":      {in: "/dns4/example.com/tcp/80"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			addr, _ := NewAddress(tt.in)
			require.Equal(t, tt.in, addr.String())
		})
	}
}

var testsMarshalText = map[string]struct {
	in  string
	out []byte
}{
	"all interface": {in: "/ip4/0.0.0.0/tcp/80", out: []byte("/ip4/0.0.0.0/tcp/80")},
	"localhost":     {in: "/ip4/127.0.0.1/tcp/80", out: []byte("/ip4/127.0.0.1/tcp/80")},
	"hostname":      {in: "/dns4/example.com/tcp/80", out: []byte("/dns4/example.com/tcp/80")},
}

func TestAddress_MarshalText(t *testing.T) {
	for name, tt := range testsMarshalText {
		t.Run(name, func(t *testing.T) {
			addr, _ := NewAddress(tt.in)

			result, err := addr.MarshalText()
			require.NoError(t, err)
			require.Equal(t, tt.out, result)
		})
	}
}

func TestAddressFactory_FromText(t *testing.T) {
	factory := AddressFactory{}
	for name, tt := range testsMarshalText {
		t.Run(name, func(t *testing.T) {
			expected, _ := NewAddress(tt.in)

			result := factory.FromText(tt.out)
			require.Equal(t, expected, result)
		})
	}
}

// ----------------------------------------------------------------------------
// Utility functions

type fakeAddress struct {
	mino.Address
}
