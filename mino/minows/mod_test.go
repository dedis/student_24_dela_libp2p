package minows

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	AddrWS  = "/ip4/127.0.0.1/tcp/80/ws"
	AddrWSS = "/ip4/127.0.0.1/tcp/443/wss"
)

func TestNewMinows(t *testing.T) {
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
			got, err := NewMinows(tt.args.listen, tt.args.public, tt.args.privKey)
			require.NoError(t, err)
			require.NotNil(t, got)
			got.close() // clean up
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
	secret := mustCreateSecret()
	pid := mustDerivePeerID(secret)
	tests := map[string]struct {
		m    *minows
		want address
	}{
		"ws": {mustCreateMinows(AddrAllInterface, AddrWS, secret),
			address{mustCreateMultiaddress(AddrWS), pid}},
		"wss": {mustCreateMinows(AddrAllInterface, AddrWSS, secret),
			address{mustCreateMultiaddress(AddrWSS), pid}},
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
	defer m.close()

	got := m.WithSegment("")
	require.Equal(t, m, got)
}

func Test_minows_WithSegment(t *testing.T) {
	m := mustCreateMinows(AddrAllInterface, AddrWS, mustCreateSecret())
	defer m.close()

	got := m.WithSegment("test")
	require.NotEqual(t, m, got)
	require.Equal(t, []string{"test"}, got.(*minows).namespace)
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

func mustCreateMinows(listen string, public string, secret crypto.PrivKey) *minows {
	m, err := NewMinows(mustCreateMultiaddress(listen),
		mustCreateMultiaddress(public), secret)
	if err != nil {
		panic(err)
	}
	return m
}

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
