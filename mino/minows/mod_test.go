package minows

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_newMinows(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000/ws"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	const publicWSS = "/ip4/127.0.0.1/tcp/443/wss"
	type args struct {
		listen string
		public string
	}
	var tests = map[string]struct {
		args args
	}{
		"ws": {
			args: args{
				listen: listen,
				public: publicWS,
			},
		},
		"wss": {
			args: args{
				listen: listen,
				public: publicWSS,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			listen := mustCreateMultiaddress(t, tt.args.listen)
			public := mustCreateMultiaddress(t, tt.args.public)
			secret := mustCreateSecret(t)

			m, err := newMinows(listen, public, secret)
			require.NoError(t, err)
			require.NotNil(t, m)
			require.IsType(t, &minows{}, m)
			require.NoError(t, m.stop())
		})
	}
}

func Test_minows_GetAddressFactory(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	const publicWSS = "/ip4/127.0.0.1/tcp/443/wss"
	type m struct {
		listen string
		public string
	}
	tests := map[string]struct {
		m m
	}{
		"ws":  {m{listen, publicWS}},
		"wss": {m{listen, publicWSS}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m, stop := mustCreateMinows(t, tt.m.listen, tt.m.public)
			defer stop()

			factory := m.GetAddressFactory()
			require.NotNil(t, factory)
			require.IsType(t, addressFactory{}, factory)
		})
	}
}

func Test_minows_GetAddress(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/80"
	const publicWS = "/ip4/127.0.0.1/tcp/80/ws"
	const publicWSS = "/ip4/127.0.0.1/tcp/443/wss"
	secret := mustCreateSecret(t)
	id := mustDerivePeerID(t, secret).String()
	type m struct {
		listen string
		public string
		secret crypto.PrivKey
	}
	type want struct {
		location string
		identity string
	}
	tests := map[string]struct {
		m    m
		want want
	}{
		"ws":  {m{listen, publicWS, secret}, want{publicWS, id}},
		"wss": {m{listen, publicWSS, secret}, want{publicWSS, id}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			m, err := newMinows(mustCreateMultiaddress(t, tt.m.listen),
				mustCreateMultiaddress(t, tt.m.public), tt.m.secret)
			require.NoError(t, err)
			defer require.NoError(t, m.stop())
			want := mustCreateAddress(t, tt.want.location, tt.want.identity)

			got := m.GetAddress()
			require.Equal(t, want, got)
		})
	}
}

func Test_minows_WithSegment_Empty(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	m, stop := mustCreateMinows(t, listen, publicWS)
	defer stop()

	got := m.WithSegment("")
	require.Equal(t, m, got)
}

func Test_minows_WithSegment(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	m, stop := mustCreateMinows(t, listen, publicWS)
	defer stop()

	got := m.WithSegment("test")
	require.NotEqual(t, m, got)

	got2 := m.WithSegment("test").WithSegment("test")
	require.NotEqual(t, m, got2)
	require.NotEqual(t, got, got2)
}

func Test_minows_CreateRPC_InvalidName(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	m, stop := mustCreateMinows(t, listen, publicWS)
	defer stop()

	_, err := m.CreateRPC("invalid name", nil, nil)
	require.Error(t, err)
}

func Test_minows_CreateRPC_AlreadyExists(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	m, stop := mustCreateMinows(t, listen, publicWS)
	defer stop()

	_, err := m.CreateRPC("test", nil, nil)
	require.NoError(t, err)
	_, err = m.CreateRPC("test", nil, nil)
	require.Error(t, err)
}

func Test_minows_CreateRPC_InvalidSegment(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	m, stop := mustCreateMinows(t, listen, publicWS)
	defer stop()
	m = m.WithSegment("invalid segment").(*minows)

	_, err := m.CreateRPC("test", nil, nil)
	require.Error(t, err)
}

func Test_minows_CreateRPC(t *testing.T) {
	const listen = "/ip4/0.0.0.0/tcp/6000"
	const publicWS = "/ip4/127.0.0.1/tcp/6000/ws"
	m, stop := mustCreateMinows(t, listen, publicWS)
	defer stop()

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

func mustCreateMinows(t *testing.T, listen string, public string) (*minows, func()) {
	secret := mustCreateSecret(t)
	m, err := newMinows(mustCreateMultiaddress(t, listen),
		mustCreateMultiaddress(t, public), secret) // starts listening
	require.NoError(t, err)
	stop := func() { require.NoError(t, m.stop()) }
	return m, stop
}

func mustCreateSecret(t *testing.T) crypto.PrivKey {
	secret, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	return secret
}

func mustDerivePeerID(t *testing.T, secret crypto.PrivKey) peer.ID {
	pid, err := peer.IDFromPrivateKey(secret)
	require.NoError(t, err)
	return pid
}
