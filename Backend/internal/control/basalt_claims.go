package control

import (
	"errors"
	"strings"
	"time"

	"araneae-go/internal/common"

	"gorm.io/gorm"
)

var (
	defaultBasaltRoleClaimKeys  = []string{"roles", "role", "app_roles"}
	defaultBasaltGroupClaimKeys = []string{"groups", "group", "teams", "team"}
)

func parseClaimKeys(raw string, fallback []string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	seen := make(map[string]struct{}, len(parts))
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	if len(out) == 0 {
		return fallback
	}
	return out
}

func claimValueStrings(payload map[string]any, keys []string) []string {
	if len(keys) == 0 || payload == nil {
		return nil
	}

	collect := func(source map[string]any, acc *[]string) {
		for _, key := range keys {
			if raw, ok := source[key]; ok {
				*acc = append(*acc, normalizeClaimValue(raw)...)
			}
		}
	}

	values := make([]string, 0)
	collect(payload, &values)
	for _, nestedKey := range []string{"data", "result", "payload"} {
		nested, ok := payload[nestedKey].(map[string]any)
		if !ok {
			continue
		}
		collect(nested, &values)
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func normalizeClaimValue(raw any) []string {
	switch v := raw.(type) {
	case string:
		return splitClaimString(v)
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, splitClaimString(item)...)
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, splitClaimString(s)...)
		}
		return out
	default:
		return nil
	}
}

func splitClaimString(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func roleFromClaims(payload map[string]any, claimKeys []string) string {
	for _, raw := range claimValueStrings(payload, claimKeys) {
		value := strings.ToLower(strings.TrimSpace(raw))
		switch value {
		case "admin", "owner", "superadmin", "araneae_admin", "araneae.admin":
			return "admin"
		case "operator", "editor", "developer", "maintainer", "araneae_editor", "araneae_operator", "araneae.write":
			return "operator"
		case "viewer", "readonly", "read_only", "araneae_viewer", "araneae.read":
			return "viewer"
		}
	}
	return ""
}

func mergeClaims(first, second map[string]any) map[string]any {
	if first == nil && second == nil {
		return nil
	}
	merged := make(map[string]any)
	for k, v := range first {
		merged[k] = v
	}
	for k, v := range second {
		merged[k] = v
	}
	return merged
}

func (a *App) basaltRoleFromIdentity(scopeRaw string, claims map[string]any) string {
	roleByScope := roleFromScope(scopeRaw)
	if normalizeScopes(scopeRaw) != "" {
		return roleByScope
	}
	claimRole := roleFromClaims(claims, parseClaimKeys(a.cfg.BasaltRoleClaimKeys, defaultBasaltRoleClaimKeys))
	if claimRole != "" {
		return claimRole
	}
	return roleByScope
}

func sanitizeBasaltGroupName(group string) string {
	trimmed := strings.TrimSpace(group)
	if trimmed == "" {
		return ""
	}
	// Keep group identifiers compact and printable for UI.
	if len(trimmed) > 80 {
		trimmed = trimmed[:80]
	}
	return trimmed
}

func (a *App) syncBasaltGroups(user common.User, claims map[string]any) error {
	if !a.cfg.BasaltTeamSyncEnabled || claims == nil {
		return nil
	}

	groups := claimValueStrings(claims, parseClaimKeys(a.cfg.BasaltGroupClaimKeys, defaultBasaltGroupClaimKeys))
	if len(groups) == 0 {
		return nil
	}

	prefix := strings.TrimSpace(a.cfg.BasaltTeamPrefix)
	if prefix == "" {
		prefix = "Basalt::"
	}

	now := time.Now()
	desiredTeamNames := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		safeGroup := sanitizeBasaltGroupName(group)
		if safeGroup == "" {
			continue
		}
		teamName := prefix + safeGroup
		desiredTeamNames[teamName] = struct{}{}

		var team common.Team
		err := a.db.Where("name = ?", teamName).First(&team).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			team = common.Team{
				Name:        teamName,
				Description: "Auto-synced from BasaltPass group " + safeGroup,
				JoinAble:    false,
				IsPersonal:  false,
				CreatedBy:   user.ID,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := a.db.Create(&team).Error; err != nil {
				return err
			}
		}

		var member common.TeamMember
		err = a.db.Where("team_id = ? AND user_id = ?", team.ID, user.ID).First(&member).Error
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err := a.db.Create(&common.TeamMember{
			TeamID:    team.ID,
			UserID:    user.ID,
			Role:      "member",
			CreatedAt: now,
		}).Error; err != nil {
			return err
		}
	}

	if !a.cfg.BasaltTeamSyncPrune {
		return nil
	}

	type membershipRow struct {
		TeamID uint
		Name   string
	}
	var rows []membershipRow
	if err := a.db.Table("team_members AS tm").
		Select("tm.team_id AS team_id, t.name AS name").
		Joins("JOIN teams t ON t.id = tm.team_id").
		Where("tm.user_id = ?", user.ID).
		Where("t.name LIKE ?", prefix+"%").
		Scan(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if _, keep := desiredTeamNames[row.Name]; keep {
			continue
		}
		if err := a.db.Where("team_id = ? AND user_id = ?", row.TeamID, user.ID).Delete(&common.TeamMember{}).Error; err != nil {
			return err
		}
	}

	return nil
}
