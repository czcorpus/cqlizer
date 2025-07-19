package embedding

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sajari/word2vec"
)

// CQLEmbedding represents the vector embedding of a CQL query
type CQLEmbedding struct {
	OriginalQuery string    `json:"originalQuery"`
	AbstractQuery string    `json:"abstractQuery"`
	Vector        []float32 `json:"vector"`
	TokenCount    int       `json:"tokenCount"`
}

// CQLEmbedder handles the creation of embeddings for CQL queries
type CQLEmbedder struct {
	model *word2vec.Model
}

// NewCQLEmbedder creates a new CQL embedder with a word2vec model
func NewCQLEmbedder(modelPath string) (*CQLEmbedder, error) {
	file, err := os.Open(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open model file: %w", err)
	}
	defer file.Close()

	model, err := word2vec.FromReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to load word2vec model: %w", err)
	}

	return &CQLEmbedder{
		model: model,
	}, nil
}

// NewCQLEmbedderFromReader creates a new CQL embedder with a word2vec model from an io.Reader
func NewCQLEmbedderFromReader(reader io.Reader) (*CQLEmbedder, error) {
	model, err := word2vec.FromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to load word2vec model: %w", err)
	}

	return &CQLEmbedder{
		model: model,
	}, nil
}

// tokenizeAbstractQuery splits an abstract CQL query into tokens
func (ce *CQLEmbedder) tokenizeAbstractQuery(abstractQuery string) []string {
	// Split by whitespace and filter out empty strings
	tokens := strings.Fields(abstractQuery)

	// Additional filtering - remove very short tokens and normalize
	var filteredTokens []string
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if len(token) > 0 {
			// Convert to lowercase for better word2vec matching
			filteredTokens = append(filteredTokens, strings.ToLower(token))
		}
	}

	return filteredTokens
}

// CreateEmbedding creates a vector embedding for a CQL query
func (ce *CQLEmbedder) CreateEmbedding(abstractQuery string) (*CQLEmbedding, error) {

	// Tokenize the abstract query
	tokens := ce.tokenizeAbstractQuery(abstractQuery)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no valid tokens found in abstract query '%s'", abstractQuery)
	}

	// Get vector dimensions from the model
	vectorSize := ce.model.Dim()
	sumVector := make([]float32, vectorSize)
	validTokenCount := 0

	// Sum vectors for each token using the Map method
	wordMap := ce.model.Map(tokens)

	for _, token := range tokens {
		if vector, found := wordMap[token]; found {
			for i, val := range vector {
				sumVector[i] += val
			}
			validTokenCount++
		}
		// Note: We silently skip tokens not found in the model
		// This is common in word2vec applications
	}

	// If no tokens were found in the model, return an error
	if validTokenCount == 0 {
		return nil, fmt.Errorf("no tokens from abstract query '%s' found in word2vec model", abstractQuery)
	}

	// Optionally, we could normalize by dividing by the number of valid tokens
	// For now, we return the sum vector as requested

	return &CQLEmbedding{
		AbstractQuery: abstractQuery,
		Vector:        sumVector,
		TokenCount:    validTokenCount,
	}, nil
}

// CreateEmbeddingNormalized creates a normalized vector embedding (average instead of sum)
func (ce *CQLEmbedder) CreateEmbeddingNormalized(cqlQuery string) (*CQLEmbedding, error) {
	embedding, err := ce.CreateEmbedding(cqlQuery)
	if err != nil {
		return nil, err
	}

	// Normalize by dividing by the number of valid tokens
	if embedding.TokenCount > 0 {
		for i := range embedding.Vector {
			embedding.Vector[i] /= float32(embedding.TokenCount)
		}
	}

	return embedding, nil
}

// GetVectorDimensions returns the dimensionality of the word2vec model
func (ce *CQLEmbedder) GetVectorDimensions() int {
	return ce.model.Dim()
}

// GetModelVocabularySize returns the number of words in the word2vec model
func (ce *CQLEmbedder) GetModelVocabularySize() int {
	return ce.model.Size()
}
