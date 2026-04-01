package executor

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc/metadata"
)

const (
	nodeAuthHeader          = "X-Node-Key"
	controlNodeAuthMetadata = "x-node-key"
)

func ensureNodeAuthKey(cfg *common.ExecutorConfig) error {
	cfg.NodeAuthKey = strings.TrimSpace(cfg.NodeAuthKey)
	if cfg.NodeAuthKey != "" {
		return nil
	}

	cfg.NodeAuthKeyFile = strings.TrimSpace(cfg.NodeAuthKeyFile)
	if cfg.NodeAuthKeyFile == "" {
		return errors.New("executor node auth key file path is empty")
	}

	if b, err := os.ReadFile(cfg.NodeAuthKeyFile); err == nil {
		if key := strings.TrimSpace(string(b)); key != "" {
			cfg.NodeAuthKey = key
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	key, err := generateNodeAuthKey()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.NodeAuthKeyFile), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(cfg.NodeAuthKeyFile, []byte(key+"\n"), 0o600); err != nil {
		return err
	}
	cfg.NodeAuthKey = key
	return nil
}

func generateNodeAuthKey() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (a *App) nodeAuthMiddleware(c *fiber.Ctx) error {
	provided := strings.TrimSpace(c.Get(nodeAuthHeader))
	if provided == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing node key")
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(a.cfg.NodeAuthKey)) != 1 {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid node key")
	}
	return c.Next()
}

func (a *App) withControlNodeAuth(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, controlNodeAuthMetadata, a.cfg.NodeAuthKey)
}
