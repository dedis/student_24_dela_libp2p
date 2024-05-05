package minows

import (
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/core/store/kv"
	"go.dedis.ch/dela/mino/minows/key"
	"golang.org/x/xerrors"
)

// controller
// - implements node.Initializer
type controller struct{}

// NewController creates a CLI app to start a Minows instance.
func NewController() node.Initializer {
	return controller{}
}

const flagListen = "listen"
const flagPublic = "public"

func (c controller) SetCommands(builder node.Builder) {
	builder.SetStartFlags(
		cli.StringFlag{
			Name: flagListen,
			Usage: "set the address to listen on (default all interfaces, " +
				"random port)",
			Required: false,
			Value:    "/ip4/0.0.0.0/tcp/0", // todo add test
		},
		cli.StringFlag{
			Name: flagPublic,
			Usage: "set the publicly reachable address (" +
				"default listen address)",
			Required: false, // todo add test
			Value:    "",
		},
	)
}

func (c controller) OnStart(flags cli.Flags, inj node.Injector) error {
	listen, err := ma.NewMultiaddr(flags.String(flagListen))
	if err != nil {
		return xerrors.Errorf("could not parse listen addr: %v", err)
	}

	var db kv.DB
	err = inj.Resolve(&db)
	if err != nil {
		return xerrors.Errorf("could not resolve db: %v", err)
	}
	storage := key.NewStorage(db)
	key, err := storage.LoadOrCreate()
	if err != nil {
		return xerrors.Errorf("could not load key: %v", err)
	}

	var public ma.Multiaddr
	p := flags.String(flagPublic)
	if p != "" {
		public, err = ma.NewMultiaddr(p)
		if err != nil {
			return xerrors.Errorf("could not parse public addr: %v", err)
		}
	}

	m, err := NewMinows(listen, public, key)
	if err != nil {
		return xerrors.Errorf("could not start mino: %v", err)
	}
	inj.Inject(m)
	return nil
}

func (c controller) OnStop(inj node.Injector) error {
	var m *minows
	err := inj.Resolve(&m)
	if err != nil {
		return xerrors.Errorf("could not resolve mino: %v", err)
	}
	err = m.stop()
	if err != nil {
		return xerrors.Errorf("could not stop mino: %v", err)
	}
	return nil
}
