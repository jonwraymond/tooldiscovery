// Package search provides BM25-based search for the index package.
package search

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"

	"github.com/jonwraymond/tooldiscovery/index"
)

// BM25Config configures the BM25 searcher behavior.
type BM25Config struct {
	// Field weighting via token duplication.
	NameBoost      int // default 3
	NamespaceBoost int // default 2
	TagsBoost      int // default 2

	// Safety / performance controls.
	MaxDocs       int // 0 = unlimited
	MaxDocTextLen int // 0 = unlimited
}

// BM25Searcher implements index.Searcher using BM25 ranking.
type BM25Searcher struct {
	cfg BM25Config

	mu              sync.RWMutex
	index           bleve.Index
	idToSummary     map[string]index.Summary
	lastFingerprint string
	indexBuildCount int
}

// Ensure interface compliance at compile time.
var _ index.Searcher = (*BM25Searcher)(nil)
var _ index.DeterministicSearcher = (*BM25Searcher)(nil)

// NewBM25Searcher creates a new BM25-based searcher with the given config.
// Zero values in config are replaced with sensible defaults.
func NewBM25Searcher(cfg BM25Config) *BM25Searcher {
	// Apply defaults for zero values
	if cfg.NameBoost == 0 {
		cfg.NameBoost = 3
	}
	if cfg.NamespaceBoost == 0 {
		cfg.NamespaceBoost = 2
	}
	if cfg.TagsBoost == 0 {
		cfg.TagsBoost = 2
	}

	return &BM25Searcher{
		cfg: cfg,
	}
}

// Deterministic reports whether this searcher returns stable ordering.
func (s *BM25Searcher) Deterministic() bool {
	return true
}

// IndexBuildCount returns the number of times the index has been built.
// This is useful for testing cache behavior.
func (s *BM25Searcher) IndexBuildCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.indexBuildCount
}

// buildWeightedDoc creates a weighted document text for BM25 indexing.
// It duplicates high-signal tokens according to their boost values to
// bias ranking toward name, namespace, and tags.
func buildWeightedDoc(cfg BM25Config, doc index.SearchDoc) string {
	var parts []string

	// Duplicate name tokens according to boost
	if cfg.NameBoost > 0 {
		nameLower := strings.ToLower(doc.Summary.Name)
		for range cfg.NameBoost {
			parts = append(parts, nameLower)
		}
	}

	// Duplicate namespace tokens according to boost
	if cfg.NamespaceBoost > 0 {
		nsLower := strings.ToLower(doc.Summary.Namespace)
		for range cfg.NamespaceBoost {
			parts = append(parts, nsLower)
		}
	}

	// Duplicate each tag according to boost
	if cfg.TagsBoost > 0 {
		for _, tag := range doc.Summary.Tags {
			tagLower := strings.ToLower(tag)
			for range cfg.TagsBoost {
				parts = append(parts, tagLower)
			}
		}
	}

	// Add DocText (possibly truncated)
	docText := strings.ToLower(doc.DocText)
	if cfg.MaxDocTextLen > 0 && len(docText) > cfg.MaxDocTextLen {
		docText = docText[:cfg.MaxDocTextLen]
	}
	if docText != "" {
		parts = append(parts, docText)
	}

	return strings.Join(parts, " ")
}

// indexedDoc is the document structure indexed by Bleve.
type indexedDoc struct {
	Content string `json:"content"`
}

// Search performs a BM25-ranked search over the provided documents.
func (s *BM25Searcher) Search(query string, limit int, docs []index.SearchDoc) ([]index.Summary, error) {
	query = strings.TrimSpace(query)

	// 1. Sort docs by ID FIRST for determinism (before any other operations)
	sortedDocs := sortDocsByID(docs)

	// 2. Apply MaxDocs AFTER sorting for deterministic subset selection
	if s.cfg.MaxDocs > 0 && len(sortedDocs) > s.cfg.MaxDocs {
		sortedDocs = sortedDocs[:s.cfg.MaxDocs]
	}

	// 3. Empty query returns first limit docs from sortedDocs
	if query == "" {
		n := limit
		if n > len(sortedDocs) {
			n = len(sortedDocs)
		}
		results := make([]index.Summary, n)
		for i := range n {
			results[i] = sortedDocs[i].Summary
		}
		return results, nil
	}

	// 4. No docs means no results
	if len(sortedDocs) == 0 {
		return []index.Summary{}, nil
	}
	if limit <= 0 {
		return []index.Summary{}, nil
	}

	// 5. Compute fingerprint from sortedDocs (already sorted)
	fingerprint := computeFingerprint(sortedDocs)

	// 6. Check if we need to rebuild the index
	s.mu.RLock()
	needsRebuild := s.index == nil || s.lastFingerprint != fingerprint
	s.mu.RUnlock()

	// 7. Rebuild uses sortedDocs
	if needsRebuild {
		if err := s.rebuildIndex(sortedDocs, fingerprint); err != nil {
			return nil, err
		}
	}

	// Execute search with read lock
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Normalize query
	query = strings.ToLower(query)

	// 8. Search uses a plain match query to avoid query syntax injection.
	matchQuery := bleve.NewMatchQuery(query)
	matchQuery.SetField("content")
	searchRequest := bleve.NewSearchRequest(matchQuery)
	if limit > len(sortedDocs) {
		limit = len(sortedDocs)
	}
	searchRequest.Size = limit
	searchRequest.SortBy([]string{"-_score", "_id"})
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	// Collect hits with scores for deterministic tie-breaking
	type scoredHit struct {
		id    string
		score float64
	}
	hits := make([]scoredHit, 0, len(searchResult.Hits))
	for _, hit := range searchResult.Hits {
		if _, ok := s.idToSummary[hit.ID]; ok {
			hits = append(hits, scoredHit{id: hit.ID, score: hit.Score})
		}
	}

	// Sort: score DESC, then ID ASC for tie-breaking
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].id < hits[j].id
	})

	// Apply limit and map to summaries
	if len(hits) > limit {
		hits = hits[:limit]
	}
	results := make([]index.Summary, len(hits))
	for i, hit := range hits {
		results[i] = s.idToSummary[hit.id]
	}

	return results, nil
}

// rebuildIndex creates a new Bleve index from the given documents.
func (s *BM25Searcher) rebuildIndex(docs []index.SearchDoc, fingerprint string) error {
	// Build ID to Summary map and create in-memory Bleve index
	idToSummary := make(map[string]index.Summary, len(docs))
	index, err := bleve.NewMemOnly(bleve.NewIndexMapping())
	if err != nil {
		return err
	}

	// Index documents
	batch := index.NewBatch()
	for _, doc := range docs {
		idToSummary[doc.ID] = doc.Summary
		weightedText := buildWeightedDoc(s.cfg, doc)
		if err := batch.Index(doc.ID, indexedDoc{Content: weightedText}); err != nil {
			if cerr := index.Close(); cerr != nil {
				return fmt.Errorf("%w; close index: %v", err, cerr)
			}
			return err
		}
	}
	if err := index.Batch(batch); err != nil {
		if cerr := index.Close(); cerr != nil {
			return fmt.Errorf("%w; close index: %v", err, cerr)
		}
		return err
	}

	// Atomically swap in the new index
	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check fingerprint (another goroutine may have rebuilt)
	if s.lastFingerprint == fingerprint {
		if cerr := index.Close(); cerr != nil {
			return fmt.Errorf("close index: %w", cerr)
		}
		return nil
	}

	// Close old index if it exists
	if s.index != nil {
		if cerr := s.index.Close(); cerr != nil {
			if nerr := index.Close(); nerr != nil {
				return fmt.Errorf("close old index: %v; close new index: %v", cerr, nerr)
			}
			return fmt.Errorf("close old index: %w", cerr)
		}
	}

	s.index = index
	s.idToSummary = idToSummary
	s.lastFingerprint = fingerprint
	s.indexBuildCount++

	return nil
}

// sortDocsByID returns a copy of docs sorted by ID for deterministic fingerprinting.
func sortDocsByID(docs []index.SearchDoc) []index.SearchDoc {
	sorted := make([]index.SearchDoc, len(docs))
	copy(sorted, docs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})
	return sorted
}

// Close releases resources held by the searcher.
func (s *BM25Searcher) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.index != nil {
		err := s.index.Close()
		s.index = nil
		s.idToSummary = nil
		s.lastFingerprint = ""
		return err
	}
	return nil
}
