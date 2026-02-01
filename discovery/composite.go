package discovery

import (
	"context"

	"github.com/jonwraymond/tooldiscovery/index"
	"github.com/jonwraymond/tooldiscovery/semantic"
)

// CompositeSearcher combines multiple search strategies into a unified searcher.
// It implements index.Searcher for compatibility with InMemoryIndex.
type CompositeSearcher interface {
	index.Searcher

	// SearchWithScores returns results with detailed score information.
	SearchWithScores(ctx context.Context, query string, limit int, docs []index.SearchDoc) (Results, error)

	// ScoreType returns the type of scoring used by this searcher.
	GetScoreType() ScoreType
}

// HybridSearcher combines BM25 and semantic search with configurable weighting.
type HybridSearcher struct {
	bm25Strategy      semantic.Strategy
	embeddingStrategy semantic.Strategy
	alpha             float64 // BM25 weight (1-alpha for semantic)
}

// HybridOptions configures a HybridSearcher.
type HybridOptions struct {
	// BM25Scorer is an optional custom BM25 scorer. If nil, uses default.
	BM25Scorer semantic.BM25Scorer

	// Embedder generates embeddings for semantic search. Required.
	Embedder semantic.Embedder

	// Alpha is the BM25 weight (0.0 to 1.0). Semantic weight is 1-Alpha.
	// Default: 0.5 (equal weighting)
	Alpha float64
}

// NewHybridSearcher creates a new hybrid searcher combining BM25 and semantic search.
func NewHybridSearcher(opts HybridOptions) (*HybridSearcher, error) {
	if opts.Embedder == nil {
		return nil, semantic.ErrInvalidEmbedder
	}

	alpha := opts.Alpha
	if alpha < 0 || alpha > 1 {
		return nil, semantic.ErrInvalidHybridConfig
	}

	bm25 := semantic.NewBM25Strategy(opts.BM25Scorer)
	embedding := semantic.NewEmbeddingStrategy(opts.Embedder)

	return &HybridSearcher{
		bm25Strategy:      bm25,
		embeddingStrategy: embedding,
		alpha:             alpha,
	}, nil
}

// Search implements index.Searcher using hybrid scoring.
func (h *HybridSearcher) Search(query string, limit int, docs []index.SearchDoc) ([]index.Summary, error) {
	if limit <= 0 {
		return []index.Summary{}, nil
	}

	results, err := h.SearchWithScores(context.Background(), query, limit, docs)
	if err != nil {
		return nil, err
	}

	return results.Summaries(), nil
}

// SearchWithScores returns results with detailed hybrid scores.
func (h *HybridSearcher) SearchWithScores(ctx context.Context, query string, limit int, docs []index.SearchDoc) (Results, error) {
	if limit <= 0 {
		return Results{}, nil
	}

	// Convert SearchDocs to semantic Documents
	semDocs := semantic.DocumentsFromSearchDocs(docs)

	// Score all documents
	scored := make([]scoredDoc, 0, len(docs))

	for i, doc := range semDocs {
		normalized := doc.Normalized()

		bm25Score, err := h.bm25Strategy.Score(ctx, query, normalized)
		if err != nil {
			return nil, err
		}

		embScore, err := h.embeddingStrategy.Score(ctx, query, normalized)
		if err != nil {
			return nil, err
		}

		// Weighted combination
		hybridScore := h.alpha*bm25Score + (1-h.alpha)*embScore

		if hybridScore > 0 {
			scored = append(scored, scoredDoc{idx: i, score: hybridScore})
		}
	}

	// Sort by score descending, then ID ascending for determinism
	sortScoredDocs(scored, docs)

	// Build results
	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make(Results, len(scored))
	for i, s := range scored {
		results[i] = Result{
			Summary:   docs[s.idx].Summary,
			Score:     s.score,
			ScoreType: ScoreHybrid,
		}
	}

	return results, nil
}

// GetScoreType returns ScoreHybrid.
func (h *HybridSearcher) GetScoreType() ScoreType {
	return ScoreHybrid
}

// Deterministic reports whether this searcher provides deterministic ordering.
func (h *HybridSearcher) Deterministic() bool {
	return true
}

// BM25OnlySearcher wraps the index's lexical search with score tracking.
type BM25OnlySearcher struct {
	strategy semantic.Strategy
}

// NewBM25OnlySearcher creates a searcher using only BM25 scoring.
func NewBM25OnlySearcher(scorer semantic.BM25Scorer) *BM25OnlySearcher {
	return &BM25OnlySearcher{
		strategy: semantic.NewBM25Strategy(scorer),
	}
}

// Search implements index.Searcher.
func (s *BM25OnlySearcher) Search(query string, limit int, docs []index.SearchDoc) ([]index.Summary, error) {
	if limit <= 0 {
		return []index.Summary{}, nil
	}

	results, err := s.SearchWithScores(context.Background(), query, limit, docs)
	if err != nil {
		return nil, err
	}

	return results.Summaries(), nil
}

// SearchWithScores returns results with BM25 scores.
func (s *BM25OnlySearcher) SearchWithScores(ctx context.Context, query string, limit int, docs []index.SearchDoc) (Results, error) {
	if limit <= 0 {
		return Results{}, nil
	}

	semDocs := semantic.DocumentsFromSearchDocs(docs)

	scored := make([]scoredDoc, 0, len(docs))

	for i, doc := range semDocs {
		normalized := doc.Normalized()
		score, err := s.strategy.Score(ctx, query, normalized)
		if err != nil {
			return nil, err
		}
		if score > 0 {
			scored = append(scored, scoredDoc{idx: i, score: score})
		}
	}

	sortScoredDocs(scored, docs)

	if len(scored) > limit {
		scored = scored[:limit]
	}

	results := make(Results, len(scored))
	for i, sd := range scored {
		results[i] = Result{
			Summary:   docs[sd.idx].Summary,
			Score:     sd.score,
			ScoreType: ScoreBM25,
		}
	}

	return results, nil
}

// GetScoreType returns ScoreBM25.
func (s *BM25OnlySearcher) GetScoreType() ScoreType {
	return ScoreBM25
}

// Deterministic reports whether this searcher provides deterministic ordering.
func (s *BM25OnlySearcher) Deterministic() bool {
	return true
}

// scoredDoc holds a document index with its score.
type scoredDoc struct {
	idx   int
	score float64
}

// sortScoredDocs sorts by score descending, then by ID ascending for determinism.
func sortScoredDocs(scored []scoredDoc, docs []index.SearchDoc) {
	// Sort by score descending, ID ascending for determinism
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			swap := false
			if scored[j].score > scored[i].score {
				swap = true
			} else if scored[j].score == scored[i].score {
				if docs[scored[j].idx].ID < docs[scored[i].idx].ID {
					swap = true
				}
			}
			if swap {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
}
