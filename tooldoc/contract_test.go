package tooldoc

import (
	"errors"
	"testing"
)

func TestStoreContract_Errors(t *testing.T) {
	store := NewInMemoryStore(StoreOptions{})

	_, err := store.DescribeTool("missing:tool", DetailSummary)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("DescribeTool error = %v, want ErrNotFound", err)
	}

	_, err = store.DescribeTool("missing:tool", DetailLevel("invalid"))
	if !errors.Is(err, ErrInvalidDetail) {
		t.Fatalf("DescribeTool error = %v, want ErrInvalidDetail", err)
	}

	if err := store.RegisterDoc("docs:only", DocEntry{Summary: "docs"}); err != nil {
		t.Fatalf("RegisterDoc failed: %v", err)
	}
	_, err = store.DescribeTool("docs:only", DetailSchema)
	if !errors.Is(err, ErrNoTool) {
		t.Fatalf("DescribeTool error = %v, want ErrNoTool", err)
	}

	_, err = store.ListExamples("missing:tool", 1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ListExamples error = %v, want ErrNotFound", err)
	}
}
