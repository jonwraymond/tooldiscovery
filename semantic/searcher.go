package semantic

import (
	"context"
	"errors"
	"sort"
)

var ErrInvalidSearcher = errors.New("semantic: searcher requires index and strategy")

// Result represents a scored search result.
type Result struct {
	Document Document
	Score    float64
}

// Strategy scores a document for a given query.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: must honor cancellation/deadlines.
// - Determinism: identical inputs must yield stable scores.
type Strategy interface {
	Score(ctx context.Context, query string, doc Document) (float64, error)
}

// Searcher performs semantic search over indexed documents.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: must honor cancellation/deadlines.
// - Determinism: ordering must be stable for identical inputs.
type Searcher interface {
	Search(ctx context.Context, query string) ([]Result, error)
}

// InMemorySearcher is a deterministic searcher over an Indexer.
type InMemorySearcher struct {
	index    Indexer
	strategy Strategy
}

// NewSearcher creates a new searcher.
func NewSearcher(index Indexer, strategy Strategy) *InMemorySearcher {
	return &InMemorySearcher{index: index, strategy: strategy}
}

// Search scores all documents and returns results ordered by score desc, ID asc.
func (s *InMemorySearcher) Search(ctx context.Context, query string) ([]Result, error) {
	if s.index == nil || s.strategy == nil {
		return nil, ErrInvalidSearcher
	}

	docs := s.index.List(ctx)
	results := make([]Result, 0, len(docs))
	for _, doc := range docs {
		score, err := s.strategy.Score(ctx, query, doc)
		if err != nil {
			return nil, err
		}
		results = append(results, Result{Document: doc, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Document.ID < results[j].Document.ID
		}
		return results[i].Score > results[j].Score
	})

	return results, nil
}

var _ Searcher = (*InMemorySearcher)(nil)
