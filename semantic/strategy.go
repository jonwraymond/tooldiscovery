package semantic

import (
	"context"
	"errors"
	"math"
	"strings"
)

var (
	ErrInvalidEmbedder     = errors.New("semantic: embedder is required")
	ErrInvalidHybridConfig = errors.New("semantic: hybrid strategy requires bm25, embedding, and alpha in [0,1]")
)

// BM25Scorer scores documents for a query using a lexical strategy.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Determinism: identical inputs must yield stable scores.
type BM25Scorer interface {
	Score(query string, doc Document) float64
}

// Embedder produces embeddings for text.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: must honor cancellation/deadlines.
// - Errors: return an error for invalid input or provider failure.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// NewBM25Strategy creates a BM25-only strategy. If scorer is nil, a default
// token-overlap scorer is used.
func NewBM25Strategy(scorer BM25Scorer) Strategy {
	if scorer == nil {
		scorer = defaultBM25Scorer{}
	}
	return bm25Strategy{scorer: scorer}
}

// NewEmbeddingStrategy creates an embedding-only strategy.
func NewEmbeddingStrategy(embedder Embedder) Strategy {
	return embeddingStrategy{embedder: embedder}
}

// NewHybridStrategy creates a weighted hybrid strategy.
func NewHybridStrategy(bm25 Strategy, embedding Strategy, alpha float64) (Strategy, error) {
	if bm25 == nil || embedding == nil || alpha < 0 || alpha > 1 {
		return nil, ErrInvalidHybridConfig
	}
	return hybridStrategy{bm25: bm25, embedding: embedding, alpha: alpha}, nil
}

type bm25Strategy struct {
	scorer BM25Scorer
}

func (s bm25Strategy) Score(_ context.Context, query string, doc Document) (float64, error) {
	norm := doc
	if norm.Text == "" {
		norm = doc.Normalized()
	}
	return s.scorer.Score(query, norm), nil
}

type embeddingStrategy struct {
	embedder Embedder
}

func (s embeddingStrategy) Score(ctx context.Context, query string, doc Document) (float64, error) {
	if s.embedder == nil {
		return 0, ErrInvalidEmbedder
	}

	qVec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return 0, err
	}

	text := doc.Text
	if text == "" {
		text = doc.Normalized().Text
	}
	dVec, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return 0, err
	}

	return cosineSimilarity(qVec, dVec), nil
}

type hybridStrategy struct {
	bm25     Strategy
	embedding Strategy
	alpha    float64
}

func (s hybridStrategy) Score(ctx context.Context, query string, doc Document) (float64, error) {
	bm25Score, err := s.bm25.Score(ctx, query, doc)
	if err != nil {
		return 0, err
	}
	embScore, err := s.embedding.Score(ctx, query, doc)
	if err != nil {
		return 0, err
	}
	return s.alpha*bm25Score + (1-s.alpha)*embScore, nil
}

// defaultBM25Scorer is a simple token-overlap scorer.
type defaultBM25Scorer struct{}

func (defaultBM25Scorer) Score(query string, doc Document) float64 {
	qTokens := tokenize(query)
	if len(qTokens) == 0 {
		return 0
	}
	dTokens := tokenize(doc.Text)
	if len(dTokens) == 0 {
		return 0
	}

	set := make(map[string]struct{}, len(dTokens))
	for _, t := range dTokens {
		set[t] = struct{}{}
	}

	matches := 0
	for _, t := range qTokens {
		if _, ok := set[t]; ok {
			matches++
		}
	}

	return float64(matches)
}

func tokenize(s string) []string {
	fields := strings.Fields(strings.ToLower(s))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
