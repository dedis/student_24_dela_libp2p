// Theis file contains the network message handler implementations for the
// collective signing reactor and the module rpc.
//
// Documentation Last Review: 12.10.2020
//

package cosipbft

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"go.dedis.ch/dela/core"
	"go.dedis.ch/dela/core/access"
	"go.dedis.ch/dela/core/ordering/cosipbft/authority"
	"go.dedis.ch/dela/core/ordering/cosipbft/blockstore"
	"go.dedis.ch/dela/core/ordering/cosipbft/blocksync"
	"go.dedis.ch/dela/core/ordering/cosipbft/contracts/viewchange"
	"go.dedis.ch/dela/core/ordering/cosipbft/fastsync"
	"go.dedis.ch/dela/core/ordering/cosipbft/pbft"
	"go.dedis.ch/dela/core/ordering/cosipbft/types"
	"go.dedis.ch/dela/core/store"
	"go.dedis.ch/dela/core/store/hashtree"
	"go.dedis.ch/dela/core/txn/pool"
	"go.dedis.ch/dela/crypto"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/dela/serde"
	"go.dedis.ch/dela/serde/json"
	"golang.org/x/xerrors"
)

type syncMethodType bool

const (
	syncMethodFast  syncMethodType = false
	syncMethodBlock syncMethodType = true
)

// Processor processes the messages to run a collective signing PBFT consensus.
//
// - implements cosi.Reactor
// - implements mino.Handler
type processor struct {
	mino.UnsupportedHandler
	types.MessageFactory

	logger      zerolog.Logger
	pbftsm      pbft.StateMachine
	bsync       blocksync.Synchronizer
	fsync       fastsync.Synchronizer
	tree        blockstore.TreeCache
	pool        pool.Pool
	watcher     core.Observable
	rosterFac   authority.Factory
	hashFactory crypto.HashFactory
	access      access.Service

	context serde.Context
	genesis blockstore.GenesisStore
	blocks  blockstore.BlockStore
	// catchup sends catchup requests to the players to get new blocks
	catchup chan mino.Players

	started chan struct{}
}

func newProcessor() *processor {
	proc := &processor{
		watcher: core.NewWatcher(),
		context: json.NewContext(),
		started: make(chan struct{}),
		catchup: make(chan mino.Players),
	}
	go proc.catchupHandler()
	return proc
}

func (h *processor) syncMethod() syncMethodType {
	if h.bsync != nil {
		return syncMethodBlock
	}
	return syncMethodFast
}

// Invoke implements cosi.Reactor. It processes the messages from the collective
// signature module. The messages are either from the prepare or the commit
// phase.
func (h *processor) Invoke(from mino.Address, msg serde.Message) ([]byte, error) {
	switch in := msg.(type) {
	case types.BlockMessage:
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if h.syncMethod() == syncMethodFast {
			// Check if a catchup is needed for fastsync
			var roster mino.Players
			if h.blocks.Len() == 0 && in.GetBlock().GetIndex() > 0 {
				h.logger.Info().Msgf("node joined an existing blockchain from %+v", from)
				roster = mino.NewAddresses(from)
			} else if in.GetBlock().GetIndex() > h.blocks.Len() {
				h.logger.Warn().Msgf("node got asked to sign block-index %d, "+
					"but has only %d blocks", in.GetBlock().GetIndex(),
					h.blocks.Len())
				var err error
				roster, err = h.getCurrentRoster()
				if err != nil {
					return nil, xerrors.Errorf("failed to get roster: %v", err)
				}
			}

			if roster != nil {
				h.catchup <- roster
				return nil, xerrors.Errorf("needed to catch up")
			}
		} else {
			blocks := h.blocks.Watch(ctx)

			// In case the node is falling behind the chain, it gives it a chance to
			// catch up before moving forward.
			latest := h.bsync.GetLatest()

			if latest > h.blocks.Len() {
				for link := range blocks {
					if link.GetBlock().GetIndex() >= latest {
						cancel()
					}
				}
			}
		}

		viewMsgs := in.GetViews()
		if len(viewMsgs) > 0 {
			h.logger.Debug().Int("num", len(viewMsgs)).Msg("process views")

			views := make([]pbft.View, 0, len(viewMsgs))
			for addr, view := range viewMsgs {
				param := pbft.ViewParam{
					From:   addr,
					ID:     view.GetID(),
					Leader: view.GetLeader(),
				}

				views = append(views, pbft.NewView(param, view.GetSignature()))
			}

			// Force a view change if enough views are provided in the situation
			// where the current node is falling behind the others.
			err := h.pbftsm.AcceptAll(views)
			if err != nil {
				return nil, xerrors.Errorf("accept all: %v", err)
			}
		}

		digest, err := h.pbftsm.Prepare(from, in.GetBlock())
		if err != nil {
			return nil, xerrors.Errorf("pbft prepare failed: %v", err)
		}

		return digest[:], nil
	case types.CommitMessage:
		err := h.pbftsm.Commit(in.GetID(), in.GetSignature())
		if err != nil {
			h.logger.Debug().Msg("commit failed")

			return nil, xerrors.Errorf("pbft commit failed: %v", err)
		}

		buffer, err := in.GetSignature().MarshalBinary()
		if err != nil {
			return nil, xerrors.Errorf("couldn't marshal signature: %v", err)
		}

		return buffer, nil
	default:
		return nil, xerrors.Errorf("unsupported message of type '%T'", msg)
	}
}

// Process implements mino.Handler. It processes the messages from the RPC.
func (h *processor) Process(req mino.Request) (serde.Message, error) {
	switch msg := req.Message.(type) {
	case types.GenesisMessage:
		if h.genesis.Exists() {
			return nil, nil
		}

		root := msg.GetGenesis().GetRoot()

		return nil, h.storeGenesis(msg.GetGenesis().GetRoster(), &root)
	case types.DoneMessage:
		if h.pbftsm.GetState() == pbft.InitialState {
			h.logger.Warn().Msgf("Got block without commit from %v - catching up", req.Address)
			h.catchup <- mino.NewAddresses(req.Address)
		} else {
			err := h.pbftsm.Finalize(msg.GetID(), msg.GetSignature())
			if err != nil {
				return nil, xerrors.Errorf("pbftsm finalized failed: %v", err)
			}
		}
	case types.ViewMessage:
		param := pbft.ViewParam{
			From:   req.Address,
			ID:     msg.GetID(),
			Leader: msg.GetLeader(),
		}

		err := h.pbftsm.Accept(pbft.NewView(param, msg.GetSignature()))
		if err != nil {
			h.logger.Warn().Err(err).Msg("view message refused")
		}
	default:
		return nil, xerrors.Errorf("unsupported message of type '%T'", req.Message)
	}

	return nil, nil
}

func (h *processor) getCurrentRoster() (authority.Authority, error) {
	return h.readRoster(h.tree.Get())
}

func (h *processor) readRoster(tree hashtree.Tree) (authority.Authority, error) {
	data, err := tree.Get(viewchange.GetRosterKey())
	if err != nil {
		return nil, xerrors.Errorf("read from tree: %v", err)
	}

	roster, err := h.rosterFac.AuthorityOf(h.context, data)
	if err != nil {
		return nil, xerrors.Errorf("decode failed: %v", err)
	}

	return roster, nil
}

func (h *processor) storeGenesis(roster authority.Authority, match *types.Digest) error {
	value, err := roster.Serialize(h.context)
	if err != nil {
		return xerrors.Errorf("failed to serialize roster: %v", err)
	}

	stageTree, err := h.tree.Get().Stage(func(snap store.Snapshot) error {
		err := h.makeAccess(snap, roster)
		if err != nil {
			return xerrors.Errorf("failed to set access: %v", err)
		}

		err = snap.Set(viewchange.GetRosterKey(), value)
		if err != nil {
			return xerrors.Errorf("failed to store roster: %v", err)
		}

		return nil
	})
	if err != nil {
		return xerrors.Errorf("while updating tree: %v", err)
	}

	root := types.Digest{}
	copy(root[:], stageTree.GetRoot())

	if match != nil && *match != root {
		return xerrors.Errorf("mismatch tree root '%v' != '%v'", match, root)
	}

	genesis, err := types.NewGenesis(roster, types.WithGenesisRoot(root))
	if err != nil {
		return xerrors.Errorf("creating genesis: %v", err)
	}

	err = stageTree.Commit()
	if err != nil {
		return xerrors.Errorf("tree commit failed: %v", err)
	}

	h.tree.Set(stageTree)

	err = h.genesis.Set(genesis)
	if err != nil {
		return xerrors.Errorf("set genesis failed: %v", err)
	}

	close(h.started)

	return nil
}

func (h *processor) makeAccess(store store.Snapshot, roster authority.Authority) error {
	creds := viewchange.NewCreds()

	iter := roster.PublicKeyIterator()
	for iter.HasNext() {
		// Grant each member of the roster an access to change the roster.
		err := h.access.Grant(store, creds, iter.GetNext())
		if err != nil {
			return err
		}
	}

	return nil
}

// catchupHandler listens to incoming requests for potentially missing blocks.
// It is started as a go-routine
func (h *processor) catchupHandler() {
	for players := range h.catchup {
		if h.syncMethod() == syncMethodFast {
			ctx, cancel := context.WithCancel(context.Background())
			for {
				err := h.fsync.Sync(ctx, players,
					fastsync.Config{SplitMessageSize: DefaultFastSyncMessageSize})
				if err == nil {
					break
				}
				h.logger.Err(err).Msg("Couldn't sync - trying again in 10 seconds")
				time.Sleep(10 * time.Second)
			}
			cancel()
		}
	}
	panic("catchup channel got closed - this should not happen")
}
