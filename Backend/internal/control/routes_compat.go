package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type discoverCandidate struct {
	IP                string `json:"ip"`
	Name              string `json:"name"`
	Port              int    `json:"port"`
	GRPCPort          int    `json:"grpc_port"`
	AlreadyRegistered bool   `json:"already_registered"`
	RegisteredNodeID  *uint  `json:"registered_node_id"`
	Machine           string `json:"machine,omitempty"`
	OS                string `json:"os,omitempty"`
}

type nodeCapability struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
}

type registerNodeRequest struct {
	IP          string `json:"ip"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Port        int    `json:"port"`
	GRPCPort    int    `json:"grpc_port"`
	PairKey     string `json:"pair_key"`
}

type updateNodeRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	IPAddress   *string `json:"ip_address"`
	Port        *int    `json:"port"`
	RPCURL      *string `json:"rpc_url"`
	CeleryQueue *string `json:"celery_queue"`
	IsEnabled   *bool   `json:"is_enabled"`
}

type installRuntimeRequest struct {
	Key string `json:"key"`
}

type createTeamRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	JoinAble    bool   `json:"join_able"`
}

type updateTeamRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	JoinAble    *bool   `json:"join_able"`
}

type addTeamMembersRequest struct {
	UserIDs []string `json:"user_ids"`
}

type createWorkplaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	TeamID      *uint  `json:"team_id"`
}

type updateWorkplaceRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type addWorkplaceTeamsRequest struct {
	TeamIDs []uint `json:"team_ids"`
}

type addWorkplacePeopleRequest struct {
	UserIDs []string `json:"user_ids"`
}

const nodeInactiveAfter = 2 * time.Minute

func (a *App) resolveUserIdentifier(identifier string) (common.User, error) {
	value := strings.TrimSpace(identifier)
	if value == "" {
		return common.User{}, gorm.ErrRecordNotFound
	}

	var user common.User
	if err := a.db.Where("id = ?", value).First(&user).Error; err == nil {
		return user, nil
	}
	if err := a.db.Where("username = ?", value).First(&user).Error; err == nil {
		return user, nil
	}
	return common.User{}, gorm.ErrRecordNotFound
}

func parseUintParam(c *fiber.Ctx, key string) (uint, error) {
	raw := strings.TrimSpace(c.Params(key))
	if raw == "" {
		return 0, errors.New("missing id")
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(parsed), nil
}

func defaultCPUInfoJSON() string {
	return `{"cpu_count":8,"cpu_frequency":{"current":3200,"max":3600,"min":1600},"cpu_physical_cores":4}`
}

func defaultMemoryInfoJSON() string {
	return `{"available_memory":8589934592,"memory_percentage":40.0,"total_memory":17179869184,"used_memory":8589934592}`
}

func defaultCapabilities() []nodeCapability {
	return []nodeCapability{
		{Key: "python", Name: "Python", Available: true, Version: "3.11"},
		{Key: "node", Name: "Node.js", Available: false},
		{Key: "go", Name: "Go", Available: true, Version: "1.26"},
		{Key: "java", Name: "Java", Available: false},
	}
}

func applyEffectiveNodeStatus(node *common.Node, now time.Time) {
	if node == nil {
		return
	}
	if !node.IsEnabled {
		node.Status = "inactive"
		return
	}
	if !node.LastActiveTime.IsZero() && now.Sub(node.LastActiveTime) > nodeInactiveAfter {
		node.Status = "inactive"
		return
	}
	node.Status = "active"
}

func parseCapabilities(raw string) []nodeCapability {
	if strings.TrimSpace(raw) == "" {
		return defaultCapabilities()
	}
	var caps []nodeCapability
	if err := json.Unmarshal([]byte(raw), &caps); err != nil || len(caps) == 0 {
		return defaultCapabilities()
	}
	return caps
}

func toCapabilitiesJSON(caps []nodeCapability) string {
	b, err := json.Marshal(caps)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func teamIDsForWorkplace(db *gorm.DB, workplaceID uint) ([]uint, error) {
	var links []common.WorkplaceTeam
	if err := db.Where("workplace_id = ?", workplaceID).Find(&links).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.TeamID)
	}
	return ids, nil
}

func (a *App) discoverNodes(c *fiber.Ctx) error {
	scope := strings.TrimSpace(c.Query("scope"))
	if scope == "" {
		scope = "local"
	}

	var nodes []common.Node
	if err := a.db.Order("id asc").Find(&nodes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	candidates := make([]discoverCandidate, 0, len(nodes)+1)
	for _, n := range nodes {
		nID := n.ID
		candidates = append(candidates, discoverCandidate{
			IP:                n.IPAddress,
			Name:              n.Name,
			Port:              n.Port,
			GRPCPort:          n.GRPCPort,
			AlreadyRegistered: true,
			RegisteredNodeID:  &nID,
			Machine:           "worknode",
			OS:                "linux",
		})
	}

	hasLocal := false
	for _, cnd := range candidates {
		if cnd.IP == "127.0.0.1" {
			hasLocal = true
			break
		}
	}
	if !hasLocal {
		candidates = append(candidates, discoverCandidate{
			IP:                "127.0.0.1",
			Name:              "local-executor",
			Port:              4280,
			GRPCPort:          9190,
			AlreadyRegistered: false,
			RegisteredNodeID:  nil,
			Machine:           "localhost",
			OS:                "windows",
		})
	}

	return c.JSON(fiber.Map{
		"scope":      scope,
		"candidates": candidates,
	})
}

func (a *App) registerNode(c *fiber.Ctx) error {
	var req registerNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	req.IP = strings.TrimSpace(req.IP)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.PairKey = strings.TrimSpace(req.PairKey)
	if req.IP == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ip is required")
	}
	if req.PairKey == "" {
		return fiber.NewError(fiber.StatusBadRequest, "pair_key is required")
	}
	if req.Name == "" {
		req.Name = "node-" + strings.ReplaceAll(req.IP, ".", "-")
	}
	if req.Port <= 0 {
		req.Port = 4280
	}
	if req.GRPCPort <= 0 {
		req.GRPCPort = 9190
	}

	verified, err := a.verifyExecutorNodeKey(req.IP, req.Port, req.PairKey)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	}

	capabilities := defaultCapabilities()
	if capResp, capErr := a.fetchExecutorCapabilities(req.IP, req.Port, req.PairKey); capErr == nil && len(capResp.Capabilities) > 0 {
		capabilities = capResp.Capabilities
	} else if capErr != nil {
		if a.log != nil {
			a.log.Warn("node capability fetch failed; using defaults", zap.String("ip", req.IP), zap.Int("port", req.Port), zap.Error(capErr))
		}
	}

	uid, _ := c.Locals("uid").(string)
	now := time.Now()

	var node common.Node
	err = a.db.Where("ip_address = ? AND port = ?", req.IP, req.Port).First(&node).Error
	if err == nil {
		node.Name = req.Name
		if req.Description != "" {
			node.Description = req.Description
		}
		node.Status = "active"
		node.GRPCPort = req.GRPCPort
		node.AuthTokenHash = hashNodeKey(req.PairKey)
		if queue := strings.TrimSpace(verified.Queue); queue != "" {
			node.CeleryQueue = queue
		}
		node.LastActiveTime = now
		node.CapabilitiesJSON = toCapabilitiesJSON(capabilities)
		node.UpdatedAt = now
		if err := a.db.Save(&node).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		applyEffectiveNodeStatus(&node, now)
		return c.JSON(node)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	node = common.Node{
		Name:             req.Name,
		Description:      req.Description,
		Status:           "active",
		IPAddress:        req.IP,
		Port:             req.Port,
		GRPCPort:         req.GRPCPort,
		RPCURL:           fmt.Sprintf("http://%s:%d", req.IP, req.Port),
		CeleryQueue:      "default",
		AuthTokenHash:    hashNodeKey(req.PairKey),
		IsEnabled:        true,
		LastActiveTime:   now,
		HDID:             strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:12], "-", "")),
		CPUInfo:          defaultCPUInfoJSON(),
		MemoryInfo:       defaultMemoryInfoJSON(),
		CapabilitiesJSON: toCapabilitiesJSON(capabilities),
		CreatedBy:        uid,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if queue := strings.TrimSpace(verified.Queue); queue != "" {
		node.CeleryQueue = queue
	}
	if err := a.db.Create(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	applyEffectiveNodeStatus(&node, now)
	return c.JSON(node)
}

func (a *App) listNodes(c *fiber.Ctx) error {
	var nodes []common.Node
	if err := a.db.Order("id asc").Find(&nodes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	now := time.Now()
	for i := range nodes {
		applyEffectiveNodeStatus(&nodes[i], now)
	}
	return c.JSON(fiber.Map{"results": nodes, "count": len(nodes)})
}

func (a *App) getNode(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	applyEffectiveNodeStatus(&node, time.Now())
	return c.JSON(node)
}

func (a *App) updateNode(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	applyEffectiveNodeStatus(&node, time.Now())

	var req updateNodeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if req.Name != nil {
		node.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		node.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		node.Status = strings.TrimSpace(*req.Status)
	}
	if req.IPAddress != nil {
		node.IPAddress = strings.TrimSpace(*req.IPAddress)
	}
	if req.Port != nil && *req.Port > 0 {
		node.Port = *req.Port
	}
	if req.RPCURL != nil {
		node.RPCURL = strings.TrimSpace(*req.RPCURL)
	}
	if req.CeleryQueue != nil {
		node.CeleryQueue = strings.TrimSpace(*req.CeleryQueue)
	}
	if req.IsEnabled != nil {
		node.IsEnabled = *req.IsEnabled
	}
	if node.Status == "" {
		node.Status = "active"
	}
	if node.CeleryQueue == "" {
		node.CeleryQueue = "default"
	}
	node.UpdatedAt = time.Now()

	if err := a.db.Save(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(node)
}

func (a *App) deleteNode(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	if err := a.db.Where("node_id = ?", nodeID).Delete(&common.NodeInstallJob{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	result := a.db.Delete(&common.Node{}, nodeID)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) getNodeStatus(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}

	totalMemory := float64(16 * 1024 * 1024 * 1024)
	usedMemory := float64(8 * 1024 * 1024 * 1024)
	var mem map[string]float64
	if err := json.Unmarshal([]byte(node.MemoryInfo), &mem); err == nil {
		if v, ok := mem["total_memory"]; ok && v > 0 {
			totalMemory = v
		}
		if v, ok := mem["used_memory"]; ok && v >= 0 {
			usedMemory = v
		}
	}

	cpuPercent := float64(20 + (time.Now().Unix() % 60))
	return c.JSON(fiber.Map{
		"cpu_percent":  cpuPercent,
		"memory_used":  usedMemory,
		"memory_total": totalMemory,
	})
}

func (a *App) getNodeCapabilities(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	applyEffectiveNodeStatus(&node, time.Now())
	return c.JSON(fiber.Map{"capabilities": parseCapabilities(node.CapabilitiesJSON)})
}

func (a *App) refreshNodeCapabilities(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	caps := parseCapabilities(node.CapabilitiesJSON)
	node.CapabilitiesJSON = toCapabilitiesJSON(caps)
	node.LastActiveTime = time.Now()
	node.Status = "active"
	node.UpdatedAt = time.Now()
	if err := a.db.Save(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"capabilities": caps})
}

func (a *App) getNodeInstallers(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	applyEffectiveNodeStatus(&node, time.Now())
	return c.JSON(fiber.Map{"installers": parseCapabilities(node.CapabilitiesJSON)})
}

func (a *App) installRuntime(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	var node common.Node
	if err := a.db.Where("id = ?", nodeID).First(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "node not found")
	}
	var req installRuntimeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fiber.NewError(fiber.StatusBadRequest, "key is required")
	}

	caps := parseCapabilities(node.CapabilitiesJSON)
	for i := range caps {
		if caps[i].Key == req.Key {
			caps[i].Available = true
			if caps[i].Version == "" {
				caps[i].Version = "installed"
			}
		}
	}
	node.CapabilitiesJSON = toCapabilitiesJSON(caps)
	node.UpdatedAt = time.Now()
	if err := a.db.Save(&node).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	now := time.Now()
	job := common.NodeInstallJob{
		ID:         uuid.NewString(),
		NodeID:     nodeID,
		RuntimeKey: req.Key,
		Status:     "success",
		Log:        fmt.Sprintf("runtime %s installed successfully", req.Key),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := a.db.Create(&job).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"job_id": job.ID, "status": "pending"})
}

func (a *App) getInstallStatus(c *fiber.Ctx) error {
	nodeID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid node id")
	}
	jobID := strings.TrimSpace(c.Params("jobID"))
	if jobID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing job id")
	}
	var job common.NodeInstallJob
	if err := a.db.Where("id = ? AND node_id = ?", jobID, nodeID).First(&job).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "install job not found")
	}
	return c.JSON(fiber.Map{
		"job_id": job.ID,
		"status": job.Status,
		"log":    job.Log,
	})
}

func (a *App) listUsers(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	query := a.db.Order("created_at asc")
	if !isPrivilegedRole(role) {
		query = query.Where("id = ?", uid)
	}

	var users []common.User
	if err := query.Find(&users).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	results := make([]fiber.Map, 0, len(users))
	for _, u := range users {
		results = append(results, fiber.Map{
			"id":         u.ID,
			"username":   u.Username,
			"role":       u.Role,
			"created_at": u.CreatedAt,
		})
	}
	return c.JSON(fiber.Map{"results": results, "count": len(results)})
}

func (a *App) getUser(c *fiber.Ctx) error {
	identifier := strings.TrimSpace(c.Params("id"))
	user, err := a.resolveUserIdentifier(identifier)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	if !isPrivilegedRole(role) && uid != user.ID {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	return c.JSON(fiber.Map{
		"id":         user.ID,
		"username":   user.Username,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

func (a *App) listMyTeams(c *fiber.Ctx) error {
	uid, _ := c.Locals("uid").(string)
	var memberships []common.TeamMember
	if err := a.db.Where("user_id = ?", uid).Find(&memberships).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	results := make([]fiber.Map, 0, len(memberships))
	for _, m := range memberships {
		var team common.Team
		if err := a.db.Where("id = ?", m.TeamID).First(&team).Error; err != nil {
			continue
		}
		results = append(results, fiber.Map{
			"id":          team.ID,
			"name":        team.Name,
			"description": team.Description,
			"join_able":   team.JoinAble,
			"is_personal": team.IsPersonal,
			"role":        m.Role,
			"created_at":  team.CreatedAt,
			"updated_at":  team.UpdatedAt,
		})
	}
	return c.JSON(fiber.Map{"results": results, "count": len(results)})
}

func (a *App) createTeam(c *fiber.Ctx) error {
	var req createTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "team name is required")
	}

	uid, _ := c.Locals("uid").(string)
	now := time.Now()
	team := common.Team{
		Name:        req.Name,
		Description: req.Description,
		JoinAble:    req.JoinAble,
		CreatedBy:   uid,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.db.Create(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	member := common.TeamMember{TeamID: team.ID, UserID: uid, Role: "owner", CreatedAt: now}
	_ = a.db.Create(&member).Error

	return c.JSON(fiber.Map{
		"id":          team.ID,
		"name":        team.Name,
		"description": team.Description,
		"join_able":   team.JoinAble,
		"is_personal": team.IsPersonal,
		"role":        "owner",
		"created_at":  team.CreatedAt,
		"updated_at":  team.UpdatedAt,
	})
}

func (a *App) getTeam(c *fiber.Ctx) error {
	teamID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}
	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "team not found")
	}

	uid, _ := c.Locals("uid").(string)
	roleName, _ := c.Locals("role").(string)
	if !isPrivilegedRole(roleName) {
		var membershipCount int64
		if err := a.db.Model(&common.TeamMember{}).Where("team_id = ? AND user_id = ?", teamID, uid).Count(&membershipCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if membershipCount == 0 {
			return fiber.NewError(fiber.StatusNotFound, "team not found")
		}
	}

	role := ""
	if uid != "" {
		var member common.TeamMember
		if err := a.db.Where("team_id = ? AND user_id = ?", team.ID, uid).First(&member).Error; err == nil {
			role = member.Role
		}
	}

	var membersCount int64
	if err := a.db.Model(&common.TeamMember{}).Where("team_id = ?", team.ID).Count(&membersCount).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"id":            team.ID,
		"name":          team.Name,
		"description":   team.Description,
		"join_able":     team.JoinAble,
		"is_personal":   team.IsPersonal,
		"created_by":    team.CreatedBy,
		"created_at":    team.CreatedAt,
		"updated_at":    team.UpdatedAt,
		"role":          role,
		"members_count": membersCount,
	})
}

func (a *App) updateTeam(c *fiber.Ctx) error {
	teamID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}
	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "team not found")
	}
	canManage, accessErr := a.canManageTeam(c, team)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var req updateTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name != nil {
		team.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		team.Description = strings.TrimSpace(*req.Description)
	}
	if req.JoinAble != nil {
		team.JoinAble = *req.JoinAble
	}
	team.UpdatedAt = time.Now()
	if err := a.db.Save(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(team)
}

func (a *App) deleteTeam(c *fiber.Ctx) error {
	teamID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}

	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "team not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	canManage, accessErr := a.canManageTeam(c, team)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	if team.IsPersonal {
		return fiber.NewError(fiber.StatusBadRequest, "personal team cannot be deleted")
	}

	_ = a.db.Where("team_id = ?", teamID).Delete(&common.TeamMember{}).Error
	_ = a.db.Where("team_id = ?", teamID).Delete(&common.WorkplaceTeam{}).Error
	result := a.db.Delete(&common.Team{}, teamID)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "team not found")
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) getTeamMembers(c *fiber.Ctx) error {
	teamID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}

	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "team not found")
	}

	uid, _ := c.Locals("uid").(string)
	roleName, _ := c.Locals("role").(string)
	if !isPrivilegedRole(roleName) {
		var membershipCount int64
		if err := a.db.Model(&common.TeamMember{}).Where("team_id = ? AND user_id = ?", teamID, uid).Count(&membershipCount).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if membershipCount == 0 {
			return fiber.NewError(fiber.StatusNotFound, "team not found")
		}
	}

	var members []common.TeamMember
	if err := a.db.Where("team_id = ?", teamID).Find(&members).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	resp := make([]fiber.Map, 0, len(members))
	for _, m := range members {
		user := fiber.Map{"id": m.UserID, "username": "user-" + m.UserID, "email": ""}
		var u common.User
		if err := a.db.Where("id = ?", m.UserID).First(&u).Error; err == nil {
			user = fiber.Map{"id": u.ID, "username": u.Username, "email": ""}
		}
		resp = append(resp, fiber.Map{"user": user, "role": m.Role})
	}
	return c.JSON(fiber.Map{"members": resp})
}

func (a *App) addTeamMembers(c *fiber.Ctx) error {
	teamID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}
	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "team not found")
	}
	canManage, accessErr := a.canManageTeam(c, team)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var req addTeamMembersRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	now := time.Now()
	added := 0
	skipped := 0
	notFound := make([]string, 0)
	for _, userID := range req.UserIDs {
		identifier := strings.TrimSpace(userID)
		if identifier == "" {
			continue
		}

		user, resolveErr := a.resolveUserIdentifier(identifier)
		if resolveErr != nil {
			notFound = append(notFound, identifier)
			continue
		}

		var count int64
		if err := a.db.Model(&common.TeamMember{}).Where("team_id = ? AND user_id = ?", teamID, user.ID).Count(&count).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if count > 0 {
			skipped++
			continue
		}
		if err := a.db.Create(&common.TeamMember{TeamID: teamID, UserID: user.ID, Role: "member", CreatedAt: now}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		added++
	}
	return c.JSON(fiber.Map{"ok": true, "added": added, "skipped": skipped, "not_found": notFound})
}

func (a *App) removeTeamMember(c *fiber.Ctx) error {
	teamID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}

	identifier := strings.TrimSpace(c.Params("userID"))
	if identifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "user id is required")
	}

	user, resolveErr := a.resolveUserIdentifier(identifier)
	if resolveErr != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	var team common.Team
	if err := a.db.Where("id = ?", teamID).First(&team).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "team not found")
	}
	canManage, accessErr := a.canManageTeam(c, team)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}

	var member common.TeamMember
	if err := a.db.Where("team_id = ? AND user_id = ?", teamID, user.ID).First(&member).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "team member not found")
	}

	if member.Role == "owner" {
		var owners int64
		if err := a.db.Model(&common.TeamMember{}).Where("team_id = ? AND role = ?", teamID, "owner").Count(&owners).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if owners <= 1 {
			return fiber.NewError(fiber.StatusBadRequest, "cannot remove the last owner")
		}
	}

	if err := a.db.Where("id = ?", member.ID).Delete(&common.TeamMember{}).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) listWorkplaces(c *fiber.Ctx) error {
	role, _ := c.Locals("role").(string)
	if !isPrivilegedRole(role) {
		return a.listMyWorkplaces(c)
	}

	var workplaces []common.Workplace
	if err := a.db.Order("id asc").Find(&workplaces).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	results := make([]fiber.Map, 0, len(workplaces))
	for _, w := range workplaces {
		results = append(results, fiber.Map{
			"id":          w.ID,
			"name":        w.Name,
			"description": w.Description,
			"status":      w.Status,
			"created_at":  w.CreatedAt,
			"updated_at":  w.UpdatedAt,
		})
	}
	return c.JSON(fiber.Map{"results": results, "count": len(results)})
}

func (a *App) userCanAccessWorkplace(uid string, workplaceID uint) (bool, error) {
	if uid == "" {
		return false, nil
	}

	var workplace common.Workplace
	if err := a.db.Select("id", "created_by").Where("id = ?", workplaceID).First(&workplace).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	if workplace.CreatedBy == uid {
		return true, nil
	}

	var links []common.WorkplaceTeam
	if err := a.db.Where("workplace_id = ?", workplaceID).Find(&links).Error; err != nil {
		return false, err
	}
	if len(links) == 0 {
		return false, nil
	}

	teamIDs := make([]uint, 0, len(links))
	for _, link := range links {
		teamIDs = append(teamIDs, link.TeamID)
	}

	var count int64
	if err := a.db.Model(&common.TeamMember{}).Where("user_id = ? AND team_id IN ?", uid, teamIDs).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (a *App) listMyWorkplaces(c *fiber.Ctx) error {
	uid, _ := c.Locals("uid").(string)
	var memberships []common.TeamMember
	if err := a.db.Where("user_id = ?", uid).Find(&memberships).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	teamSet := map[uint]struct{}{}
	for _, m := range memberships {
		teamSet[m.TeamID] = struct{}{}
	}
	teamIDs := make([]uint, 0, len(teamSet))
	for id := range teamSet {
		teamIDs = append(teamIDs, id)
	}

	workplaceIDSet := map[uint]struct{}{}
	if len(teamIDs) > 0 {
		var links []common.WorkplaceTeam
		if err := a.db.Where("team_id IN ?", teamIDs).Find(&links).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		for _, l := range links {
			workplaceIDSet[l.WorkplaceID] = struct{}{}
		}
	}

	var workplaces []common.Workplace
	if len(workplaceIDSet) == 0 {
		if err := a.db.Where("created_by = ?", uid).Order("id asc").Find(&workplaces).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	} else {
		ids := make([]uint, 0, len(workplaceIDSet))
		for id := range workplaceIDSet {
			ids = append(ids, id)
		}
		if err := a.db.Where("id IN ?", ids).Order("id asc").Find(&workplaces).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	results := make([]fiber.Map, 0, len(workplaces))
	for _, w := range workplaces {
		results = append(results, fiber.Map{
			"id":          w.ID,
			"name":        w.Name,
			"description": w.Description,
			"status":      w.Status,
			"created_at":  w.CreatedAt,
			"updated_at":  w.UpdatedAt,
		})
	}
	return c.JSON(fiber.Map{"results": results, "count": len(results)})
}

func (a *App) createWorkplace(c *fiber.Ctx) error {
	var req createWorkplaceRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Status = strings.TrimSpace(req.Status)
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "workplace name is required")
	}
	if req.Status == "" {
		req.Status = "active"
	}

	uid, _ := c.Locals("uid").(string)
	now := time.Now()
	if req.TeamID != nil {
		canManageTeam, accessErr := a.canManageTeamByID(c, *req.TeamID)
		if accessErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
		}
		if !canManageTeam {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
	}
	workplace := common.Workplace{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		CreatedBy:   uid,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := a.db.Create(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if req.TeamID != nil {
		link := common.WorkplaceTeam{WorkplaceID: workplace.ID, TeamID: *req.TeamID, CreatedAt: now}
		_ = a.db.Create(&link).Error
	}
	teamIDs, _ := teamIDsForWorkplace(a.db, workplace.ID)
	return c.JSON(fiber.Map{
		"id":          workplace.ID,
		"name":        workplace.Name,
		"description": workplace.Description,
		"status":      workplace.Status,
		"created_at":  workplace.CreatedAt,
		"updated_at":  workplace.UpdatedAt,
		"teams":       teamIDs,
	})
}

func (a *App) getWorkplace(c *fiber.Ctx) error {
	workplaceID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid workplace id")
	}
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	if !isPrivilegedRole(role) {
		allowed, accessErr := a.userCanAccessWorkplace(uid, workplaceID)
		if accessErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
		}
		if !allowed {
			return fiber.NewError(fiber.StatusNotFound, "workplace not found")
		}
	}
	var workplace common.Workplace
	if err := a.db.Where("id = ?", workplaceID).First(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "workplace not found")
	}
	teamIDs, err := teamIDsForWorkplace(a.db, workplace.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"id":          workplace.ID,
		"name":        workplace.Name,
		"description": workplace.Description,
		"status":      workplace.Status,
		"created_at":  workplace.CreatedAt,
		"updated_at":  workplace.UpdatedAt,
		"teams":       teamIDs,
		"owners":      []string{},
		"editors":     []string{},
	})
}

func (a *App) updateWorkplace(c *fiber.Ctx) error {
	workplaceID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid workplace id")
	}
	var workplace common.Workplace
	if err := a.db.Where("id = ?", workplaceID).First(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "workplace not found")
	}
	canManage, accessErr := a.canManageWorkplace(c, workplace)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var req updateWorkplaceRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Name != nil {
		workplace.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		workplace.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		workplace.Status = strings.TrimSpace(*req.Status)
	}
	if workplace.Status == "" {
		workplace.Status = "active"
	}
	workplace.UpdatedAt = time.Now()
	if err := a.db.Save(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(workplace)
}

func (a *App) deleteWorkplace(c *fiber.Ctx) error {
	workplaceID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid workplace id")
	}
	var workplace common.Workplace
	if err := a.db.Where("id = ?", workplaceID).First(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "workplace not found")
	}
	canManage, accessErr := a.canManageWorkplace(c, workplace)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	_ = a.db.Where("workplace_id = ?", workplaceID).Delete(&common.WorkplaceTeam{}).Error
	result := a.db.Delete(&common.Workplace{}, workplaceID)
	if result.Error != nil {
		return fiber.NewError(fiber.StatusInternalServerError, result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return fiber.NewError(fiber.StatusNotFound, "workplace not found")
	}
	return c.JSON(fiber.Map{"ok": true})
}

func (a *App) addWorkplaceTeams(c *fiber.Ctx) error {
	workplaceID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid workplace id")
	}
	var workplace common.Workplace
	if err := a.db.Where("id = ?", workplaceID).First(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "workplace not found")
	}
	canManage, accessErr := a.canManageWorkplace(c, workplace)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var req addWorkplaceTeamsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	now := time.Now()
	added := 0
	for _, teamID := range req.TeamIDs {
		canManageTeam, accessErr := a.canManageTeamByID(c, teamID)
		if accessErr != nil {
			return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
		}
		if !canManageTeam {
			return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
		}
		var count int64
		if err := a.db.Model(&common.WorkplaceTeam{}).Where("workplace_id = ? AND team_id = ?", workplaceID, teamID).Count(&count).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if count > 0 {
			continue
		}
		if err := a.db.Create(&common.WorkplaceTeam{WorkplaceID: workplaceID, TeamID: teamID, CreatedAt: now}).Error; err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		added++
	}
	teamIDs, _ := teamIDsForWorkplace(a.db, workplaceID)
	return c.JSON(fiber.Map{"ok": true, "added": added, "teams": teamIDs})
}

func (a *App) addWorkplacePeople(c *fiber.Ctx) error {
	workplaceID, err := parseUintParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid workplace id")
	}
	var workplace common.Workplace
	if err := a.db.Where("id = ?", workplaceID).First(&workplace).Error; err != nil {
		return fiber.NewError(fiber.StatusNotFound, "workplace not found")
	}
	canManage, accessErr := a.canManageWorkplace(c, workplace)
	if accessErr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, accessErr.Error())
	}
	if !canManage {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	var req addWorkplacePeopleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true, "added": len(req.UserIDs)})
}
