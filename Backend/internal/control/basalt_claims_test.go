package control

import (
	"testing"

	"araneae-go/internal/common"
)

func TestBasaltRoleFromIdentityFallsBackToClaimRole(t *testing.T) {
	app := newTestControlApp(t)

	role := app.basaltRoleFromIdentity("", map[string]any{"roles": []any{"operator"}})
	if role != "operator" {
		t.Fatalf("expected operator from claims, got %q", role)
	}
}

func TestBasaltRoleFromIdentityPrefersScopeWhenPresent(t *testing.T) {
	app := newTestControlApp(t)

	role := app.basaltRoleFromIdentity("araneae.read", map[string]any{"roles": []any{"admin"}})
	if role != "viewer" {
		t.Fatalf("expected viewer from scope, got %q", role)
	}
}

func TestFindOrCreateBasaltUserSyncsGroupsIntoTeams(t *testing.T) {
	app := newTestControlApp(t)
	app.cfg.BasaltTeamSyncEnabled = true
	app.cfg.BasaltTeamPrefix = "Basalt::"

	claims := map[string]any{
		"role":   "operator",
		"groups": []any{"engineering", "ops"},
	}
	user, err := app.findOrCreateBasaltUser("subject-sync-1", "", claims)
	if err != nil {
		t.Fatalf("findOrCreateBasaltUser failed: %v", err)
	}
	if user.Role != "operator" {
		t.Fatalf("unexpected user role: %s", user.Role)
	}

	teamNames := []string{"Basalt::engineering", "Basalt::ops"}
	var teams []common.Team
	if err := app.db.Where("name IN ?", teamNames).Find(&teams).Error; err != nil {
		t.Fatalf("load teams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("expected 2 synced teams, got %d", len(teams))
	}

	for _, team := range teams {
		var count int64
		if err := app.db.Model(&common.TeamMember{}).Where("team_id = ? AND user_id = ?", team.ID, user.ID).Count(&count).Error; err != nil {
			t.Fatalf("count team membership: %v", err)
		}
		if count != 1 {
			t.Fatalf("expected membership for team %s", team.Name)
		}
	}
}

func TestSyncBasaltGroupsPrunesStaleMembershipsWhenEnabled(t *testing.T) {
	app := newTestControlApp(t)
	app.cfg.BasaltTeamSyncEnabled = true
	app.cfg.BasaltTeamSyncPrune = true
	app.cfg.BasaltTeamPrefix = "Basalt::"

	firstClaims := map[string]any{
		"role":   "operator",
		"groups": []any{"engineering", "ops"},
	}
	user, err := app.findOrCreateBasaltUser("subject-sync-prune", "", firstClaims)
	if err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	secondClaims := map[string]any{
		"role":   "operator",
		"groups": []any{"engineering"},
	}
	if _, err := app.findOrCreateBasaltUser("subject-sync-prune", "", secondClaims); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	assertMembershipCount := func(teamName string, want int64) {
		t.Helper()
		var team common.Team
		if err := app.db.Where("name = ?", teamName).First(&team).Error; err != nil {
			t.Fatalf("load team %s: %v", teamName, err)
		}
		var count int64
		if err := app.db.Model(&common.TeamMember{}).Where("team_id = ? AND user_id = ?", team.ID, user.ID).Count(&count).Error; err != nil {
			t.Fatalf("count membership for %s: %v", teamName, err)
		}
		if count != want {
			t.Fatalf("membership mismatch for %s: got=%d want=%d", teamName, count, want)
		}
	}

	assertMembershipCount("Basalt::engineering", 1)
	assertMembershipCount("Basalt::ops", 0)
}

func TestSyncBasaltGroupsKeepsStaleMembershipsWhenPruneDisabled(t *testing.T) {
	app := newTestControlApp(t)
	app.cfg.BasaltTeamSyncEnabled = true
	app.cfg.BasaltTeamSyncPrune = false
	app.cfg.BasaltTeamPrefix = "Basalt::"

	firstClaims := map[string]any{
		"role":   "operator",
		"groups": []any{"engineering", "ops"},
	}
	user, err := app.findOrCreateBasaltUser("subject-sync-keep", "", firstClaims)
	if err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}

	secondClaims := map[string]any{
		"role":   "operator",
		"groups": []any{"engineering"},
	}
	if _, err := app.findOrCreateBasaltUser("subject-sync-keep", "", secondClaims); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	var opsTeam common.Team
	if err := app.db.Where("name = ?", "Basalt::ops").First(&opsTeam).Error; err != nil {
		t.Fatalf("load team Basalt::ops: %v", err)
	}
	var count int64
	if err := app.db.Model(&common.TeamMember{}).Where("team_id = ? AND user_id = ?", opsTeam.ID, user.ID).Count(&count).Error; err != nil {
		t.Fatalf("count membership: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected stale membership kept when prune disabled, got %d", count)
	}
}
