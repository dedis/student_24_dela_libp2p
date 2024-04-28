package minows

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/testing/fake"
	"testing"
)

// todo make local to test functions
const (
	AddrWS  = "/ip4/127.0.0.1/tcp/80/ws"
	AddrWSS = "/ip4/127.0.0.1/tcp/443/wss"
)

func Test_newMinows(t *testing.T) {
	type args struct {
		listen  multiaddr.Multiaddr
		public  multiaddr.Multiaddr
		privKey crypto.PrivKey
	}
	var tests = map[string]struct {
		args args
	}{
		// 'public' only uses localhost a for local testing
		"ws": {
			args: args{
				listen:  mustCreateMultiaddress(AddrAllInterface),
				public:  mustCreateMultiaddress(AddrWS),
				privKey: mustCreateSecret(),
			},
		},
		"wss": {
			args: args{
				listen:  mustCreateMultiaddress(AddrAllInterface),
				public:  mustCreateMultiaddress(AddrWSS),
				privKey: mustCreateSecret(),
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := newMinows(tt.args.listen, tt.args.public, tt.args.privKey)
			require.NoError(t, err)
			require.NotNil(t, got)
			require.IsType(t, &minows{}, got)
			got.close() // clean up todo assert no error
		})
	}
}

func Test_minows_GetAddressFactory(t *testing.T) {
	secret := mustCreateSecret()
	tests := map[string]struct {
		m *minows
	}{
		"ws":  {mustCreateMinows(AddrAllInterface, AddrWS, secret)},
		"wss": {mustCreateMinows(AddrAllInterface, AddrWSS, secret)},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer tt.m.close()

			got := tt.m.GetAddressFactory()
			require.NotNil(t, got)
			require.IsType(t, addressFactory{}, got)
		})
	}
}

func Test_minows_GetAddress(t *testing.T) {
	secret := mustCreateSecret()            // todo feed random seed
	id := mustDerivePeerID(secret).String() // todo hardcode expected string
	tests := map[string]struct {
		m    *minows
		want address
	}{
		"ws": {mustCreateMinows(AddrAllInterface, AddrWS, secret),
			mustCreateAddress(AddrWS, id)},
		"wss": {mustCreateMinows(AddrAllInterface, AddrWSS, secret),
			mustCreateAddress(AddrWSS, id)},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defer tt.m.close()

			got := tt.m.GetAddress()
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_minows_WithSegment_Empty(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret())
	defer m.close() // todo assert no error

	got := m.WithSegment("")
	require.Equal(t, m, got)
}

func Test_minows_WithSegment(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret())
	defer m.close()

	got := m.WithSegment("test")
	require.NotEqual(t, m, got)

	got2 := m.WithSegment("test").WithSegment("test")
	require.NotEqual(t, m, got2)
	require.NotEqual(t, got, got2)
}

func Test_minows_CreateRPC_InvalidName(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret())
	defer m.close()

	_, err := m.CreateRPC("invalid name", nil, nil)
	require.Error(t, err)
}

func Test_minows_CreateRPC_AlreadyExists(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret())
	defer m.close()

	_, err := m.CreateRPC("test", nil, nil)
	require.NoError(t, err)
	_, err = m.CreateRPC("test", nil, nil)
	require.Error(t, err)
}

func Test_minows_CreateRPC_InvalidSegment(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret()).
		WithSegment("invalid segment").(*minows)
	defer m.close()

	_, err := m.CreateRPC("test", nil, nil)
	require.Error(t, err)
}

func Test_minows_CreateRPC(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret())
	defer m.close()

	r1, err := m.CreateRPC("test", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r1)
	r2, err := m.CreateRPC("Test", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r2)

	m = m.WithSegment("segment").(*minows)
	r3, err := m.CreateRPC("test", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r3)
	r4, err := m.CreateRPC("Test", nil, nil)
	require.NoError(t, err)
	require.NotNil(t, r4)
}

func mustCreateRPC(m *minows, name string, h mino.Handler) mino.RPC {
	r, err := m.CreateRPC(name, h, fake.MessageFactory{}) // registers handler
	if err != nil {
		panic(err)
	}
	return r
}

// todo return tearDown func: m.close()
func mustCreateMinows(listen string, public string, secret crypto.PrivKey) *minows {
	m, err := newMinows(mustCreateMultiaddress(listen),
		mustCreateMultiaddress(public), secret) // starts listening
	if err != nil {
		panic(err)
	}
	return m
}

// todo fix randomness to assert a known ID matches a fixed key
func mustCreateSecret() crypto.PrivKey {
	secret, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}
	return secret
}

func mustDerivePeerID(secret crypto.PrivKey) peer.ID {
	pid, err := peer.IDFromPrivateKey(secret)
	if err != nil {
		panic(err)
	}
	return pid
}
