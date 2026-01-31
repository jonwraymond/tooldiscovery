package semantic

import (
	"context"
	"testing"
)

type scoreByIDStrategy struct {
	scores map[string]float64
}

func (s scoreByIDStrategy) Score(_ context.Context, _ string, doc Document) (float64, error) {
	return s.scores[doc.ID], nil
}

type constStrategy struct {
	score float64
}

func (s constStrategy) Score(_ context.Context, _ string, _ Document) (float64, error) {
	return s.score, nil
}

func TestSearcher_DeterministicOrdering(t *testing.T) {
	idx := NewInMemoryIndex()
	ctx := context.Background()

	_ = idx.Add(ctx, Document{ID: "b", Name: "B"})
	_ = idx.Add(ctx, Document{ID: "a", Name: "A"})
	_ = idx.Add(ctx, Document{ID: "c", Name: "C"})

	strategy := scoreByIDStrategy{scores: map[string]float64{
		"a": 2,
		"b": 3,
		"c": 1,
	}}

	searcher := NewSearcher(idx, strategy)

	results1, err := searcher.Search(ctx, "query")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	results2, err := searcher.Search(ctx, "query")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	want := []string{"b", "a", "c"}
	if len(results1) != len(want) {
		t.Fatalf("results length = %d, want %d", len(results1), len(want))
	}
	for i := range want {
		if results1[i].Document.ID != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, results1[i].Document.ID, want[i])
		}
		if results2[i].Document.ID != want[i] {
			t.Fatalf("repeat order[%d] = %q, want %q", i, results2[i].Document.ID, want[i])
		}
	}
}

func TestSearcher_TieBreakByID(t *testing.T) {
	idx := NewInMemoryIndex()
	ctx := context.Background()

	_ = idx.Add(ctx, Document{ID: "b"})
	_ = idx.Add(ctx, Document{ID: "a"})
	_ = idx.Add(ctx, Document{ID: "c"})

	searcher := NewSearcher(idx, constStrategy{score: 1.0})
	results, err := searcher.Search(ctx, "query")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	want := []string{"a", "b", "c"}
	for i := range want {
		if results[i].Document.ID != want[i] {
			t.Fatalf("order[%d] = %q, want %q", i, results[i].Document.ID, want[i])
		}
	}
}
