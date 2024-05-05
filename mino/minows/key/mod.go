package key

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.dedis.ch/dela/core/store/kv"
)

// Storage is an interface for storing and retrieving private keys.
type Storage interface {
	// LoadOrCreate loads the private key associated with a mino instance.
	// If the key does not exist, it will create a new one.
	LoadOrCreate() (crypto.PrivKey, error)
}

// NewStorage creates a new private key storage
// with caching for efficient access.
func NewStorage(db kv.DB) Storage {
	return newMemoryStore(db)
}
