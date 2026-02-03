package search

import (
	"testing"

	"github.com/jonwraymond/tooldiscovery/index"
)

func TestFingerprint_SameDocsProduceSameFingerprint(t *testing.T) {
	docs := []index.SearchDoc{
		{
			ID:      "tool-1",
			DocText: "description one",
			Summary: index.Summary{ID: "tool-1", Name: "Tool1", Namespace: "ns1"},
		},
		{
			ID:      "tool-2",
			DocText: "description two",
			Summary: index.Summary{ID: "tool-2", Name: "Tool2", Namespace: "ns2"},
		},
	}

	fp1 := computeFingerprint(docs)
	fp2 := computeFingerprint(docs)

	if fp1 != fp2 {
		t.Errorf("same docs produced different fingerprints: %s vs %s", fp1, fp2)
	}
	if fp1 == "" {
		t.Error("fingerprint is empty")
	}
}

func TestFingerprint_DifferentDocsProduceDifferentFingerprint(t *testing.T) {
	docs1 := []index.SearchDoc{
		{ID: "tool-1", DocText: "description one"},
	}
	docs2 := []index.SearchDoc{
		{ID: "tool-2", DocText: "description two"},
	}

	fp1 := computeFingerprint(docs1)
	fp2 := computeFingerprint(docs2)

	if fp1 == fp2 {
		t.Error("different docs produced same fingerprint")
	}
}

func TestFingerprint_OrderMatters(t *testing.T) {
	doc1 := index.SearchDoc{ID: "tool-1", DocText: "one"}
	doc2 := index.SearchDoc{ID: "tool-2", DocText: "two"}

	fp1 := computeFingerprint([]index.SearchDoc{doc1, doc2})
	fp2 := computeFingerprint([]index.SearchDoc{doc2, doc1})

	if fp1 == fp2 {
		t.Error("different order should produce different fingerprints")
	}
}

func TestFingerprint_IncludesAllFields(t *testing.T) {
	base := index.SearchDoc{
		ID:      "tool-1",
		DocText: "description",
		Summary: index.Summary{
			ID:               "tool-1",
			Name:             "Tool1",
			Namespace:        "ns1",
			ShortDescription: "short desc",
			Summary:          "summary text",
			Category:         "category",
			InputModes:       []string{"application/json"},
			OutputModes:      []string{"application/json"},
			SecuritySummary:  "apiKey",
			Tags:             []string{"tag1", "tag2"},
		},
	}

	// Each variation should produce a different fingerprint
	variations := []index.SearchDoc{
		{ID: "tool-1-changed", DocText: base.DocText, Summary: base.Summary},
		{ID: base.ID, DocText: "changed", Summary: base.Summary},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             "ChangedName",
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        "changed-ns",
				ShortDescription: base.Summary.ShortDescription,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: "changed short desc",
				Summary:          base.Summary.Summary,
				Category:         base.Summary.Category,
				InputModes:       base.Summary.InputModes,
				OutputModes:      base.Summary.OutputModes,
				SecuritySummary:  base.Summary.SecuritySummary,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Summary:          "changed summary",
				Category:         base.Summary.Category,
				InputModes:       base.Summary.InputModes,
				OutputModes:      base.Summary.OutputModes,
				SecuritySummary:  base.Summary.SecuritySummary,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Summary:          base.Summary.Summary,
				Category:         "changed-category",
				InputModes:       base.Summary.InputModes,
				OutputModes:      base.Summary.OutputModes,
				SecuritySummary:  base.Summary.SecuritySummary,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Summary:          base.Summary.Summary,
				Category:         base.Summary.Category,
				InputModes:       []string{"text/plain"},
				OutputModes:      base.Summary.OutputModes,
				SecuritySummary:  base.Summary.SecuritySummary,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Summary:          base.Summary.Summary,
				Category:         base.Summary.Category,
				InputModes:       base.Summary.InputModes,
				OutputModes:      []string{"text/plain"},
				SecuritySummary:  base.Summary.SecuritySummary,
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Summary:          base.Summary.Summary,
				Category:         base.Summary.Category,
				InputModes:       base.Summary.InputModes,
				OutputModes:      base.Summary.OutputModes,
				SecuritySummary:  "oauth2",
				Tags:             base.Summary.Tags,
			},
		},
		{
			ID:      base.ID,
			DocText: base.DocText,
			Summary: index.Summary{
				ID:               base.Summary.ID,
				Name:             base.Summary.Name,
				Namespace:        base.Summary.Namespace,
				ShortDescription: base.Summary.ShortDescription,
				Summary:          base.Summary.Summary,
				Category:         base.Summary.Category,
				InputModes:       base.Summary.InputModes,
				OutputModes:      base.Summary.OutputModes,
				SecuritySummary:  base.Summary.SecuritySummary,
				Tags:             []string{"different-tag"},
			},
		},
	}

	baseFP := computeFingerprint([]index.SearchDoc{base})

	for i, v := range variations {
		vFP := computeFingerprint([]index.SearchDoc{v})
		if vFP == baseFP {
			t.Errorf("variation %d should produce different fingerprint from base", i)
		}
	}
}

func TestFingerprint_TagOrderIndependent(t *testing.T) {
	// Same tags in different orders should produce same fingerprint
	doc1 := index.SearchDoc{
		ID:      "tool-1",
		DocText: "description",
		Summary: index.Summary{
			ID:   "tool-1",
			Name: "Tool1",
			Tags: []string{"alpha", "bravo", "charlie"},
		},
	}
	doc2 := index.SearchDoc{
		ID:      "tool-1",
		DocText: "description",
		Summary: index.Summary{
			ID:   "tool-1",
			Name: "Tool1",
			Tags: []string{"charlie", "alpha", "bravo"},
		},
	}

	fp1 := computeFingerprint([]index.SearchDoc{doc1})
	fp2 := computeFingerprint([]index.SearchDoc{doc2})

	if fp1 != fp2 {
		t.Errorf("same tags in different order should produce same fingerprint: %s vs %s", fp1, fp2)
	}
}

func TestFingerprint_EmptyDocs(t *testing.T) {
	var docs []index.SearchDoc
	fp := computeFingerprint(docs)

	// Should return a consistent fingerprint for empty docs
	fp2 := computeFingerprint(nil)
	if fp != fp2 {
		t.Error("empty slice and nil should produce same fingerprint")
	}
	if fp == "" {
		t.Error("fingerprint should not be empty for empty docs")
	}
}
