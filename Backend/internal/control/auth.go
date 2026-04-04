package control

import (
	"errors"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type authClaims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (a *App) issueToken(userID, role string) (string, error) {
	claims := authClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(a.cfg.JWTSecret))
}

func (a *App) parseToken(raw string) (*authClaims, error) {
	tok, err := jwt.ParseWithClaims(raw, &authClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected jwt signing method")
		}
		return []byte(a.cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*authClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (a *App) authMiddleware(c *fiber.Ctx) error {
	header := c.Get("Authorization")
	if len(header) < 8 || header[:7] != "Bearer " {
		return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
	}
	claims, err := a.parseToken(header[7:])
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid bearer token")
	}
	c.Locals("uid", claims.UserID)
	c.Locals("role", claims.Role)
	return c.Next()
}

func (a *App) requireRoles(roles ...string) fiber.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *fiber.Ctx) error {
		role, _ := c.Locals("role").(string)
		if _, ok := allowed[role]; !ok {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		return c.Next()
	}
}

func isPrivilegedRole(role string) bool {
	return role == "admin" || role == "operator"
}

func isAdminRole(role string) bool {
	return role == "admin"
}

func (a *App) userAccessibleWorkplaceIDs(uid string) ([]uint, error) {
	if uid == "" {
		return nil, nil
	}

	idSet := map[uint]struct{}{}

	var ownedWorkplaceIDs []uint
	if err := a.db.Model(&common.Workplace{}).Where("created_by = ?", uid).Pluck("id", &ownedWorkplaceIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range ownedWorkplaceIDs {
		idSet[id] = struct{}{}
	}

	var teamIDs []uint
	if err := a.db.Model(&common.TeamMember{}).Where("user_id = ?", uid).Pluck("team_id", &teamIDs).Error; err != nil {
		return nil, err
	}
	if len(teamIDs) > 0 {
		var linkedWorkplaceIDs []uint
		if err := a.db.Model(&common.WorkplaceTeam{}).Where("team_id IN ?", teamIDs).Pluck("workplace_id", &linkedWorkplaceIDs).Error; err != nil {
			return nil, err
		}
		for _, id := range linkedWorkplaceIDs {
			idSet[id] = struct{}{}
		}
	}

	if len(idSet) == 0 {
		return nil, nil
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids, nil
}

func (a *App) canAccessProjectForUser(uid, role string, project common.Project) (bool, error) {
	if isPrivilegedRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if uid == project.CreatedBy {
		return true, nil
	}
	if project.WorkplaceID == nil {
		return false, nil
	}
	return a.userCanAccessWorkplace(uid, *project.WorkplaceID)
}

func (a *App) canAccessProject(c *fiber.Ctx, project common.Project) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	return a.canAccessProjectForUser(uid, role, project)
}

func (a *App) canWriteProjectForUser(uid, role string, project common.Project) (bool, error) {
	if isAdminRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if uid == project.CreatedBy {
		return true, nil
	}
	if project.WorkplaceID == nil {
		return false, nil
	}
	return a.userCanAccessWorkplace(uid, *project.WorkplaceID)
}

func (a *App) canWriteProject(c *fiber.Ctx, project common.Project) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	return a.canWriteProjectForUser(uid, role, project)
}

func (a *App) canBindWorkplace(c *fiber.Ctx, workplaceID uint) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isAdminRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	return a.userCanAccessWorkplace(uid, workplaceID)
}

func (a *App) canManageTeam(c *fiber.Ctx, team common.Team) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isAdminRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if team.CreatedBy == uid {
		return true, nil
	}

	var membership common.TeamMember
	if err := a.db.Where("team_id = ? AND user_id = ?", team.ID, uid).First(&membership).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return membership.Role == "owner", nil
}

func (a *App) canManageTeamByID(c *fiber.Ctx, teamID uint) (bool, error) {
	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return a.canManageTeam(c, team)
}

func (a *App) canManageWorkplace(c *fiber.Ctx, workplace common.Workplace) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isAdminRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if workplace.CreatedBy == uid {
		return true, nil
	}

	teamIDs, err := teamIDsForWorkplace(a.db, workplace.ID)
	if err != nil {
		return false, err
	}
	for _, teamID := range teamIDs {
		allowed, err := a.canManageTeamByID(c, teamID)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (a *App) canAccessTask(c *fiber.Ctx, task common.Task) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isPrivilegedRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if uid == task.CreatedBy {
		return true, nil
	}

	var project common.Project
	if err := a.db.Where("id = ?", task.ProjectID).First(&project).Error; err != nil {
		return false, nil
	}
	return a.canAccessProjectForUser(uid, role, project)
}

func (a *App) canWriteTask(c *fiber.Ctx, task common.Task) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isAdminRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if uid == task.CreatedBy {
		return true, nil
	}

	var project common.Project
	if err := a.db.Where("id = ?", task.ProjectID).First(&project).Error; err != nil {
		return false, nil
	}
	return a.canWriteProjectForUser(uid, role, project)
}

func (a *App) canAccessSchedule(c *fiber.Ctx, schedule common.Schedule) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isPrivilegedRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if uid == schedule.CreatedBy {
		return true, nil
	}

	var project common.Project
	if err := a.db.Where("id = ?", schedule.ProjectID).First(&project).Error; err != nil {
		return false, nil
	}
	return a.canAccessProjectForUser(uid, role, project)
}

func (a *App) canWriteSchedule(c *fiber.Ctx, schedule common.Schedule) (bool, error) {
	uid, _ := c.Locals("uid").(string)
	role, _ := c.Locals("role").(string)
	if isAdminRole(role) {
		return true, nil
	}
	if uid == "" {
		return false, nil
	}
	if uid == schedule.CreatedBy {
		return true, nil
	}

	var project common.Project
	if err := a.db.Where("id = ?", schedule.ProjectID).First(&project).Error; err != nil {
		return false, nil
	}
	return a.canWriteProjectForUser(uid, role, project)
}
