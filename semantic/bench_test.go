package semantic

import (
	"context"
	"fmt"
	"testing"
)

func makeBenchDoc(i int) Document {
	return Document{
		ID:          fmt.Sprintf("ns_%d:tool_%d", i%10, i),
		Name:        fmt.Sprintf("tool_%d", i),
		Namespace:   fmt.Sprintf("ns_%d", i%10),
		Description: fmt.Sprintf("Description for tool %d with various keywords like git docker kubernetes", i),
		Tags:        []string{"benchmark", "test", fmt.Sprintf("tag_%d", i%5)},
		Category:    fmt.Sprintf("category_%d", i%3),
	}
}

func setupIndexWithDocs(n int) *InMemoryIndex {
	idx := NewInMemoryIndex()
	ctx := context.Background()
	for i := range n {
		doc := makeBenchDoc(i)
		_ = idx.Add(ctx, doc)
	}
	return idx
}

func BenchmarkIndexer_Add(b *testing.B) {
	ctx := context.Background()

	for b.Loop() {
		idx := NewInMemoryIndex()
		doc := makeBenchDoc(0)
		_ = idx.Add(ctx, doc)
	}
}

func BenchmarkIndexer_Add_Sequential(b *testing.B) {
	idx := NewInMemoryIndex()
	ctx := context.Background()

	b.ResetTimer()
	for i := range b.N {
		doc := makeBenchDoc(i)
		_ = idx.Add(ctx, doc)
	}
}

func BenchmarkIndexer_Get(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = idx.Get(ctx, "ns_5:tool_500")
	}
}

func BenchmarkIndexer_List(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_ = idx.List(ctx)
	}
}

func BenchmarkSearcher_BM25(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	strategy := NewBM25Strategy(nil)
	searcher := NewSearcher(idx, strategy)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = searcher.Search(ctx, "git docker")
	}
}

func BenchmarkSearcher_BM25_VaryingCorpus(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
			idx := setupIndexWithDocs(size)
			strategy := NewBM25Strategy(nil)
			searcher := NewSearcher(idx, strategy)
			ctx := context.Background()

			b.ResetTimer()
			for b.Loop() {
				_, _ = searcher.Search(ctx, "git docker")
			}
		})
	}
}

// mockBenchEmbedder provides constant embeddings for benchmarking
type mockBenchEmbedder struct {
	dim int
}

func (m *mockBenchEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	vec := make([]float32, m.dim)
	for i := range vec {
		vec[i] = float32(i) / float32(m.dim)
	}
	return vec, nil
}

func BenchmarkSearcher_Embedding(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	embedder := &mockBenchEmbedder{dim: 384}
	strategy := NewEmbeddingStrategy(embedder)
	searcher := NewSearcher(idx, strategy)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = searcher.Search(ctx, "git docker")
	}
}

func BenchmarkSearcher_Hybrid(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	embedder := &mockBenchEmbedder{dim: 384}
	bm25 := NewBM25Strategy(nil)
	embedding := NewEmbeddingStrategy(embedder)
	hybrid, _ := NewHybridStrategy(bm25, embedding, 0.7)
	searcher := NewSearcher(idx, hybrid)
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = searcher.Search(ctx, "git docker")
	}
}

func BenchmarkSearcher_Hybrid_VaryingAlpha(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	embedder := &mockBenchEmbedder{dim: 384}
	bm25 := NewBM25Strategy(nil)
	embedding := NewEmbeddingStrategy(embedder)
	alphas := []float64{0.0, 0.3, 0.5, 0.7, 1.0}

	for _, alpha := range alphas {
		b.Run(fmt.Sprintf("alpha_%.1f", alpha), func(b *testing.B) {
			hybrid, _ := NewHybridStrategy(bm25, embedding, alpha)
			searcher := NewSearcher(idx, hybrid)
			ctx := context.Background()

			b.ResetTimer()
			for b.Loop() {
				_, _ = searcher.Search(ctx, "git docker")
			}
		})
	}
}

func BenchmarkStrategy_Score_BM25(b *testing.B) {
	strategy := NewBM25Strategy(nil)
	doc := makeBenchDoc(0).Normalized()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = strategy.Score(ctx, "git docker kubernetes", doc)
	}
}

func BenchmarkStrategy_Score_Embedding(b *testing.B) {
	embedder := &mockBenchEmbedder{dim: 384}
	strategy := NewEmbeddingStrategy(embedder)
	doc := makeBenchDoc(0).Normalized()
	ctx := context.Background()

	b.ResetTimer()
	for b.Loop() {
		_, _ = strategy.Score(ctx, "git docker kubernetes", doc)
	}
}

func BenchmarkDocument_Normalized(b *testing.B) {
	doc := Document{
		ID:          "test:tool",
		Name:        "Test Tool",
		Description: "A test tool for benchmarking with various keywords",
		Tags:        []string{"Test", "BENCHMARK", "performance"},
	}

	b.ResetTimer()
	for b.Loop() {
		_ = doc.Normalized()
	}
}

func BenchmarkFilter_Namespace(b *testing.B) {
	docs := make([]Document, 1000)
	for i := range docs {
		docs[i] = makeBenchDoc(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = FilterByNamespace(docs, "ns_5")
	}
}

func BenchmarkFilter_Tags(b *testing.B) {
	docs := make([]Document, 1000)
	for i := range docs {
		docs[i] = makeBenchDoc(i)
	}

	b.ResetTimer()
	for b.Loop() {
		_ = FilterByTags(docs, []string{"benchmark", "tag_2"})
	}
}

func BenchmarkConcurrent_Search(b *testing.B) {
	idx := setupIndexWithDocs(1000)
	strategy := NewBM25Strategy(nil)
	searcher := NewSearcher(idx, strategy)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = searcher.Search(ctx, "git docker")
		}
	})
}

func BenchmarkConcurrent_IndexAndSearch(b *testing.B) {
	idx := NewInMemoryIndex()
	strategy := NewBM25Strategy(nil)
	searcher := NewSearcher(idx, strategy)
	ctx := context.Background()

	// Pre-populate with some docs
	for i := range 500 {
		_ = idx.Add(ctx, makeBenchDoc(i))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 500
		for pb.Next() {
			if i%2 == 0 {
				_, _ = searcher.Search(ctx, "git")
			} else {
				_ = idx.Add(ctx, makeBenchDoc(i))
			}
			i++
		}
	})
}
