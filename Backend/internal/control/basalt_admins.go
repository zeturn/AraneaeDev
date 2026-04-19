package control

import "strings"

func parseBasaltAdminEmails(raw string) map[string]struct{} {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	out := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		email := strings.ToLower(strings.TrimSpace(part))
		if email == "" {
			continue
		}
		out[email] = struct{}{}
	}
	return out
}

func (a *App) isBasaltAdminEmail(email string) bool {
	if a == nil {
		return false
	}
	trimmed := strings.ToLower(strings.TrimSpace(email))
	if trimmed == "" {
		return false
	}
	admins := parseBasaltAdminEmails(a.cfg.BasaltAdminEmails)
	_, ok := admins[trimmed]
	return ok
}
