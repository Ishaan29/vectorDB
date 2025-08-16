package persistence

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/ishaan29/vectorDB/internal/logger"
	"github.com/ishaan29/vectorDB/pkg/types"
)

type BadgerStore struct {
	db     *badger.DB
	logger logger.Logger
}

const batchSize = 100

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

	opts.ValueLogFileSize = 256 << 20 // 256 mb
	opts.MemTableSize = 64 << 20      // 64 mb
	opts.NumMemtables = 2
	opts.NumLevelZeroTables = 2

	opts.Logger = badgerLoggerAdapter{l: log}

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("Failed to open badger db: %w", err)
	}

	go runGC(db, log)

	return &BadgerStore{
		db:     db,
		logger: log,
	}, nil
}

func runGC(db *badger.DB, log logger.Logger) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		lsm, vlog := db.Size()
		if vlog > 1<<32 {
			err := db.RunValueLogGC(0.5)
			if err != nil && err != badger.ErrNoRewrite {
				log.Warn("Value log GC error ", logger.Error("error", err))
			}
		}
		if lsm > 1<<29 {
			err := db.Flatten(2)
			if err != nil {
				log.Warn("LSM flatten error ", logger.Error("error", err))
			}
		}
	}
}

func (bs *BadgerStore) Put(vector types.Vector) error {
	data, err := json.Marshal(vector)
	if err != nil {
		return ErrBadgerMarshal
	}
	return bs.db.Update(func(txn *badger.Txn) error {
		key := []byte(vector.ID)
		return txn.Set(key, data)
	})
}

func (bs *BadgerStore) Get(id string) (types.Vector, error) {
	var vector types.Vector
	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(id))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &vector)
		})
	})
	if err == badger.ErrKeyNotFound {
		return vector, ErrBadgerKeyNotFound(id)
	}
	return vector, err
}

func (bs *BadgerStore) Delete(id string) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(id))
	})
}

func (bs *BadgerStore) BatchPut(vectors []types.Vector) error {

	for i := 0; i < len(vectors); i += batchSize {
		end := i + batchSize
		if end > len(vectors) {
			end = len(vectors)
		}
		batch := vectors[i:end]

		err := bs.db.Update(func(txn *badger.Txn) error {
			for _, vector := range batch {
				data, err := json.Marshal(vector)
				if err != nil {
					return ErrBadgerBatchMarshal(vector.ID, err)
				}

				if err := txn.Set([]byte(vector.ID), data); err != nil {
					return ErrBadgerBatchSet(vector.ID, err)
				}
			}
			return nil
		})
		if err != nil {
			return ErrBadgerBatchWriteFailed(i, err)
		}

		if bs.logger != nil {
			bs.logger.Debug("Batch written",
				logger.Int("start", i),
				logger.Int("count", len(batch)))
		}
	}
	return nil
}

func (bs *BadgerStore) Iterate(fn func(types.Vector) error) error {
	return bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = true
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()

			var vector types.Vector
			err := item.Value(func(val []byte) error {
				return json.Unmarshal(val, &vector)
			})

			if err != nil {
				if bs.logger != nil {
					bs.logger.Warn("Failed to unmarshal vector",
						logger.String("key", string(item.Key())),
						logger.Error("error", err))
				}
				continue // Skip corrupted entries
			}

			if err := fn(vector); err != nil {
				return err
			}
		}
		return nil
	})
}

func (bs *BadgerStore) Close() error {
	return bs.db.Close()
}

func (bs *BadgerStore) Stats() map[string]interface{} {
	lsm, vlog := bs.db.Size()
	return map[string]interface{}{
		"lsm_size_bytes":  lsm,
		"vlog_size_bytes": vlog,
	}
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
