package minows

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
)

const (
	ADDR2 = "/dns4/example.com/tcp/80"
	ADDR1 = "/ip4/0.0.0.0/tcp/80"
)

var (
	addr1     = address{multiaddr: newMultiAddr(ADDR1)}
	addr1Copy = address{multiaddr: newMultiAddr(ADDR1)}
	addr2     = address{multiaddr: newMultiAddr(ADDR2)}
)

func newMultiAddr(addr string) ma.Multiaddr {
	multiaddr, _ := ma.NewMultiaddr(addr)
	return multiaddr
}

func TestAddress_Equal(t *testing.T) {
	require.True(t, addr1.Equal(addr1))
	require.True(t, addr1.Equal(addr1Copy))
	require.False(t, addr1.Equal(addr2))
	// require.False(t, addr1.Equal(address{})) // TODO
	require.False(t, addr1.Equal(fakeAddress{}))
}

func TestAddress_String(t *testing.T) {
	require.Equal(t, ADDR1, addr1.String())
	require.Equal(t, ADDR2, addr2.String())
}

func TestAddress_MarshalText(t *testing.T) {
	bytes, err := addr1.MarshalText()
	require.NoError(t, err)
	require.Equal(t, ADDR1, string(bytes))

	bytes, err = addr1.MarshalText()
	require.NoError(t, err)
	require.Equal(t, ADDR1, string(bytes))
}

func TestAddressFactory_FromText(t *testing.T) {
	factory := addressFactory{}

	addr := factory.FromText([]byte(ADDR1))
	require.Equal(t, addr1, addr)

	addr = factory.FromText([]byte(ADDR2))
	require.Equal(t, addr2, addr)
}

// ------------------------------------------------------------------------------
// Utility functions

type fakeAddress struct {
	mino.Address
}
