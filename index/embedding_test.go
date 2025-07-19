package index

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoreAndFindSimilarEmbeddings(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "cqlizer_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Open test database
	db, err := OpenDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Test data
	abstractQuery1 := " ( pos \"word\" ) <AND> ( lemma \"test\" )"
	vector1 := []float32{1.5, -2.3, 0.8, 4.1, -1.2, 2.0, 3.0, -0.5, 1.8, 0.2}
	
	abstractQuery2 := " ( pos \"noun\" ) <OR> ( lemma \"run\" )"
	vector2 := []float32{1.6, -2.1, 0.9, 4.0, -1.1, 2.1, 2.9, -0.4, 1.9, 0.3}

	// Store embeddings
	err = db.StoreEmbedding(abstractQuery1, vector1)
	require.NoError(t, err)

	err = db.StoreEmbedding(abstractQuery2, vector2)
	require.NoError(t, err)

	// Search for similar queries using vector1 (should find itself and possibly vector2)
	results, err := db.FindSimilarQueries(vector1, 5)
	require.NoError(t, err)

	// Should find at least one result (the exact match)
	assert.GreaterOrEqual(t, len(results), 1)

	// The first result should be our exact query (or at least contain it)
	found := false
	for _, result := range results {
		if result.AbstractQuery == abstractQuery1 {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find the exact query we stored")
}

func TestFindSimilarQueries_Empty(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "cqlizer_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Open test database
	db, err := OpenDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Search in empty database
	queryVector := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	results, err := db.FindSimilarQueries(queryVector, 5)
	require.NoError(t, err)

	// Should return empty results
	assert.Equal(t, 0, len(results))
}

func TestDeleteEmbedding(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "cqlizer_test_")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Open test database
	db, err := OpenDB(tmpDir)
	require.NoError(t, err)
	defer db.Close()

	// Test data
	abstractQuery := " ( pos \"word\" )"
	vector := []float32{1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0}

	// Store embedding
	err = db.StoreEmbedding(abstractQuery, vector)
	require.NoError(t, err)

	// Verify it's there
	results, err := db.FindSimilarQueries(vector, 5)
	require.NoError(t, err)
	assert.Equal(t, 1, len(results))

	// Delete embedding
	err = db.DeleteEmbedding(vector)
	require.NoError(t, err)

	// Verify it's gone
	results, err = db.FindSimilarQueries(vector, 5)
	require.NoError(t, err)
	assert.Equal(t, 0, len(results))
}

func TestLSHKeyGeneration(t *testing.T) {
	// Test that LSH key generation is consistent
	vector := []float32{1.5, -2.3, 0.0, 4.1, -1.2}
	
	// Generate keys twice
	keys1 := generateLSHKeys(vector)
	keys2 := generateLSHKeys(vector)
	
	// Should be identical
	require.Equal(t, len(keys1), len(keys2))
	for i := range keys1 {
		assert.Equal(t, keys1[i], keys2[i], "LSH keys should be deterministic")
	}
	
	// Should generate the expected number of keys
	assert.Equal(t, 1, len(keys1)) // Now returns single key instead of multiple bands
}

func TestAbstractQueryEncoding(t *testing.T) {
	// Test encoding/decoding functions directly
	originalQuery := " ( pos \"word\" ) <AND> ( lemma \"test\" )"
	
	// Encode
	encoded := encodeAbstractQuery(originalQuery)
	
	// Decode
	decoded := decodeAbstractQuery(encoded)
	
	// Verify
	assert.Equal(t, originalQuery, decoded)
}