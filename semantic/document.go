package semantic

import (
	"sort"
	"strings"
)

// Document describes a tool for semantic indexing.
type Document struct {
	ID          string
	Namespace   string
	Name        string
	Description string
	Tags        []string
	Category    string
	Text        string // normalized combined text
}

// Normalized returns a copy of the document with normalized tags and text.
func (d Document) Normalized() Document {
	norm := d

	tags := normalizeTags(d.Tags)
	norm.Tags = tags

	parts := make([]string, 0, 2+len(tags))
	if name := strings.TrimSpace(d.Name); name != "" {
		parts = append(parts, name)
	}
	if desc := strings.TrimSpace(d.Description); desc != "" {
		parts = append(parts, desc)
	}
	for _, tag := range tags {
		if tag != "" {
			parts = append(parts, tag)
		}
	}

	norm.Text = strings.Join(parts, " ")
	return norm
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}

	norm := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.ToLower(strings.TrimSpace(tag))
		if t == "" {
			continue
		}
		norm = append(norm, t)
	}

	sort.Strings(norm)
	return norm
}
