package control

import "strings"

func normalizeScopes(scopeRaw string) string {
	if strings.TrimSpace(scopeRaw) == "" {
		return ""
	}
	parts := strings.Fields(scopeRaw)
	seen := make(map[string]struct{}, len(parts))
	ordered := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(strings.ToLower(part))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		ordered = append(ordered, p)
	}
	return strings.Join(ordered, " ")
}

func hasScope(scopeRaw, requiredScope string) bool {
	required := strings.TrimSpace(strings.ToLower(requiredScope))
	if required == "" {
		return true
	}
	normalized := normalizeScopes(scopeRaw)
	if normalized == "" {
		return false
	}
	for _, scope := range strings.Fields(normalized) {
		if scope == required || scope == "*" || scope == "araneae.*" {
			return true
		}
	}
	return false
}

func defaultScopesForRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return "araneae.admin araneae.write araneae.read"
	case "operator":
		return "araneae.write araneae.read"
	case "viewer":
		fallthrough
	default:
		return "araneae.read"
	}
}

func roleFromScope(scopeRaw string) string {
	normalized := normalizeScopes(scopeRaw)
	if hasScope(normalized, "araneae.admin") {
		return "admin"
	}
	if hasScope(normalized, "araneae.write") {
		return "operator"
	}
	if hasScope(normalized, "araneae.read") {
		return "viewer"
	}
	return "viewer"
}
