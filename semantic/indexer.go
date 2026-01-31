package semantic

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var ErrInvalidDocumentID = errors.New("semantic: document id is required")

// Indexer defines indexing operations for tool documents.
//
// Contract:
// - Concurrency: implementations must be safe for concurrent use.
// - Context: methods must honor cancellation/deadlines where applicable.
// - Errors: invalid IDs should return ErrInvalidDocumentID.
// - Determinism: List returns stable ordering.
type Indexer interface {
	Add(ctx context.Context, doc Document) error
	Update(ctx context.Context, doc Document) error
	Remove(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (Document, bool)
	List(ctx context.Context) []Document
}

// InMemoryIndex is a thread-safe in-memory document index.
type InMemoryIndex struct {
	mu   sync.RWMutex
	docs map[string]Document
}

// NewInMemoryIndex creates a new in-memory index.
func NewInMemoryIndex() *InMemoryIndex {
	return &InMemoryIndex{docs: make(map[string]Document)}
}

// Add inserts or updates a document in the index.
func (i *InMemoryIndex) Add(_ context.Context, doc Document) error {
	if doc.ID == "" {
		return ErrInvalidDocumentID
	}

	norm := doc.Normalized()
	i.mu.Lock()
	i.docs[doc.ID] = norm
	i.mu.Unlock()
	return nil
}

// Update updates a document by ID. If it doesn't exist, it is inserted.
func (i *InMemoryIndex) Update(ctx context.Context, doc Document) error {
	return i.Add(ctx, doc)
}

// Remove deletes a document by ID.
func (i *InMemoryIndex) Remove(_ context.Context, id string) error {
	if id == "" {
		return ErrInvalidDocumentID
	}
	i.mu.Lock()
	delete(i.docs, id)
	i.mu.Unlock()
	return nil
}

// Get retrieves a document by ID.
func (i *InMemoryIndex) Get(_ context.Context, id string) (Document, bool) {
	i.mu.RLock()
	doc, ok := i.docs[id]
	i.mu.RUnlock()
	return doc, ok
}

// List returns documents sorted by ID for deterministic ordering.
func (i *InMemoryIndex) List(_ context.Context) []Document {
	i.mu.RLock()
	ids := make([]string, 0, len(i.docs))
	for id := range i.docs {
		ids = append(ids, id)
	}
	// Copy docs after collecting IDs to minimize lock duration.
	i.mu.RUnlock()

	sort.Strings(ids)

	out := make([]Document, 0, len(ids))
	i.mu.RLock()
	for _, id := range ids {
		out = append(out, i.docs[id])
	}
	i.mu.RUnlock()
	return out
}

var _ Indexer = (*InMemoryIndex)(nil)
