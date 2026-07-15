package core

import (
	"crypto/sha256"
	"math"
	"strings"
)

const DefaultEmbeddingDimensions = 1536

func BuildEmbedding(text string) []float32 {
	embedding := make([]float32, DefaultEmbeddingDimensions)
	if strings.TrimSpace(text) == "" {
		return embedding
	}

	hash := sha256.Sum256([]byte(strings.ToLower(text)))
	for i := 0; i < DefaultEmbeddingDimensions; i++ {
		var value uint32
		for j := 0; j < 4; j++ {
			b := hash[(i*4+j)%len(hash)]
			value = (value << 8) | uint32(b)
		}
		embedding[i] = float32(math.Mod(float64(value), 1000.0)) / 1000.0
	}
	return embedding
}

func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		limit := len(a)
		if len(b) < limit {
			limit = len(b)
		}
		if limit == 0 {
			return 0
		}
		var dot, normA, normB float64
		for i := 0; i < limit; i++ {
			ai := float64(a[i])
			bi := float64(b[i])
			dot += ai * bi
			normA += ai * ai
			normB += bi * bi
		}
		if normA == 0 || normB == 0 {
			return 0
		}
		return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
	}

	var dot, normA, normB float64
	for i := 0; i < len(a); i++ {
		ai := float64(a[i])
		bi := float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(normA) * math.Sqrt(normB)))
}

func LexicalScore(text, query string) float32 {
	text = strings.ToLower(strings.TrimSpace(text))
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || text == "" {
		return 0
	}
	if strings.Contains(text, query) {
		return 1
	}

	terms := strings.Fields(query)
	if len(terms) == 0 {
		return 0
	}
	matches := 0
	for _, term := range terms {
		if strings.Contains(text, term) {
			matches++
		}
	}
	if matches == 0 {
		return 0
	}
	return float32(matches) / float32(len(terms))
}
