package minows

import (
	ma "github.com/multiformats/go-multiaddr"
	"go.dedis.ch/dela/cli"
	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/core/store/kv"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/mino/minows/secret"
	"golang.org/x/xerrors"
)

// controller
// - implements node.Initializer
type controller struct{}

// NewController creates a CLI app to start a Minows instance.
func NewController() node.Initializer {
	return controller{}
}

func (c controller) SetCommands(builder node.Builder) {
	builder.SetStartFlags(
		cli.StringFlag{
			Name: "listen",
			Usage: "set the address to listen on (default all interfaces, " +
				"random port)",
			Required: false,
			Value:    "/ip4/0.0.0.0/tcp/0",
		},
		cli.StringFlag{
			Name: "public",
			Usage: "set the publicly reachable address (" +
				"default listen address)",
			Required: false,
			Value:    "",
		},
		cli.StringFlag{
			Name:     "name",
			Usage:    "used to fetch the secret in db (e.g. node-1)",
			Required: true,
		},
	)
}

func (c controller) OnStart(flags cli.Flags, inj node.Injector) error {
	listen, err := ma.NewMultiaddr(flags.String("listen"))
	if err != nil {
		return xerrors.Errorf("could not parse listen addr: %v", err)
	}

	var db kv.DB
	err = inj.Resolve(&db)
	if err != nil {
		return xerrors.Errorf("could not resolve db: %v", err)
	}
	storage := secret.NewStorage(db, addressFactory{})
	secret, err := storage.LoadOrCreate(flags.String("name"))
	if err != nil {
		return xerrors.Errorf("could not load secret: %v", err)
	}

	var m mino.Mino
	var e error
	p := flags.String("public")
	if p == "" {
		m, e = newMinowsLocal(listen, secret)
	} else {
		public, err := ma.NewMultiaddr(p)
		if err != nil {
			return xerrors.Errorf("could not parse public addr: %v", err)
		}
		m, e = newMinows(listen, public, secret)
	}
	if e != nil {
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
