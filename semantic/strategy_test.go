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
