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

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

const (
	nodeAuthHeader          = "X-Node-Key"
	controlNodeAuthMetadata = "x-node-key"
)

type nodeVerifyResponse struct {
	Status string `json:"status"`
	Queue  string `json:"queue"`
}

func hashNodeKey(raw string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(raw)))
	return hex.EncodeToString(sum[:])
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

	return handler(ctx, req)
}
