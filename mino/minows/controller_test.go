package minows

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/cli/node"
	"testing"
	"time"
)

func TestController_OnStart(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("/dns4/p2p-1.c4dt.dela.org/tcp/443/wss")
	flags.On("Path", "config").Return("/tmp")

	ctrl := NewController()
	inj := node.NewInjector()
	err := ctrl.OnStart(flags, inj)
	require.NoError(t, err)

	var m *minows
	err = inj.Resolve(&m)
	require.NoError(t, err)
	err = m.stop()
	require.NoError(t, err)
}

func TestController_OnStop(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("/dns4/p2p-1.c4dt.dela.org/tcp/443/wss")
	flags.On("Path", "config").Return("/tmp")

	ctrl := NewController()
	inj := node.NewInjector()
	err := ctrl.OnStart(flags, inj)
	require.NoError(t, err)

	err = ctrl.OnStop(inj)
	require.NoError(t, err)
}

// mockFlags
// - implements cli.Flags
type mockFlags struct {
	mock.Mock
}

func (m *mockFlags) String(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *mockFlags) StringSlice(name string) []string {
	args := m.Called(name)
	return args.Get(0).([]string)
}

func (m *mockFlags) Duration(name string) time.Duration {
	args := m.Called(name)
	return args.Get(0).(time.Duration)
}

func (m *mockFlags) Path(name string) string {
	args := m.Called(name)
	return args.String(0)
}

func (m *mockFlags) Int(name string) int {
	args := m.Called(name)
	return args.Int(0)
}

func (m *mockFlags) Bool(name string) bool {
	args := m.Called(name)
	return args.Bool(0)
}
