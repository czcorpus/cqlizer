package index

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/czcorpus/cqlizer/lsh"
)

const (
	ConcreteQueryPrefix byte = 0x00 // query in its original form
	AbstractQueryPrefix byte = 0x01 // query in its generalized form
	AuxDataPrefix       byte = 0x02 // auxiliary data
	CQLEmbeddingPrefix  byte = 0x03 // CQL embedding vectors
	HyperplanePrefix    byte = 0x04 // LSH hyperplanes
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
	// Global hyperplanes - loaded once per application
	globalHyperplanes []lsh.Vector
)

// hyperplaneKey creates a key for storing hyperplane data
func hyperplaneKey(index int) []byte {
	key := make([]byte, 5) // prefix + 4 bytes for index
	key[0] = HyperplanePrefix
	binary.LittleEndian.PutUint32(key[1:], uint32(index))
	return key
}

// encodeHyperplane serializes a hyperplane vector using IEEE 754 binary representation
func encodeHyperplane(hyperplane lsh.Vector) []byte {
	data := make([]byte, 8*len(hyperplane))
	for i, v := range hyperplane {
		binary.LittleEndian.PutUint64(data[i*8:], math.Float64bits(v))
	}
	return data
}

// decodeHyperplane deserializes a hyperplane vector using IEEE 754 binary representation
func decodeHyperplane(data []byte) lsh.Vector {
	dim := len(data) / 8
	hyperplane := make(lsh.Vector, dim)
	for i := 0; i < dim; i++ {
		bits := binary.LittleEndian.Uint64(data[i*8:])
		hyperplane[i] = math.Float64frombits(bits)
	}
	return hyperplane
}

// generateLSHKeys creates LSH hash keys from a vector using stored hyperplanes
func generateLSHKeys(vector []float32) [][]byte {
	if len(globalHyperplanes) == 0 {
		panic("hyperplanes not initialized - call DB.InitializeHyperplanes() first")
	}

	// Convert float32 to float64
	point := make(lsh.Vector, len(vector))
	for i, v := range vector {
		point[i] = float64(v)
	}

	// Normalize the vector for cosine similarity
	point = point.Normalize()

	// Compute binary hash using stored hyperplanes
	numBytes := (len(globalHyperplanes) + 7) / 8
	hash := make([]byte, numBytes)

	for i, hyperplane := range globalHyperplanes {
		if point.Dot(hyperplane) > 0 {
			// Set bit i to 1
			byteIndex := i / 8
			bitIndex := uint(i % 8)
			hash[byteIndex] |= 1 << bitIndex
		}
	}

	// Create a single key with the binary hash
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
