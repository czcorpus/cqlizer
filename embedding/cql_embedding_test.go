package embedding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizeAbstractQuery(t *testing.T) {
	// Create a dummy embedder (we don't need a real model for tokenization tests)
	embedder := &CQLEmbedder{}
	
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple query",
			input:    " ( pos \"word\" )",
			expected: []string{"(", "pos", "\"word\"", ")"},
		},
		{
			name:     "query with operators",
			input:    " ( pos \"word\" ) <AND> ( lemma \"test\" )",
			expected: []string{"(", "pos", "\"word\"", ")", "<and>", "(", "lemma", "\"test\"", ")"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   \t\n  ",
			expected: nil,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := embedder.tokenizeAbstractQuery(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCreateEmbedding_ParseError(t *testing.T) {
	// Create a dummy embedder 
	embedder := &CQLEmbedder{}
	
	// Test with invalid CQL query
	_, err := embedder.CreateEmbedding("invalid cql query [[[")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse CQL query")
}

// Note: Full integration tests would require a real word2vec model file,
// which is typically large and not suitable for unit tests.
// In practice, you would create integration tests with a small test model.