package controller

import (
	"crypto/rand"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/crypto/loader"
	"go.dedis.ch/dela/mino/minows"
	"golang.org/x/xerrors"
)

// implements node.Initializer
type controller struct{}

func (c controller) SetCommands(builder node.Builder) {
	// TODO implement me
	panic("implement me")
}

func (c controller) OnStart(flags cli.Flags, injector node.Injector) error {
	listen, err := ma.NewMultiaddr(flags.String("listen"))
	if err != nil {
		return xerrors.Errorf("could not parse listen addr: %v", err)
	}
	public, err := ma.NewMultiaddr(flags.String("public"))
	if err != nil {
		return xerrors.Errorf("could not parse public addr: %v", err)
	}
	key, err := loadPrivKey(filepath.Join(flags.Path("config"), "priv.key"))
	if err != nil {
		return err
	}
	// start mino instance
	m, err := minows.NewMinows(listen, public, key)
	if err != nil {
		return xerrors.Errorf("could not create mino: %v", err)
	}
	// inject as dependency
	injector.Inject(m)

	return nil
}

func (c controller) OnStop(injector node.Injector) error {
	// TODO close any passed contexts, reset any ongoing streams
	// TODO shut down host

	// TODO implement me
	panic("implement me")
}

func loadPrivKey(path string) (crypto.PrivKey, error) {
	keyLoader := loader.NewFileLoader(path)
	bytes, err := keyLoader.LoadOrCreate(newGenerator())
	if err != nil {
		return nil, xerrors.Errorf("could not load key: %v", err)
	}

	private, err := crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return nil, xerrors.Errorf("could not unmarshal key: %v", err)
	}

	return private, nil
}

// TODO simplify with Copilot: anonymous class equivalent?
type generator struct{}

func newGenerator() loader.Generator {
	return generator{}
}

func (g generator) Generate() ([]byte, error) {
	private, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, xerrors.Errorf("could not generate keys: %v", err)
	}

	bytes, err := crypto.MarshalPrivateKey(private)
	if err != nil {
		return nil, xerrors.Errorf("could not marshal key: %v", err)
	}

	return bytes, nil
}
