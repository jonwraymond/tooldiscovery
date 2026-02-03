package tooldoc

import (
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func stringSliceFromAny(v any) []string {
	switch t := v.(type) {
	case []string:
		out := make([]string, len(t))
		copy(out, t)
		return out
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func securitySummaryFromMeta(meta mcp.Meta) string {
	if meta == nil {
		return ""
	}
	schemes := schemeNamesFromRequirements(meta["securityRequirements"])
	if len(schemes) == 0 {
		schemes = schemeNamesFromSchemes(meta["securitySchemes"])
	}
	if len(schemes) == 0 {
		return ""
	}
	sort.Strings(schemes)
	return strings.Join(schemes, ",")
}

func schemeNamesFromRequirements(raw any) []string {
	switch reqs := raw.(type) {
	case []map[string][]string:
		out := make([]string, 0, len(reqs))
		for _, req := range reqs {
			for name := range req {
				out = append(out, name)
			}
		}
		return out
	case []map[string]any:
		out := make([]string, 0, len(reqs))
		for _, req := range reqs {
			for name := range req {
				out = append(out, name)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(reqs))
		for _, item := range reqs {
			if reqMap, ok := item.(map[string]any); ok {
				for name := range reqMap {
					out = append(out, name)
				}
			}
			if reqMap, ok := item.(map[string][]string); ok {
				for name := range reqMap {
					out = append(out, name)
				}
			}
		}
		return out
	default:
		return nil
	}
}

func schemeNamesFromSchemes(raw any) []string {
	switch schemes := raw.(type) {
	case map[string]any:
		out := make([]string, 0, len(schemes))
		for name := range schemes {
			out = append(out, name)
		}
		return out
	case map[string]map[string]any:
		out := make([]string, 0, len(schemes))
		for name := range schemes {
			out = append(out, name)
		}
		return out
	default:
		return nil
	}
}

func annotationsFromTool(ann *mcp.ToolAnnotations) map[string]any {
	if ann == nil {
		return nil
	}
	out := map[string]any{}
	if ann.DestructiveHint != nil {
		out["destructiveHint"] = *ann.DestructiveHint
	}
	if ann.OpenWorldHint != nil {
		out["openWorldHint"] = *ann.OpenWorldHint
	}
	out["idempotentHint"] = ann.IdempotentHint
	out["readOnlyHint"] = ann.ReadOnlyHint
	if ann.Title != "" {
		out["title"] = ann.Title
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
