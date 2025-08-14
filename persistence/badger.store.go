package persistence

import (
	"encoding/json"
	"fmt"

	"github.com/dgraph-io/badger/v4"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

type BadgerStore struct {
	db     *badger.DB
	logger logger.Logger
}

// badgerLoggerAdapter adapts our logger to Badger's logger interface.
type badgerLoggerAdapter struct {
	l logger.Logger
}

func (b badgerLoggerAdapter) Errorf(msg string, args ...interface{}) {
	b.l.Error(fmt.Sprintf(msg, args...))
}
func (b badgerLoggerAdapter) Warningf(msg string, args ...interface{}) {
	b.l.Warn(fmt.Sprintf(msg, args...))
}
func (b badgerLoggerAdapter) Infof(msg string, args ...interface{}) {
	b.l.Info(fmt.Sprintf(msg, args...))
}
func (b badgerLoggerAdapter) Debugf(msg string, args ...interface{}) {
	b.l.Debug(fmt.Sprintf(msg, args...))
}

func NewBadgerStore(path string, log logger.Logger) (*BadgerStore, error) {
	opts := badger.DefaultOptions(path)
	opts.Logger = badgerLoggerAdapter{l: log}
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	return &BadgerStore{db: db, logger: log}, nil
}

func (b *BadgerStore) Put(vector types.Vector) error {
	payload, err := json.Marshal(vector.Embedding)
	if err != nil {
		b.logger.Error("Failed to marshal vector embedding: {}")
	}

	err = b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(b.GetVectorKey(vector.ID)), payload)
	})
	if err != nil {
		b.logger.Error("Failed to update vector: {}")
	}
	return err
}

func (b *BadgerStore) Get(id string) (types.Vector, error) {
	var vector types.Vector
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(b.GetVectorKey(id)))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &vector.Embedding)
		})
	})
	return vector, err
}

func (b *BadgerStore) GetAllVectors() ([]types.Vector, error) {
	var vectors []types.Vector
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			var vector types.Vector
			if err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &vector.Embedding)
			}); err != nil {
				return err
			}
			vectors = append(vectors, vector)
		}
		return nil
	})
	return vectors, err
}

func (b *BadgerStore) GetIndexKey(id string) string {
	return fmt.Sprintf("i:%s", id)
}
func (b *BadgerStore) GetMetadataKey(id string) string {
	return fmt.Sprintf("m:%s", id)
}
func (b *BadgerStore) GetVectorKey(id string) string {
	return fmt.Sprintf("v:%s", id)
}

func (b *BadgerStore) Close() error {
	return b.db.Close()
}
