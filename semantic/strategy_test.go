package semantic

import (
	"context"
	"math"
	"testing"
)

type stubBM25Scorer struct {
	score float64
}

func (s stubBM25Scorer) Score(_ string, _ Document) float64 {
	return s.score
}

type stubEmbedder struct {
	queryVec []float32
	docVec   []float32
}

func (s stubEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if text == "query" {
		return s.queryVec, nil
	}
	return s.docVec, nil
}

type stubStrategy struct {
	score float64
}

func (s stubStrategy) Score(_ context.Context, _ string, _ Document) (float64, error) {
	return s.score, nil
}

func TestStrategy_BM25Only(t *testing.T) {
	bm25 := NewBM25Strategy(stubBM25Scorer{score: 2.5})
	score, err := bm25.Score(context.Background(), "query", Document{ID: "d1"})
	if err != nil {
		t.Fatalf("score failed: %v", err)
	}
	if score != 2.5 {
		t.Fatalf("score = %v, want 2.5", score)
	}
}

func TestStrategy_EmbeddingOnly(t *testing.T) {
	embed := NewEmbeddingStrategy(stubEmbedder{
		queryVec: []float32{1, 0},
		docVec:   []float32{1, 0},
	})

	score, err := embed.Score(context.Background(), "query", Document{ID: "d1", Text: "doc"})
	if err != nil {
		t.Fatalf("score failed: %v", err)
	}

	if math.Abs(score-1.0) > 1e-6 {
		t.Fatalf("score = %v, want 1.0", score)
	}
}

func TestStrategy_HybridWeights(t *testing.T) {
	bm25 := stubStrategy{score: 1}
	emb := stubStrategy{score: 3}

	hybrid, err := NewHybridStrategy(bm25, emb, 0.25)
	if err != nil {
		t.Fatalf("NewHybridStrategy failed: %v", err)
	}

	score, err := hybrid.Score(context.Background(), "query", Document{ID: "d1"})
	if err != nil {
		t.Fatalf("score failed: %v", err)
	}

	want := 2.5
	if math.Abs(score-want) > 1e-6 {
		t.Fatalf("score = %v, want %v", score, want)
	}
}

// ============================================================
// Tests for NewBM25Strategy with nil scorer
// ============================================================

func TestNewBM25Strategy_NilScorer_UsesDefault(t *testing.T) {
	// When nil scorer is passed, default scorer should be used
	strategy := NewBM25Strategy(nil)

	// Test with documents that have matching tokens
	doc := Document{
		ID:   "test",
		Text: "hello world",
	}

	score, err := strategy.Score(context.Background(), "hello", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// Default scorer does token overlap - "hello" matches "hello" in doc
	if score != 1.0 {
		t.Errorf("expected score 1.0 for single match, got %v", score)
	}
}

func TestNewBM25Strategy_DefaultScorer_MultipleMatches(t *testing.T) {
	strategy := NewBM25Strategy(nil)

	doc := Document{
		ID:   "test",
		Text: "hello world hello",
	}

	// Query with multiple tokens that match
	score, err := strategy.Score(context.Background(), "hello world", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// Both "hello" and "world" are in the doc
	if score != 2.0 {
		t.Errorf("expected score 2.0 for two matches, got %v", score)
	}
}

func TestNewBM25Strategy_DefaultScorer_NoMatches(t *testing.T) {
	strategy := NewBM25Strategy(nil)

	doc := Document{
		ID:   "test",
		Text: "foo bar baz",
	}

	score, err := strategy.Score(context.Background(), "hello world", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	if score != 0.0 {
		t.Errorf("expected score 0.0 for no matches, got %v", score)
	}
}

func TestNewBM25Strategy_DefaultScorer_EmptyQuery(t *testing.T) {
	strategy := NewBM25Strategy(nil)

	doc := Document{
		ID:   "test",
		Text: "hello world",
	}

	score, err := strategy.Score(context.Background(), "", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	if score != 0.0 {
		t.Errorf("expected score 0.0 for empty query, got %v", score)
	}
}

func TestNewBM25Strategy_DefaultScorer_EmptyDoc(t *testing.T) {
	strategy := NewBM25Strategy(nil)

	doc := Document{
		ID:   "test",
		Text: "",
	}

	score, err := strategy.Score(context.Background(), "hello", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	if score != 0.0 {
		t.Errorf("expected score 0.0 for empty doc, got %v", score)
	}
}

func TestNewBM25Strategy_DefaultScorer_CaseInsensitive(t *testing.T) {
	strategy := NewBM25Strategy(nil)

	doc := Document{
		ID:   "test",
		Text: "Hello World",
	}

	score, err := strategy.Score(context.Background(), "HELLO", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	if score != 1.0 {
		t.Errorf("expected score 1.0 for case-insensitive match, got %v", score)
	}
}

func TestNewBM25Strategy_UsesNormalizedDoc(t *testing.T) {
	strategy := NewBM25Strategy(nil)

	// Document with empty Text but has Name/Description fields
	doc := Document{
		ID:          "test",
		Name:        "mytool",
		Description: "a useful tool",
		Text:        "", // Empty - should use Normalized()
	}

	score, err := strategy.Score(context.Background(), "mytool", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// Should find "mytool" in the normalized text
	if score < 1.0 {
		t.Errorf("expected score >= 1.0 for normalized match, got %v", score)
	}
}

// ============================================================
// Tests for NewHybridStrategy validation
// ============================================================

func TestNewHybridStrategy_NilBM25(t *testing.T) {
	_, err := NewHybridStrategy(nil, stubStrategy{}, 0.5)
	if err != ErrInvalidHybridConfig {
		t.Errorf("expected ErrInvalidHybridConfig, got %v", err)
	}
}

func TestNewHybridStrategy_NilEmbedding(t *testing.T) {
	_, err := NewHybridStrategy(stubStrategy{}, nil, 0.5)
	if err != ErrInvalidHybridConfig {
		t.Errorf("expected ErrInvalidHybridConfig, got %v", err)
	}
}

func TestNewHybridStrategy_AlphaNegative(t *testing.T) {
	_, err := NewHybridStrategy(stubStrategy{}, stubStrategy{}, -0.1)
	if err != ErrInvalidHybridConfig {
		t.Errorf("expected ErrInvalidHybridConfig, got %v", err)
	}
}

func TestNewHybridStrategy_AlphaGreaterThanOne(t *testing.T) {
	_, err := NewHybridStrategy(stubStrategy{}, stubStrategy{}, 1.1)
	if err != ErrInvalidHybridConfig {
		t.Errorf("expected ErrInvalidHybridConfig, got %v", err)
	}
}

func TestNewHybridStrategy_AlphaZero(t *testing.T) {
	// Alpha = 0 means all embedding weight
	hybrid, err := NewHybridStrategy(stubStrategy{score: 10}, stubStrategy{score: 5}, 0.0)
	if err != nil {
		t.Fatalf("NewHybridStrategy failed: %v", err)
	}

	score, err := hybrid.Score(context.Background(), "query", Document{})
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// 0*10 + 1*5 = 5
	if score != 5.0 {
		t.Errorf("expected score 5.0 with alpha=0, got %v", score)
	}
}

func TestNewHybridStrategy_AlphaOne(t *testing.T) {
	// Alpha = 1 means all BM25 weight
	hybrid, err := NewHybridStrategy(stubStrategy{score: 10}, stubStrategy{score: 5}, 1.0)
	if err != nil {
		t.Fatalf("NewHybridStrategy failed: %v", err)
	}

	score, err := hybrid.Score(context.Background(), "query", Document{})
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// 1*10 + 0*5 = 10
	if score != 10.0 {
		t.Errorf("expected score 10.0 with alpha=1, got %v", score)
	}
}

// ============================================================
// Tests for embeddingStrategy error cases
// ============================================================

func TestEmbeddingStrategy_NilEmbedder(t *testing.T) {
	strategy := NewEmbeddingStrategy(nil)

	_, err := strategy.Score(context.Background(), "query", Document{Text: "doc"})
	if err != ErrInvalidEmbedder {
		t.Errorf("expected ErrInvalidEmbedder, got %v", err)
	}
}

type errorEmbedder struct {
	err error
}

func (e errorEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	return nil, e.err
}

func TestEmbeddingStrategy_EmbedQueryError(t *testing.T) {
	expectedErr := context.DeadlineExceeded
	strategy := NewEmbeddingStrategy(errorEmbedder{err: expectedErr})

	_, err := strategy.Score(context.Background(), "query", Document{Text: "doc"})
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

type queryOnlyEmbedder struct {
	queryVec []float32
	docErr   error
}

func (e queryOnlyEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	if text == "query" {
		return e.queryVec, nil
	}
	return nil, e.docErr
}

func TestEmbeddingStrategy_EmbedDocError(t *testing.T) {
	expectedErr := context.Canceled
	strategy := NewEmbeddingStrategy(queryOnlyEmbedder{
		queryVec: []float32{1, 0},
		docErr:   expectedErr,
	})

	_, err := strategy.Score(context.Background(), "query", Document{Text: "doc"})
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestEmbeddingStrategy_UsesNormalizedDoc(t *testing.T) {
	// Test that empty Text uses Normalized()
	strategy := NewEmbeddingStrategy(stubEmbedder{
		queryVec: []float32{1, 0},
		docVec:   []float32{1, 0},
	})

	doc := Document{
		ID:          "test",
		Name:        "mytool",
		Description: "description",
		Text:        "", // Empty - should use Normalized()
	}

	score, err := strategy.Score(context.Background(), "query", doc)
	if err != nil {
		t.Fatalf("Score failed: %v", err)
	}

	// Should succeed with normalized text
	if score != 1.0 {
		t.Errorf("expected score 1.0, got %v", score)
	}
}

// ============================================================
// Tests for hybridStrategy error propagation
// ============================================================

type errorStrategy struct {
	err error
}

func (s errorStrategy) Score(_ context.Context, _ string, _ Document) (float64, error) {
	return 0, s.err
}

func TestHybridStrategy_BM25Error(t *testing.T) {
	expectedErr := context.DeadlineExceeded
	hybrid, _ := NewHybridStrategy(errorStrategy{err: expectedErr}, stubStrategy{}, 0.5)

	_, err := hybrid.Score(context.Background(), "query", Document{})
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestHybridStrategy_EmbeddingError(t *testing.T) {
	expectedErr := context.Canceled
	hybrid, _ := NewHybridStrategy(stubStrategy{}, errorStrategy{err: expectedErr}, 0.5)

	_, err := hybrid.Score(context.Background(), "query", Document{})
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

// ============================================================
// Tests for cosineSimilarity edge cases
// ============================================================

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"both empty", []float32{}, []float32{}, 0},
		{"a empty", []float32{}, []float32{1, 0}, 0},
		{"b empty", []float32{1, 0}, []float32{}, 0},
		{"different lengths", []float32{1, 0}, []float32{1, 0, 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCosineSimilarity_ZeroVectors(t *testing.T) {
	// Zero vectors should return 0 (avoid division by zero)
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{"a zero", []float32{0, 0}, []float32{1, 0}, 0},
		{"b zero", []float32{1, 0}, []float32{0, 0}, 0},
		{"both zero", []float32{0, 0}, []float32{0, 0}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("cosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	// Orthogonal vectors should have similarity 0
	a := []float32{1, 0}
	b := []float32{0, 1}

	got := cosineSimilarity(a, b)
	if got != 0 {
		t.Errorf("cosineSimilarity of orthogonal vectors = %v, want 0", got)
	}
}

func TestCosineSimilarity_IdenticalVectors(t *testing.T) {
	a := []float32{3, 4}
	b := []float32{3, 4}

	got := cosineSimilarity(a, b)
	if math.Abs(got-1.0) > 1e-6 {
		t.Errorf("cosineSimilarity of identical vectors = %v, want 1.0", got)
	}
}

func TestCosineSimilarity_OppositeVectors(t *testing.T) {
	a := []float32{1, 0}
	b := []float32{-1, 0}

	got := cosineSimilarity(a, b)
	if math.Abs(got-(-1.0)) > 1e-6 {
		t.Errorf("cosineSimilarity of opposite vectors = %v, want -1.0", got)
	}
}
