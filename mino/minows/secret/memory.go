package secret

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.dedis.ch/dela/core/store/kv"
	"go.dedis.ch/dela/mino"
	"golang.org/x/xerrors"
	"sync"
)

// cacheStorage offers a layer of in-memory caching on top of a diskStorage.
type cacheStorage struct {
	sync.Mutex
	*diskStorage

	secret map[string]crypto.PrivKey
}

func newMemoryStore(db kv.DB, factory mino.AddressFactory) *cacheStorage {
	return &cacheStorage{
		diskStorage: newDiskStore(db, factory),
		secret:      make(map[string]crypto.PrivKey),
	}
}

func (s *cacheStorage) LoadOrCreate(name string) (crypto.PrivKey, error) {
	s.Lock()
	defer s.Unlock()

	secret, ok := s.secret[name]
	if ok {
		return secret, nil
	}

	secret, err := s.diskStorage.LoadOrCreate(name)
	if err != nil {
		return nil, xerrors.Errorf("could not load from disk: %v", err)
	}

	s.secret[name] = secret
	return secret, nil
}
