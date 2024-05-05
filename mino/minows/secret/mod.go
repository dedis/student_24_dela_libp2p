package secret

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.dedis.ch/dela/core/store/kv"
	"go.dedis.ch/dela/mino"
)

// Storage is an interface for storing and retrieving secrets.
type Storage interface {
	// LoadOrCreate loads the secret associated with the address. If the secret
	// does not exist, it will create a new one.
	LoadOrCreate(name string) (crypto.PrivKey, error)
}

// NewStorage creates a new secret storage with caching for efficient access.
func NewStorage(db kv.DB, factory mino.AddressFactory) Storage {
	return newMemoryStore(db, factory)
}
