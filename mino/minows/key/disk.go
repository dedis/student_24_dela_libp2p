package key

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.dedis.ch/dela/core/store/kv"
	"golang.org/x/xerrors"
)

// diskStorage provides persistent storage for private keys on disk.
type diskStorage struct {
	bucket []byte
	db     kv.DB
}

func newDiskStore(db kv.DB) *diskStorage {
	return &diskStorage{
		bucket: []byte("keys"),
		db:     db,
	}
}

func (s *diskStorage) LoadOrCreate() (crypto.PrivKey, error) {
	key := []byte("private_key")
	var buffer []byte
	err := s.db.Update(func(tx kv.WritableTx) error {
		bucket, err := tx.GetBucketOrCreate(s.bucket)
		if err != nil {
			return xerrors.Errorf("could not get bucket: %v", err)
		}
		stored := bucket.Get(key)
		if stored != nil {
			buffer = make([]byte, len(stored))
			copy(buffer, stored)
			return nil
		}

		private, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return xerrors.Errorf("could not generate key: %v", err)
		}
		generated, err := crypto.MarshalPrivateKey(private)
		if err != nil {
			return xerrors.Errorf("could not marshal key: %v", err)
		}

		err = bucket.Set(key, generated)
		if err != nil {
			return xerrors.Errorf("could not store key: %v", err)
		}
		buffer = make([]byte, len(generated))
		copy(buffer, generated)
		return nil
	})
	if err != nil {
		return nil, xerrors.Errorf("could not update db: %v", err)
	}

	private, err := crypto.UnmarshalPrivateKey(buffer)
	if err != nil {
		return nil, xerrors.Errorf("could not unmarshal key: %v", err)
	}
	return private, nil
}
