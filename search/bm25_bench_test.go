package search

import (
	"fmt"
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
)

func makeBenchDocs(n int) []index.SearchDoc {
	docs := make([]index.SearchDoc, n)
	for i := range n {
		id := fmt.Sprintf("tool-%d", i)
		docs[i] = index.SearchDoc{
			ID:      id,
			DocText: fmt.Sprintf("tool %d description for benchmarking with various keywords like git docker kubernetes", i),
			Summary: index.Summary{
				ID:               id,
				Name:             fmt.Sprintf("Tool%d", i),
				Namespace:        "benchmark",
				ShortDescription: fmt.Sprintf("Description for tool %d", i),
				Tags:             []string{"benchmark", "test"},
			},
		}
	}
	return docs
}

func BenchmarkSearch_ColdIndex(b *testing.B) {
	docs := makeBenchDocs(1000)

	for b.Loop() {
		s := NewBM25Searcher(BM25Config{})
		if _, err := s.Search("git", 10, docs); err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}

func BenchmarkSearch_WarmIndex(b *testing.B) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeBenchDocs(1000)

	// Warm up the index
	if _, err := s.Search("git", 10, docs); err != nil {
		b.Fatalf("warmup search failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := s.Search("kubernetes", 10, docs); err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}

func BenchmarkSearch_EmptyQuery(b *testing.B) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeBenchDocs(1000)

	for b.Loop() {
		if _, err := s.Search("", 10, docs); err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}

func BenchmarkSearch_MultiTerm(b *testing.B) {
	s := NewBM25Searcher(BM25Config{})
	docs := makeBenchDocs(1000)

	// Warm up
	if _, err := s.Search("git docker", 10, docs); err != nil {
		b.Fatalf("warmup search failed: %v", err)
	}

	b.ResetTimer()
	for b.Loop() {
		if _, err := s.Search("git docker kubernetes", 10, docs); err != nil {
			b.Fatalf("search failed: %v", err)
		}
	}
}

func BenchmarkSearch_VaryingCatalogSize(b *testing.B) {
	sizes := []int{100, 500, 1000, 2000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			s := NewBM25Searcher(BM25Config{})
			docs := makeBenchDocs(size)

			// Warm up
			if _, err := s.Search("git", 10, docs); err != nil {
				b.Fatalf("warmup search failed: %v", err)
			}

			b.ResetTimer()
			for b.Loop() {
				if _, err := s.Search("git", 10, docs); err != nil {
					b.Fatalf("search failed: %v", err)
				}
			}
		})
	}
}
