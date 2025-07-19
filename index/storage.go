package index

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/czcorpus/cqlizer/lsh"
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

// EmbeddingResult represents a similarity search result
type EmbeddingResult struct {
	AbstractQuery string  `json:"abstractQuery"`
	Score         float32 `json:"score"` // Can be used for ranking if needed
}

// StoreEmbedding stores a vector->query mapping using LSH for similarity search
func (db *DB) StoreEmbedding(abstractQuery string, vector []float32) error {
	return db.bdb.Update(func(txn *badger.Txn) error {
		return db.StoreEmbeddingTx(txn, abstractQuery, vector)
	})
}

func (db *DB) StoreEmbeddingTx(txn *badger.Txn, abstractQuery string, vector []float32) error {

	// Generate all LSH keys for this vector
	lshKeys := generateLSHKeys(vector)
	value := encodeAbstractQuery(abstractQuery)

	// Store the query in all LSH buckets
	for _, key := range lshKeys {
		if err := txn.Set(key, value); err != nil {
			return fmt.Errorf("failed to store LSH key: %w", err)
		}
	}
	return nil
}

// FindSimilarQueries finds abstract queries similar to the given vector
func (db *DB) FindSimilarQueries(queryVector []float32, maxResults int) ([]EmbeddingResult, error) {
	// Generate LSH keys for the query vector
	lshKeys := generateLSHKeys(queryVector)
	
	// Use map to deduplicate results (same query might appear in multiple LSH buckets)
	resultMap := make(map[string]*EmbeddingResult)

	err := db.bdb.View(func(txn *badger.Txn) error {
		// Check each LSH bucket
		for _, key := range lshKeys {
			item, err := txn.Get(key)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					continue // This LSH bucket is empty, which is normal
				}
				return fmt.Errorf("failed to read LSH key: %w", err)
			}
			err = item.Value(func(val []byte) error {
				abstractQuery := decodeAbstractQuery(val)

				// Add to results if not already present
				if _, exists := resultMap[abstractQuery]; !exists {
					resultMap[abstractQuery] = &EmbeddingResult{
						AbstractQuery: abstractQuery,
						Score:         1.0, // Simple scoring - can be improved
					}
				} else {
					// Query found in multiple buckets - increase score
					resultMap[abstractQuery].Score += 0.1
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Convert map to slice and limit results
	results := make([]EmbeddingResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, *result)
	}

	// Sort by score (descending) and limit results
	// Simple sorting - can be optimized for large result sets
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// DeleteEmbedding removes all LSH entries for a given vector
func (db *DB) DeleteEmbedding(vector []float32) error {
	lshKeys := generateLSHKeys(vector)

	return db.bdb.Update(func(txn *badger.Txn) error {
		for _, key := range lshKeys {
			if err := txn.Delete(key); err != nil && err != badger.ErrKeyNotFound {
				return fmt.Errorf("failed to delete LSH key: %w", err)
			}
		}
		return nil
	})
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
	ans = ans[:min(limit, len(ans))]
	return ans, err
}

// InitializeHyperplanes loads or creates hyperplanes for the database
func (db *DB) InitializeHyperplanes(dimension int) error {
	// First try to load existing hyperplanes
	err := db.loadHyperplanes(dimension)
	if err == nil {
		return nil // Successfully loaded existing hyperplanes
	}

	// If loading failed, create new hyperplanes
	return db.createAndStoreHyperplanes(dimension)
}

// loadHyperplanes attempts to load hyperplanes from the database
func (db *DB) loadHyperplanes(expectedDimension int) error {
	globalHyperplanes = make([]lsh.Vector, 0, LSHNumHyperplanes)
	
	return db.bdb.View(func(txn *badger.Txn) error {
		for i := 0; i < LSHNumHyperplanes; i++ {
			key := hyperplaneKey(i)
			item, err := txn.Get(key)
			if err != nil {
				return fmt.Errorf("hyperplane %d not found: %w", i, err)
			}
			
			err = item.Value(func(val []byte) error {
				hyperplane := decodeHyperplane(val)
				if len(hyperplane) != expectedDimension {
					return fmt.Errorf("hyperplane dimension mismatch: expected %d, got %d", expectedDimension, len(hyperplane))
				}
				globalHyperplanes = append(globalHyperplanes, hyperplane)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// createAndStoreHyperplanes generates new random hyperplanes and stores them
func (db *DB) createAndStoreHyperplanes(dimension int) error {
	rng := rand.New(rand.NewSource(LSHSeed))
	globalHyperplanes = make([]lsh.Vector, LSHNumHyperplanes)

	return db.bdb.Update(func(txn *badger.Txn) error {
		for i := 0; i < LSHNumHyperplanes; i++ {
			// Generate random hyperplane
			hyperplane := make(lsh.Vector, dimension)
			for j := 0; j < dimension; j++ {
				hyperplane[j] = rng.NormFloat64()
			}
			// Normalize hyperplane
			hyperplane = hyperplane.Normalize()
			
			// Store in global array
			globalHyperplanes[i] = hyperplane
			
			// Persist to database
			key := hyperplaneKey(i)
			value := encodeHyperplane(hyperplane)
			if err := txn.Set(key, value); err != nil {
				return fmt.Errorf("failed to store hyperplane %d: %w", i, err)
			}
		}
		return nil
	})
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
