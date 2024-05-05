package key

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.dedis.ch/dela/core/store/kv"
	"golang.org/x/xerrors"
	"sync"
)

// cachedStorage offers a layer of in-memory caching on top of a diskStorage.
type cachedStorage struct {
	sync.Mutex
	*diskStorage

	key crypto.PrivKey
}

func newMemoryStore(db kv.DB) *cachedStorage {
	return &cachedStorage{
		diskStorage: newDiskStore(db),
	}
}

func (s *cachedStorage) LoadOrCreate() (crypto.PrivKey, error) {
	s.Lock()
	defer s.Unlock()

	if s.key != nil {
		return s.key, nil
	}

	key, err := s.diskStorage.LoadOrCreate()
	if err != nil {
		return nil, xerrors.Errorf("could not load from disk: %v", err)
	}

	s.key = key
	return key, nil
}
