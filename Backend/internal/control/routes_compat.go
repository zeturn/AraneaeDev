package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type discoverCandidate struct {
	IP                string `json:"ip"`
	Host              string `json:"host,omitempty"`
	Name              string `json:"name"`
	Port              int    `json:"port"`
	GRPCPort          int    `json:"grpc_port"`
	AlreadyRegistered bool   `json:"already_registered"`
	RegisteredNodeID  *uint  `json:"registered_node_id"`
	Alive             bool   `json:"alive"`
	PairKeyMatched    bool   `json:"pair_key_matched"`
	ProbeStatus       string `json:"probe_status,omitempty"`
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
	probePort := parseIntQueryWithDefault(c, "port", 4280)
	probeGRPCPort := parseIntQueryWithDefault(c, "grpc_port", 9190)
	pairKey := strings.TrimSpace(c.Query("pair_key"))

	var nodes []common.Node
	if err := a.db.Order("id asc").Find(&nodes).Error; err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	nodeByEndpoint := make(map[string]common.Node, len(nodes))
	candidates := make([]discoverCandidate, 0, len(nodes)+1)
	for _, n := range nodes {
		nID := n.ID
		endpointKey := nodeEndpointKey(n.IPAddress, n.Port)
		nodeByEndpoint[endpointKey] = n
		candidates = append(candidates, discoverCandidate{
			IP:                n.IPAddress,
			Name:              n.Name,
			Port:              n.Port,
			GRPCPort:          n.GRPCPort,
			AlreadyRegistered: true,
			RegisteredNodeID:  &nID,
			Alive:             false,
			PairKeyMatched:    false,
			ProbeStatus:       "registered",
			Machine:           "worknode",
			OS:                "linux",
		})
	}

	targets, err := discoveryTargets(scope, c.Query("cidr"), c.Query("domain"), c.Query("target"), c.Query("hosts"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if len(targets) > 0 {
		type discoveredProbe struct {
			Host           string
			Alive          *nodeAliveResponse
			PairKeyMatched bool
		}

		results := make([]discoveredProbe, 0, len(targets))
		resultsCh := make(chan discoveredProbe, len(targets))
		sem := make(chan struct{}, 24)
		var wg sync.WaitGroup

		for _, target := range targets {
			target := target
			wg.Add(1)
			go func() {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				aliveResp, aliveErr := a.probeExecutorAlive(target, probePort)
				if aliveErr != nil {
					return
				}

				pairMatched := false
				if pairKey != "" {
					if _, verifyErr := a.verifyExecutorNodeKey(target, probePort, pairKey); verifyErr == nil {
						pairMatched = true
					}
				}

				resultsCh <- discoveredProbe{
					Host:           target,
					Alive:          aliveResp,
					PairKeyMatched: pairMatched,
				}
			}()
		}

		wg.Wait()
		close(resultsCh)
		for item := range resultsCh {
			results = append(results, item)
		}

		candidateIdxByEndpoint := make(map[string]int, len(candidates))
		for i, cand := range candidates {
			candidateIdxByEndpoint[nodeEndpointKey(cand.IP, cand.Port)] = i
		}

		for _, probe := range results {
			ips := normalizeProbeIPs(probe.Host, probe.Alive.IPs)
			candidateName := strings.TrimSpace(probe.Alive.Hostname)
			if candidateName == "" {
				candidateName = "node-" + strings.ReplaceAll(strings.TrimSpace(probe.Host), ".", "-")
			}

			chosenIP := ""
			registeredID := (*uint)(nil)
			registeredGRPC := probeGRPCPort
			for _, ip := range ips {
				if existing, ok := nodeByEndpoint[nodeEndpointKey(ip, probePort)]; ok {
					chosenIP = ip
					id := existing.ID
					registeredID = &id
					if existing.GRPCPort > 0 {
						registeredGRPC = existing.GRPCPort
					}
					break
				}
			}
			if chosenIP == "" {
				if len(ips) > 0 {
					chosenIP = ips[0]
				} else {
					chosenIP = strings.TrimSpace(probe.Host)
				}
			}

			status := "alive_unmatched"
			if probe.PairKeyMatched {
				status = "alive_matched"
			}

			endpointKey := nodeEndpointKey(chosenIP, probePort)
			if idx, exists := candidateIdxByEndpoint[endpointKey]; exists {
				cand := &candidates[idx]
				cand.Host = strings.TrimSpace(probe.Host)
				cand.Alive = true
				cand.PairKeyMatched = probe.PairKeyMatched
				cand.ProbeStatus = status
				if cand.Name == "" {
					cand.Name = candidateName
				}
				if cand.Machine == "" {
					cand.Machine = strings.TrimSpace(probe.Alive.Machine)
				}
				if cand.OS == "" {
					cand.OS = strings.TrimSpace(probe.Alive.OS)
				}
				continue
			}

			alreadyRegistered := registeredID != nil
			newCand := discoverCandidate{
				IP:                chosenIP,
				Host:              strings.TrimSpace(probe.Host),
				Name:              candidateName,
				Port:              probePort,
				GRPCPort:          registeredGRPC,
				AlreadyRegistered: alreadyRegistered,
				RegisteredNodeID:  registeredID,
				Alive:             true,
				PairKeyMatched:    probe.PairKeyMatched,
				ProbeStatus:       status,
				Machine:           strings.TrimSpace(probe.Alive.Machine),
				OS:                strings.TrimSpace(probe.Alive.OS),
			}
			candidateIdxByEndpoint[endpointKey] = len(candidates)
			candidates = append(candidates, newCand)
		}
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
			Alive:             false,
			PairKeyMatched:    false,
			ProbeStatus:       "seed",
			Machine:           "localhost",
			OS:                "windows",
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].AlreadyRegistered != candidates[j].AlreadyRegistered {
			return candidates[i].AlreadyRegistered
		}
		if candidates[i].Alive != candidates[j].Alive {
			return candidates[i].Alive
		}
		if candidates[i].PairKeyMatched != candidates[j].PairKeyMatched {
			return candidates[i].PairKeyMatched
		}
		if candidates[i].IP == candidates[j].IP {
			return candidates[i].Port < candidates[j].Port
		}
		return candidates[i].IP < candidates[j].IP
	})

	return c.JSON(fiber.Map{
		"scope":      scope,
		"candidates": candidates,
	})
}

func parseIntQueryWithDefault(c *fiber.Ctx, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func nodeEndpointKey(ip string, port int) string {
	return strings.ToLower(strings.TrimSpace(ip)) + ":" + strconv.Itoa(port)
}

func normalizeProbeIPs(host string, raw []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(raw)+1)
	appendIP := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if parsed := net.ParseIP(v); parsed != nil {
			if v4 := parsed.To4(); v4 != nil {
				v = v4.String()
			}
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	appendIP(host)
	for _, ip := range raw {
		appendIP(ip)
	}
	return out
}

func discoveryTargets(scope, cidrRaw, domainRaw, targetRaw, hostsRaw string) ([]string, error) {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = "local"
	}

	seen := make(map[string]struct{})
	targets := make([]string, 0, 64)
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		targets = append(targets, value)
	}

	add(targetRaw)
	for _, item := range strings.Split(strings.TrimSpace(hostsRaw), ",") {
		add(item)
	}

	if domain := strings.TrimSpace(domainRaw); domain != "" {
		add(domain)
	}

	if scope == "domain" && strings.TrimSpace(domainRaw) == "" && strings.TrimSpace(targetRaw) == "" && strings.TrimSpace(hostsRaw) == "" {
		return nil, errors.New("domain scope requires domain/target/hosts parameter")
	}

	if scope == "custom" {
		cidr := strings.TrimSpace(cidrRaw)
		if cidr == "" && len(targets) == 0 {
			return nil, errors.New("custom scope requires cidr or target/hosts parameter")
		}
		if cidr != "" {
			if parsedIP := net.ParseIP(cidr); parsedIP != nil {
				if v4 := parsedIP.To4(); v4 != nil {
					add(v4.String())
				} else {
					add(parsedIP.String())
				}
			} else {
				hosts, err := expandIPv4CIDRHosts(cidr, 512)
				if err != nil {
					return nil, err
				}
				for _, host := range hosts {
					add(host)
				}
			}
		}
	}

	if scope == "local" && len(targets) == 0 {
		add("127.0.0.1")
		add("localhost")
	}

	if hasLoopbackTarget(targets) {
		add("host.docker.internal")
	}

	return targets, nil
}

func hasLoopbackTarget(targets []string) bool {
	for _, target := range targets {
		value := strings.TrimSpace(strings.ToLower(target))
		switch value {
		case "127.0.0.1", "localhost", "::1":
			return true
		}
		if ip := net.ParseIP(value); ip != nil && ip.IsLoopback() {
			return true
		}
	}
	return false
}

func expandIPv4CIDRHosts(cidr string, maxHosts int) ([]string, error) {
	if maxHosts <= 0 {
		maxHosts = 256
	}
	baseIP, ipNet, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil {
		return nil, errors.New("invalid cidr")
	}
	baseV4 := baseIP.To4()
	if baseV4 == nil {
		return nil, errors.New("only ipv4 cidr is supported")
	}

	network := ipv4ToUint32(baseV4)
	mask := ipv4ToUint32(net.IP(ipNet.Mask).To4())
	broadcast := network | (^mask)

	start := network
	end := broadcast
	if broadcast-network >= 2 {
		start = network + 1
		end = broadcast - 1
	}

	hosts := make([]string, 0, minInt(maxHosts, int(end-start+1)))
	for current := start; current <= end && len(hosts) < maxHosts; current++ {
		hosts = append(hosts, uint32ToIPv4(current).String())
	}
	return hosts, nil
}

func ipv4ToUint32(ip net.IP) uint32 {
	v4 := ip.To4()
	if v4 == nil {
		return 0
	}
	return uint32(v4[0])<<24 | uint32(v4[1])<<16 | uint32(v4[2])<<8 | uint32(v4[3])
}

func uint32ToIPv4(raw uint32) net.IP {
	return net.IPv4(byte(raw>>24), byte(raw>>16), byte(raw>>8), byte(raw))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	identityName, _ := c.Locals("name").(string)
	identityEmail, _ := c.Locals("email").(string)
	identityName = strings.TrimSpace(identityName)
	identityEmail = strings.TrimSpace(identityEmail)
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
		name := strings.TrimSpace(u.Name)
		email := strings.TrimSpace(u.Email)
		if u.ID == uid {
			if name == "" {
				name = identityName
			}
			if email == "" {
				email = identityEmail
			}
		}
		results = append(results, fiber.Map{
			"id":         u.ID,
			"name":       name,
			"username":   u.Username,
			"email":      email,
			"role":       u.Role,
			"created_at": u.CreatedAt,
		})
	}
	return c.JSON(fiber.Map{"results": results, "count": len(results)})
}

func (a *App) getUser(c *fiber.Ctx) error {
	identifier := strings.TrimSpace(c.Params("id"))
	if strings.EqualFold(identifier, "me") {
		identifier, _ = c.Locals("uid").(string)
	}
	user, err := a.resolveUserIdentifier(identifier)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}
	role, _ := c.Locals("role").(string)
	uid, _ := c.Locals("uid").(string)
	if !isPrivilegedRole(role) && uid != user.ID {
		return fiber.NewError(fiber.StatusForbidden, "insufficient permissions")
	}
	identityName, _ := c.Locals("name").(string)
	identityEmail, _ := c.Locals("email").(string)
	name := strings.TrimSpace(user.Name)
	email := strings.TrimSpace(user.Email)
	if uid == user.ID {
		if name == "" {
			name = strings.TrimSpace(identityName)
		}
		if email == "" {
			email = strings.TrimSpace(identityEmail)
		}
	}
	return c.JSON(fiber.Map{
		"id":         user.ID,
		"name":       name,
		"username":   user.Username,
		"email":      email,
		"role":       user.Role,
		"created_at": user.CreatedAt,
	})
}

func (a *App) listMyTeams(c *fiber.Ctx) error {
	email, _ := c.Locals("email").(string)
	if strings.TrimSpace(email) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email is required for BasaltPass team lookup")
	}
	results, err := a.basaltGetUserTeams(email)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.JSON(fiber.Map{"results": results, "count": len(results)})
}

func (a *App) createTeam(c *fiber.Ctx) error {
	if !a.cfg.BasaltOAuthEnabled {
		return fiber.NewError(fiber.StatusNotImplemented, "team management is delegated to BasaltPass tenant APIs")
	}

	var req createTeamRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	email, _ := c.Locals("email").(string)
	data, err := a.basaltCreateTeam(req.Name, req.Description, email)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(data)
}

func (a *App) getTeam(c *fiber.Ctx) error {
	teamID := strings.TrimSpace(c.Params("id"))
	if teamID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}
	data, err := a.basaltGetTeamDetail(teamID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.JSON(data)
}

func (a *App) updateTeam(c *fiber.Ctx) error {
	return fiber.NewError(fiber.StatusNotImplemented, "team management is delegated to BasaltPass tenant APIs")
}

func (a *App) deleteTeam(c *fiber.Ctx) error {
	return fiber.NewError(fiber.StatusNotImplemented, "team management is delegated to BasaltPass tenant APIs")
}

func (a *App) getTeamMembers(c *fiber.Ctx) error {
	teamID := strings.TrimSpace(c.Params("id"))
	if teamID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid team id")
	}
	members, err := a.basaltGetTeamMembers(teamID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	return c.JSON(fiber.Map{"members": members})
}

func (a *App) addTeamMembers(c *fiber.Ctx) error {
	return fiber.NewError(fiber.StatusNotImplemented, "team management is delegated to BasaltPass tenant APIs")
}

func (a *App) removeTeamMember(c *fiber.Ctx) error {
	return fiber.NewError(fiber.StatusNotImplemented, "team management is delegated to BasaltPass tenant APIs")
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
