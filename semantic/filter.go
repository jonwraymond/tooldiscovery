package semantic

import "strings"

// FilterByNamespace returns documents matching the namespace.
func FilterByNamespace(docs []Document, namespace string) []Document {
	if namespace == "" {
		return nil
	}

	out := make([]Document, 0, len(docs))
	for _, doc := range docs {
		if doc.Namespace == namespace {
			out = append(out, doc)
		}
	}
	return out
}

// FilterByTags returns documents that contain any of the provided tags.
func FilterByTags(docs []Document, tags []string) []Document {
	if len(tags) == 0 {
		return nil
	}

	want := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		t := strings.ToLower(strings.TrimSpace(tag))
		if t == "" {
			continue
		}
		want[t] = struct{}{}
	}
	if len(want) == 0 {
		return nil
	}

	out := make([]Document, 0, len(docs))
	for _, doc := range docs {
		for _, tag := range doc.Tags {
			t := strings.ToLower(strings.TrimSpace(tag))
			if _, ok := want[t]; ok {
				out = append(out, doc)
				break
			}
		}
	}
	return out
}

// FilterByCategory returns documents matching the category.
func FilterByCategory(docs []Document, category string) []Document {
	if category == "" {
		return nil
	}

	cat := strings.ToLower(strings.TrimSpace(category))
	out := make([]Document, 0, len(docs))
	for _, doc := range docs {
		if strings.ToLower(strings.TrimSpace(doc.Category)) == cat {
			out = append(out, doc)
		}
	}
	return out
}
