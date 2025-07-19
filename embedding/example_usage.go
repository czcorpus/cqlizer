package embedding

import (
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the CQL embedding functionality
func ExampleUsage() {
	// Load a word2vec model (you need to provide the path to a trained model)
	modelPath := "./word2vec_model.bin" // Replace with actual model path
	
	embedder, err := NewCQLEmbedder(modelPath)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	
	// Example CQL queries
	queries := []string{
		`[word="hello"]`,
		`[lemma="run" & pos="V.*"]`,
		`[pos="N.*"] [pos="V.*"]`,
		`"hello" "world"`,
	}
	
	fmt.Printf("Model dimensions: %d\n", embedder.GetVectorDimensions())
	fmt.Printf("Model vocabulary size: %d\n", embedder.GetModelVocabularySize())
	fmt.Println()
	
	for i, query := range queries {
		fmt.Printf("Query %d: %s\n", i+1, query)
		
		// Create embedding
		embedding, err := embedder.CreateEmbedding(query)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		
		fmt.Printf("  Abstract form: %s\n", embedding.AbstractQuery)
		fmt.Printf("  Valid tokens: %d\n", embedding.TokenCount)
		fmt.Printf("  Vector dimensions: %d\n", len(embedding.Vector))
		
		// Show first few dimensions of the vector
		if len(embedding.Vector) > 0 {
			fmt.Printf("  First 5 dimensions: ")
			for j := 0; j < 5 && j < len(embedding.Vector); j++ {
				fmt.Printf("%.4f ", embedding.Vector[j])
			}
			fmt.Println()
		}
		
		// Also try normalized version
		normalizedEmbedding, err := embedder.CreateEmbeddingNormalized(query)
		if err != nil {
			fmt.Printf("  Normalized error: %v\n", err)
		} else {
			fmt.Printf("  Normalized first 5: ")
			for j := 0; j < 5 && j < len(normalizedEmbedding.Vector); j++ {
				fmt.Printf("%.4f ", normalizedEmbedding.Vector[j])
			}
			fmt.Println()
		}
		
		fmt.Println()
	}
}

// BatchProcessQueries demonstrates processing multiple queries efficiently
func BatchProcessQueries(embedder *CQLEmbedder, queries []string) ([]*CQLEmbedding, []error) {
	embeddings := make([]*CQLEmbedding, 0, len(queries))
	errors := make([]error, 0)
	
	for _, query := range queries {
		embedding, err := embedder.CreateEmbedding(query)
		if err != nil {
			errors = append(errors, fmt.Errorf("query '%s': %w", query, err))
			continue
		}
		embeddings = append(embeddings, embedding)
	}
	
	return embeddings, errors
}