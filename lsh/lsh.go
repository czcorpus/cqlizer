package lsh

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// Vector represents a high-dimensional vector
type Vector []float64

// VectorWithID stores a vector with its ID
type VectorWithID struct {
	ID     int
	Vector Vector
}

// SimilarityResult represents a query result
type SimilarityResult struct {
	ID         int
	Similarity float64
}

// RandomHyperplaneLSH implements LSH using random hyperplanes
type RandomHyperplaneLSH struct {
	dim            int
	numHyperplanes int
	hyperplanes    []Vector
	storage        map[string][]VectorWithID
	vectors        map[int]Vector
	rng            *rand.Rand
}

// NewRandomHyperplaneLSH creates a new LSH index
func NewRandomHyperplaneLSH(dim, numHyperplanes int, seed int64) *RandomHyperplaneLSH {
	rng := rand.New(rand.NewSource(seed))
	lsh := &RandomHyperplaneLSH{
		dim:            dim,
		numHyperplanes: numHyperplanes,
		hyperplanes:    make([]Vector, numHyperplanes),
		storage:        make(map[string][]VectorWithID),
		vectors:        make(map[int]Vector),
		rng:            rng,
	}

	// Generate random hyperplanes
	for i := 0; i < numHyperplanes; i++ {
		hyperplane := make(Vector, dim)
		for j := 0; j < dim; j++ {
			hyperplane[j] = rng.NormFloat64()
		}
		// Normalize hyperplane
		norm := hyperplane.norm()
		for j := 0; j < dim; j++ {
			hyperplane[j] /= norm
		}
		lsh.hyperplanes[i] = hyperplane
	}

	return lsh
}

// Vector operations
func (v Vector) dot(other Vector) float64 {
	sum := 0.0
	for i := range v {
		sum += v[i] * other[i]
	}
	return sum
}

func (v Vector) norm() float64 {
	sum := 0.0
	for _, val := range v {
		sum += val * val
	}
	return math.Sqrt(sum)
}

func (v Vector) normalize() Vector {
	norm := v.norm()
	normalized := make(Vector, len(v))
	for i := range v {
		normalized[i] = v[i] / norm
	}
	return normalized
}

// ComputeHash computes binary hash for a vector
func (lsh *RandomHyperplaneLSH) ComputeHash(vector Vector) []byte {
	// We'll pack bits into bytes
	numBytes := (lsh.numHyperplanes + 7) / 8
	hash := make([]byte, numBytes)

	for i, hyperplane := range lsh.hyperplanes {
		if vector.dot(hyperplane) > 0 {
			// Set bit i to 1
			byteIndex := i / 8
			bitIndex := uint(i % 8)
			hash[byteIndex] |= 1 << bitIndex
		}
	}

	return hash
}

// Insert adds a vector to the index
func (lsh *RandomHyperplaneLSH) Insert(id int, vector Vector) {
	// Normalize vector for cosine similarity
	normalized := vector.normalize()

	// Compute hash
	hash := lsh.ComputeHash(normalized)
	hashKey := string(hash)

	// Store in map
	lsh.storage[hashKey] = append(lsh.storage[hashKey], VectorWithID{
		ID:     id,
		Vector: normalized,
	})

	// Keep original normalized vector
	lsh.vectors[id] = normalized
}

// hammingDistance computes Hamming distance between two hashes
func hammingDistance(a, b []byte) int {
	dist := 0
	for i := range a {
		xor := a[i] ^ b[i]
		// Count set bits (Brian Kernighan's algorithm)
		for xor != 0 {
			dist++
			xor &= xor - 1
		}
	}
	return dist
}

// getCandidates retrieves candidate vectors for similarity search
func (lsh *RandomHyperplaneLSH) getCandidates(query Vector, maxHammingDistance int) map[int]bool {
	candidates := make(map[int]bool)

	// Compute query hash
	queryHash := lsh.ComputeHash(query)

	// Check all stored hashes
	for hashKey, vectors := range lsh.storage {
		storedHash := []byte(hashKey)

		// Check Hamming distance
		if hammingDistance(queryHash, storedHash) <= maxHammingDistance {
			for _, v := range vectors {
				candidates[v.ID] = true
			}
		}
	}

	return candidates
}

// Query finds k nearest neighbors
func (lsh *RandomHyperplaneLSH) Query(queryVector Vector, k int, maxHammingDistance int) []SimilarityResult {
	// Normalize query
	queryNormalized := queryVector.normalize()

	// Get candidates
	candidates := lsh.getCandidates(queryNormalized, maxHammingDistance)

	if len(candidates) == 0 {
		return []SimilarityResult{}
	}

	// Compute exact similarities for candidates
	results := make([]SimilarityResult, 0, len(candidates))
	for candidateID := range candidates {
		candidateVector := lsh.vectors[candidateID]
		similarity := queryNormalized.dot(candidateVector)
		results = append(results, SimilarityResult{
			ID:         candidateID,
			Similarity: similarity,
		})
	}

	// Sort by similarity
	sort.Slice(results, func(i, j int) bool {
		return results[i].Similarity > results[j].Similarity
	})

	// Return top k
	if len(results) > k {
		results = results[:k]
	}

	return results
}

// PrefixSearch finds vectors with matching hash prefix
func (lsh *RandomHyperplaneLSH) PrefixSearch(prefixBits string) []int {
	var matchingIDs []int
	prefixLen := len(prefixBits)

	for hashKey, vectors := range lsh.storage {
		hash := []byte(hashKey)

		// Check if prefix matches
		matches := true
		for i := 0; i < prefixLen && matches; i++ {
			byteIndex := i / 8
			bitIndex := uint(i % 8)

			bit := (hash[byteIndex] >> bitIndex) & 1
			expectedBit := prefixBits[i] - '0'

			if bit != byte(expectedBit) {
				matches = false
			}
		}

		if matches {
			for _, v := range vectors {
				matchingIDs = append(matchingIDs, v.ID)
			}
		}
	}

	return matchingIDs
}

// Example of key-value store interface for my use case
type KVStore interface {
	Put(key []byte, value []byte) error
	Get(key []byte) ([]byte, error)
	PrefixScan(prefix []byte) ([][]byte, error)
}

// StoreLSHVector shows how to store in a KV store
func StoreLSHVector(store KVStore, hash []byte, vectorID int, vector Vector) error {
	// Create key: hash/vectorID
	key := append(hash, []byte(fmt.Sprintf("/%d", vectorID))...)

	// Serialize vector (simple example - use protobuf/msgpack in production)
	value := make([]byte, 8*len(vector))
	for i, v := range vector {
		binary.LittleEndian.PutUint64(value[i*8:], math.Float64bits(v))
	}

	return store.Put(key, value)
}
