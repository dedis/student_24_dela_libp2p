package minows

import (
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/core/store/kv"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestController_OnStart(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("/dns4/p2p-1.c4dt.dela.org/tcp/443/wss")
	inj, clean := mustCreateInjector(t)
	defer clean()
	ctrl, stop := mustCreateController(t, inj)
	defer stop()

	err := ctrl.OnStart(flags, inj)
	require.NoError(t, err)
	var m *minows
	err = inj.Resolve(&m)
	require.NoError(t, err)
}

func TestController_OptionalPublic(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("")
	inj, clean := mustCreateInjector(t)
	defer clean()
	ctrl, stop := mustCreateController(t, inj)
	defer stop()

	err := ctrl.OnStart(flags, inj)
	require.NoError(t, err)
	var m *minows
	err = inj.Resolve(&m)
	require.NoError(t, err)
}

func TestController_InvalidListen(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("invalid")
	flags.On("String", "public").Return("/dns4/p2p-1.c4dt.dela.org/tcp/443/wss")
	inj, clean := mustCreateInjector(t)
	defer clean()
	ctrl, _ := mustCreateController(t, inj)

	err := ctrl.OnStart(flags, inj)
	require.Error(t, err)
}

func TestController_InvalidPublic(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("invalid")
	inj, clean := mustCreateInjector(t)
	defer clean()
	ctrl, _ := mustCreateController(t, inj)

	err := ctrl.OnStart(flags, inj)
	require.Error(t, err)
}

func TestController_OnStop(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("/dns4/p2p-1.c4dt.dela.org/tcp/443/wss")
	inj, clean := mustCreateInjector(t)
	defer clean()
	ctrl, _ := mustCreateController(t, inj)
	err := ctrl.OnStart(flags, inj)
	require.NoError(t, err)

	err = ctrl.OnStop(inj)
	require.NoError(t, err)
}

func TestController_MissingDependency(t *testing.T) {
	flags := new(mockFlags)
	flags.On("String", "listen").Return("/ip4/0.0.0.0/tcp/8000/ws")
	flags.On("String", "public").Return("/dns4/p2p-1.c4dt.dela.org/tcp/443/wss")
	inj, clean := mustCreateInjector(t)
	defer clean()
	ctrl, _ := mustCreateController(t, inj)
	err := ctrl.OnStart(flags, inj)
	require.NoError(t, err)

	err = ctrl.OnStop(node.NewInjector())
	require.Error(t, err)
}

func mustCreateInjector(t *testing.T) (node.Injector, func()) {
	dir, err := os.MkdirTemp(os.TempDir(), "test")
	require.NoError(t, err)
	db, err := kv.New(filepath.Join(dir, "test.db"))
	require.NoError(t, err)
	inj := node.NewInjector()
	inj.Inject(db)

	clean := func() {
		os.RemoveAll(dir)
	}

	return inj, clean
}

func mustCreateController(t *testing.T, inj node.Injector) (node.Initializer, func()) {

	ctrl := NewController()
	stop := func() {
		require.NoError(t, ctrl.OnStop(inj))
	}
	return ctrl, stop
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
