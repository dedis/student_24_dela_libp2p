package secret

import (
	"crypto/rand"
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.dedis.ch/dela/core/store/kv"
	"golang.org/x/xerrors"
)

// diskStorage provides persistent storage for secrets on disk.
type diskStorage struct {
	bucket []byte
	db     kv.DB
}

func newDiskStore(db kv.DB) *diskStorage {
	return &diskStorage{
		bucket: []byte("secrets"),
		db:     db,
	}
}

func (s *diskStorage) LoadOrCreate(name string) (crypto.PrivKey, error) {
	key := []byte(name)
	var buffer []byte
	err := s.db.Update(func(tx kv.WritableTx) error {
		bucket, err := tx.GetBucketOrCreate(s.bucket)
		if err != nil {
			return xerrors.Errorf("could not get bucket: %v", err)
		}

		stored := bucket.Get(key)
		if stored != nil {
			copy(buffer, stored)
			return nil
		}

		generated, err := generateSecret()
		if err != nil {
			return xerrors.Errorf("could not generate secret: %v", err)
		}
		err = bucket.Set(key, generated)
		if err != nil {
			return xerrors.Errorf("could not store secret: %v", err)
		}
		copy(buffer, generated)
		return nil
	})
	if err != nil {
		return nil, xerrors.Errorf("could not update db: %v", err)
	}

	secret, err := crypto.UnmarshalPrivateKey(buffer)
	if err != nil {
		return nil, xerrors.Errorf("could not unmarshal secret: %v", err)
	}
	return secret, nil
}

func generateSecret() ([]byte, error) {
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
