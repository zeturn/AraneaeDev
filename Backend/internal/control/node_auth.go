package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	nodeAuthHeader          = "X-Node-Key"
	controlNodeAuthMetadata = "x-node-key"
	controlRunIDMetadata    = "x-run-id"
	controlRunTokenMetadata = "x-run-token"
	controlCorrelationIDMD  = "x-correlation-id"
)

type nodeContextKey string

const authenticatedNodeContextKey nodeContextKey = "authenticated-node"

type nodeVerifyResponse struct {
	Status string `json:"status"`
	Queue  string `json:"queue"`
}

type nodeCapabilitiesResponse struct {
	Status       string           `json:"status"`
	Queue        string           `json:"queue"`
	Capabilities []nodeCapability `json:"capabilities"`
}

type nodeAliveResponse struct {
	Status   string   `json:"status"`
	Queue    string   `json:"queue"`
	Hostname string   `json:"hostname"`
	Machine  string   `json:"machine"`
	OS       string   `json:"os"`
	HTTPAddr string   `json:"http_addr"`
	IPs      []string `json:"ips"`
}

func hashNodeKey(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
}

// bootstrapExecutorNode keeps Kubernetes deployments self-contained. The
// legacy deployment path required a human to call the admin registration API,
// which left a healthy executor fleet unable to authenticate after a fresh
// control database was created. The bootstrap is opt-in and stores only the
// hash of the shared executor key.
func (a *App) bootstrapExecutorNode() error {
	key := strings.TrimSpace(a.cfg.NodeBootstrapKey)
	if key == "" {
		return nil
	}
	ip := strings.TrimSpace(a.cfg.NodeBootstrapIP)
	if ip == "" {
		return errors.New("CONTROL_NODE_BOOTSTRAP_IP is required when CONTROL_NODE_BOOTSTRAP_KEY is set")
	}
	name := strings.TrimSpace(a.cfg.NodeBootstrapName)
	if name == "" {
		name = "executor-" + strings.NewReplacer(".", "-", ":", "-").Replace(ip)
	}
	port := a.cfg.NodeBootstrapPort
	if port <= 0 {
		port = 4280
	}
	grpcPort := a.cfg.NodeBootstrapGRPCPort
	if grpcPort <= 0 {
		grpcPort = 9190
	}
	queue := strings.TrimSpace(a.cfg.NodeBootstrapQueue)
	if queue == "" {
		queue = "default"
	}
	now := time.Now()
	keyHash := hashNodeKey(key)
	capabilities := toCapabilitiesJSON(defaultCapabilities())
	updates := map[string]interface{}{
		"description":       "Kubernetes executor pool",
		"status":            "active",
		"ip_address":        ip,
		"port":              port,
		"g_rpc_port":        grpcPort,
		"rpc_url":           fmt.Sprintf("http://%s:%d", ip, port),
		"celery_queue":      queue,
		"auth_token_hash":   keyHash,
		"is_enabled":        true,
		"last_active_time":  now,
		"updated_at":        now,
		"capabilities_json": capabilities,
	}

	var node common.Node
	err := a.db.Where("name = ?", name).First(&node).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		node = common.Node{
			Name:             name,
			Description:      "Kubernetes executor pool",
			Status:           "active",
			IPAddress:        ip,
			Port:             port,
			GRPCPort:         grpcPort,
			RPCURL:           fmt.Sprintf("http://%s:%d", ip, port),
			CeleryQueue:      queue,
			AuthTokenHash:    keyHash,
			IsEnabled:        true,
			LastActiveTime:   now,
			HDID:             strings.ToUpper(strings.ReplaceAll(uuid.NewString()[:12], "-", "")),
			CPUInfo:          defaultCPUInfoJSON(),
			MemoryInfo:       defaultMemoryInfoJSON(),
			CapabilitiesJSON: capabilities,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := a.db.Create(&node).Error; err != nil {
			return err
		}
		if a.log != nil {
			a.log.Info("bootstrapped executor node", zap.String("name", name), zap.String("queue", queue))
		}
		return nil
	}
	if err != nil {
		return err
	}
	if err := a.db.Model(&node).Updates(updates).Error; err != nil {
		return err
	}
	if a.log != nil {
		a.log.Info("reconciled executor node", zap.String("name", name), zap.String("queue", queue))
	}
	return nil
}

func (a *App) verifyExecutorNodeKey(ip string, port int, pairKey string) (*nodeVerifyResponse, error) {
	pairKey = strings.TrimSpace(pairKey)
	if pairKey == "" {
		return nil, errors.New("pair_key is required")
	}

	scheme := strings.ToLower(strings.TrimSpace(a.cfg.NodeVerifyScheme))
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return nil, errors.New("invalid CONTROL_NODE_VERIFY_SCHEME")
	}

	url := fmt.Sprintf("%s://%s:%d/node/verify", scheme, ip, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build executor verify request failed: %w", err)
	}
	req.Header.Set(nodeAuthHeader, pairKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executor verify request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("pair_key rejected by executor")
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("executor verify failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out nodeVerifyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode executor verify response failed: %w", err)
	}
	if strings.TrimSpace(out.Status) != "ok" {
		return nil, errors.New("executor verify response is invalid")
	}
	return &out, nil
}

func (a *App) fetchExecutorCapabilities(ip string, port int, pairKey string) (*nodeCapabilitiesResponse, error) {
	pairKey = strings.TrimSpace(pairKey)
	if pairKey == "" {
		return nil, errors.New("pair_key is required")
	}

	scheme := strings.ToLower(strings.TrimSpace(a.cfg.NodeVerifyScheme))
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return nil, errors.New("invalid CONTROL_NODE_VERIFY_SCHEME")
	}

	url := fmt.Sprintf("%s://%s:%d/node/capabilities", scheme, ip, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build executor capabilities request failed: %w", err)
	}
	req.Header.Set(nodeAuthHeader, pairKey)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executor capabilities request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("pair_key rejected by executor")
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("executor capabilities failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out nodeCapabilitiesResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16*1024)).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode executor capabilities response failed: %w", err)
	}
	if strings.TrimSpace(out.Status) != "ok" {
		return nil, errors.New("executor capabilities response is invalid")
	}
	return &out, nil
}

func (a *App) probeExecutorAlive(host string, port int) (*nodeAliveResponse, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return nil, errors.New("host is required")
	}

	scheme := strings.ToLower(strings.TrimSpace(a.cfg.NodeVerifyScheme))
	if scheme == "" {
		scheme = "http"
	}
	if scheme != "http" && scheme != "https" {
		return nil, errors.New("invalid CONTROL_NODE_VERIFY_SCHEME")
	}

	url := fmt.Sprintf("%s://%s:%d/node/alive", scheme, host, port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build executor alive request failed: %w", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executor alive request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusMultipleChoices {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("executor alive failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var out nodeAliveResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8192)).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode executor alive response failed: %w", err)
	}
	statusText := strings.ToLower(strings.TrimSpace(out.Status))
	if statusText != "ok" && statusText != "alive" {
		return nil, errors.New("executor alive response is invalid")
	}
	return &out, nil
}

func (a *App) nodeAuthUnaryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing node auth metadata")
	}

	nodeKeyValues := md.Get(controlNodeAuthMetadata)
	nodeKey := ""
	if len(nodeKeyValues) > 0 {
		nodeKey = strings.TrimSpace(nodeKeyValues[0])
	}
	if nodeKey == "" {
		return nil, status.Error(codes.Unauthenticated, "missing node key")
	}

	keyHash := hashNodeKey(nodeKey)
	var node common.Node
	if err := a.db.WithContext(ctx).Where("auth_token_hash = ? AND is_enabled = ?", keyHash, true).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.Unauthenticated, "invalid node key")
		}
		return nil, status.Error(codes.Internal, "node auth lookup failed")
	}

	now := time.Now()
	_ = a.db.Model(&common.Node{}).Where("id = ?", node.ID).Updates(map[string]interface{}{
		"status":           "active",
		"last_active_time": now,
		"updated_at":       now,
	}).Error

	return handler(context.WithValue(ctx, authenticatedNodeContextKey, node), req)
}

func authenticatedNodeFromContext(ctx context.Context) (common.Node, bool) {
	node, ok := ctx.Value(authenticatedNodeContextKey).(common.Node)
	return node, ok
}
