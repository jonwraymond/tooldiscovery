package discovery

import (
	"github.com/jonwraymond/tooldiscovery/index"
)

// ScoreType indicates the source of a search result's score.
type ScoreType string

const (
	// ScoreBM25 indicates the score came from BM25 lexical search.
	ScoreBM25 ScoreType = "bm25"

	// ScoreEmbedding indicates the score came from embedding-based semantic search.
	ScoreEmbedding ScoreType = "embedding"

	// ScoreHybrid indicates the score is a weighted combination of BM25 and embedding.
	ScoreHybrid ScoreType = "hybrid"
)

// Result represents a unified search result with score details.
type Result struct {
	// Summary contains the tool's metadata (ID, name, namespace, description, tags).
	Summary index.Summary

	// Score is the relevance score for this result.
	// The score's interpretation depends on ScoreType.
	Score float64

	// ScoreType indicates how the Score was computed.
	ScoreType ScoreType
}

// Results is a slice of Result with helper methods.
type Results []Result

// IDs returns just the tool IDs from the results.
func (r Results) IDs() []string {
	ids := make([]string, len(r))
	for i, result := range r {
		ids[i] = result.Summary.ID
	}
	return ids
}

// Summaries returns just the summaries from the results.
func (r Results) Summaries() []index.Summary {
	summaries := make([]index.Summary, len(r))
	for i, result := range r {
		summaries[i] = result.Summary
	}
	return summaries
}

// FilterByNamespace returns results matching the given namespace.
func (r Results) FilterByNamespace(namespace string) Results {
	var filtered Results
	for _, result := range r {
		if result.Summary.Namespace == namespace {
			filtered = append(filtered, result)
		}
	}
	return filtered
}

// FilterByMinScore returns results with score >= minScore.
func (r Results) FilterByMinScore(minScore float64) Results {
	var filtered Results
	for _, result := range r {
		if result.Score >= minScore {
			filtered = append(filtered, result)
		}
	}
	return filtered
}
