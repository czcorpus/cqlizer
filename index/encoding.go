package index

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/czcorpus/cqlizer/lsh"
)

const (
	ConcreteQueryPrefix byte = 0x00 // query in its original form
	AbstractQueryPrefix byte = 0x01 // query in its generalized form
	AuxDataPrefix       byte = 0x02 // auxiliary data
	CQLEmbeddingPrefix  byte = 0x03 // CQL embedding vectors
)

func encodeOriginalQuery(lemma string) []byte {
	lemmaBytes := []byte(lemma)
	key := make([]byte, 1+len(lemmaBytes))
	key[0] = ConcreteQueryPrefix
	copy(key[1:], lemmaBytes)
	return key
}

func encodeFrequency(freq uint32) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, freq)
	return buf
}

func decodeFrequency(data []byte) uint32 {
	return binary.LittleEndian.Uint32(data)
}

func encodeTime(t time.Time) []byte {
	buf := make([]byte, 16) // 8 bytes for seconds + 8 bytes for nanoseconds

	// Convert to UTC to ensure consistency
	utc := t.UTC()

	// Store Unix timestamp (seconds since epoch)
	binary.BigEndian.PutUint64(buf[0:8], uint64(utc.Unix()))

	// Store nanoseconds component
	binary.BigEndian.PutUint64(buf[8:16], uint64(utc.Nanosecond()))

	return buf
}

func decodeTime(data []byte) (time.Time, error) {
	if len(data) != 16 {
		return time.Time{}, fmt.Errorf("invalid byte slice length: expected 16, got %d", len(data))
	}

	// Extract seconds
	seconds := int64(binary.BigEndian.Uint64(data[0:8]))

	// Extract nanoseconds
	nanoseconds := int64(binary.BigEndian.Uint64(data[8:16]))

	// Reconstruct time in UTC
	return time.Unix(seconds, nanoseconds).UTC(), nil
}

// LSH parameters - can be tuned for performance vs accuracy
const (
	LSHNumHyperplanes = 64 // Number of hyperplanes for hash computation
	LSHSeed           = 42 // Random seed for reproducible results
)

var (
	// Global LSH instance - initialized once per application
	globalLSH *lsh.RandomHyperplaneLSH
)

// initializeLSH creates a global LSH instance
func initializeLSH(dimension int) {
	if globalLSH == nil {
		// Use the actual vector dimension
		globalLSH = lsh.NewRandomHyperplaneLSH(dimension, LSHNumHyperplanes, LSHSeed)
	}
}

// generateLSHKeys creates LSH hash keys from a vector using the local RandomHyperplaneLSH
func generateLSHKeys(vector []float32) [][]byte {
	// Convert float32 to float64 as required by the local lsh implementation
	point := make(lsh.Vector, len(vector))
	for i, v := range vector {
		point[i] = float64(v)
	}

	// Initialize LSH with the actual vector dimension
	initializeLSH(len(vector))

	// Compute hash directly using the local implementation
	hash := globalLSH.ComputeHash(point)

	// Create a single key with the computed hash
	key := make([]byte, 1+len(hash)) // prefix + hash
	key[0] = CQLEmbeddingPrefix
	copy(key[1:], hash)

	// Return single key as array for compatibility
	return [][]byte{key}
}

// encodeAbstractQuery encodes the abstract query string as the value
func encodeAbstractQuery(abstractQuery string) []byte {
	return []byte(abstractQuery)
}

// decodeAbstractQuery decodes the abstract query string from bytes
func decodeAbstractQuery(data []byte) string {
	return string(data)
}
