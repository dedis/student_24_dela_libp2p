package minows

import (
	"crypto/rand"
	"path/filepath"

	"github.com/libp2p/go-libp2p/core/crypto"
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/crypto/loader"
	"golang.org/x/xerrors"
)

// controller implements node.Initializer
type controller struct{}

func NewController() node.Initializer {
	return controller{}
}

func (c controller) SetCommands(builder node.Builder) {
	builder.SetStartFlags(
		cli.StringFlag{
			Name:     "listen",
			Usage:    "set the address to listen on",
			Required: true,
			Value:    "/ip4/0.0.0.0/tcp/0", // default all interfaces,
			// random port, any protocol
		},
		// todo need public?
		// todo where is 'config' set?
	)
}

func (c controller) OnStart(flags cli.Flags, injector node.Injector) error {
	listen, err := ma.NewMultiaddr(flags.String("listen"))
	if err != nil {
		return xerrors.Errorf("could not parse listen addr: %w", err)
	}
	// todo 'public' is not used? remove?
	public, err := ma.NewMultiaddr(flags.String("public"))
	if err != nil {
		return xerrors.Errorf("could not parse public addr: %w", err)
	}
	secret, err := loadSecret(filepath.Join(flags.Path("config"), "priv.key"))
	if err != nil {
		return err
	}
	m, err := newMinows(listen, public, secret)
	if err != nil {
		return xerrors.Errorf("could not start mino: %w", err)
	}
	injector.Inject(m)
	return nil
}

func (c controller) OnStop(injector node.Injector) error {
	var m *minows
	err := injector.Resolve(m)
	if err != nil {
		return xerrors.Errorf("could not resolve mino: %w", err)
	}
	err = m.stop()
	if err != nil {
		return xerrors.Errorf("could not stop mino: %w", err)
	}
	return nil
}

func loadSecret(path string) (crypto.PrivKey, error) {
	// TODO use DiskStore instead
	keyLoader := loader.NewFileLoader(path)
	bytes, err := keyLoader.LoadOrCreate(newGenerator())
	if err != nil {
		return nil, xerrors.Errorf("could not load key: %w", err)
	}
	private, err := crypto.UnmarshalPrivateKey(bytes)
	if err != nil {
		return nil, xerrors.Errorf("could not unmarshal key: %w", err)
	}
	return private, nil
}

type generator struct{}

func (g generator) Generate() ([]byte, error) {
	private, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, xerrors.Errorf("could not generate keys: %w", err)
	}
	bytes, err := crypto.MarshalPrivateKey(private)
	if err != nil {
		return nil, xerrors.Errorf("could not marshal key: %w", err)
	}
	return bytes, nil
}

func newGenerator() loader.Generator {
	return generator{}
}
