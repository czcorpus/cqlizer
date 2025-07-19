package index

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type SearchResult struct {
	Value string `json:"value"`
	Freq  uint32 `json:"freq"`
}

// DB is a wrapper around badger.DB providing concrete
// methods for adding/retrieving collocation information.
type DB struct {
	bdb *badger.DB
}

// Close closes the internal Badger database.
// It is necessary to perform the close especially
// in cases of data writing.
// It is possible to call the method on nil instance
// or on an uninitialized DB object, in which case
// it is a NOP.
func (db *DB) Close() error {
	if db != nil && db.bdb != nil {
		return db.bdb.Close()
	}
	return nil
}

func (db *DB) Flush() error {
	return db.bdb.DropAll()
}

func (db *DB) Size() (int64, int64) {
	return db.bdb.Size()
}

func (db *DB) StoreTimestamp(key string, value time.Time) error {
	keyBytes := make([]byte, 1+len(key))
	keyBytes[0] = AuxDataPrefix
	copy(keyBytes[1:], []byte(key))

	valueBytes := encodeTime(value)

	return db.bdb.Update(func(txn *badger.Txn) error {
		return txn.Set(keyBytes, valueBytes)
	})
}

func (db *DB) ReadTimestamp(key string) (time.Time, error) {
	keyBytes := make([]byte, 1+len(key))
	keyBytes[0] = AuxDataPrefix
	copy(keyBytes[1:], []byte(key))

	var result time.Time
	err := db.bdb.View(func(txn *badger.Txn) error {
		item, err := txn.Get(keyBytes)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			t, decodeErr := decodeTime(val)
			if decodeErr != nil {
				return decodeErr
			}
			result = t
			return nil
		})
	})

	return result, err
}

func (db *DB) StoreQueryTx(txn *badger.Txn, query string, freq uint32) error {
	key := encodeOriginalQuery(query)
	var currFreq uint32

	bCurrFreq, err := txn.Get(key)
	if err != nil && err != badger.ErrKeyNotFound {
		return fmt.Errorf("failed to store query: %w", err)

	} else if err == nil {
		err = bCurrFreq.Value(func(val []byte) error {
			currFreq = decodeFrequency(val)
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to fetch value from index: %w", err)
		}
	}

	bFreq := encodeFrequency(freq + currFreq)
	if err := txn.Set(key, bFreq); err != nil {
		return fmt.Errorf("failed to store query into index: %w", err)
	}
	return nil
}

func (db *DB) Update(fn func(txn *badger.Txn) error) error {
	return db.bdb.Update(fn)
}

func (db *DB) SearchByPrefix(cqlPrefix string, limit int) ([]SearchResult, error) {
	ans := make([]SearchResult, 0, 8)
	err := db.bdb.View(func(txn *badger.Txn) error {
		key := encodeOriginalQuery(cqlPrefix)
		opts := badger.DefaultIteratorOptions
		opts.Prefix = key
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item().Key()[1:]
			var freq uint32
			err := it.Item().Value(func(val []byte) error {
				freq = binary.LittleEndian.Uint32(val)
				return nil
			})
			if err != nil {
				return err
			}
			ans = append(
				ans,
				SearchResult{
					Value: strings.TrimSpace(string(item)),
					Freq:  freq,
				},
			)
		}
		return nil
	})
	sort.Slice(ans, func(i, j int) bool {
		return ans[i].Freq > ans[j].Freq
	})
	ans = ans[:limit]
	return ans, err
}

func OpenDB(path string) (*DB, error) {
	opts := badger.DefaultOptions(path).
		WithValueLogFileSize(256 << 20). // 256MB value log files
		WithNumMemtables(8).             // More memtables for writes
		WithNumLevelZeroTables(8)

	ans := &DB{}
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open cql database: %w", err)
	}
	ans.bdb = db
	return ans, nil
}
