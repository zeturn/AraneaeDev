package executor

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"araneae-go/gen/pb"
	"araneae-go/internal/common"
	"araneae-go/internal/executor/contracts"
	"araneae-go/internal/executor/runtimeexec"
	"araneae-go/internal/executor/store"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

type App struct {
	cfg        common.ExecutorConfig
	log        *zap.Logger
	db         *gorm.DB
	http       *fiber.App
	rabbitMu   sync.Mutex
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	grpcConn   *grpc.ClientConn
	grpcClient pb.ArtifactServiceClient
	httpClient *http.Client
	tokenMu    sync.Mutex
	tokenValue string
	tokenUntil time.Time
}

type runtimeCapability struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
}

const maxExecutorOutputBytes = 900 * 1024

func truncateOutput(raw string, maxBytes int) string {
	if maxBytes <= 0 || len(raw) <= maxBytes {
		return raw
	}
	return raw[:maxBytes]
}

func validateExecutorSecurityConfig(cfg common.ExecutorConfig) error {
	isProd := strings.EqualFold(strings.TrimSpace(cfg.Environment), "production")
	if strings.TrimSpace(cfg.ControlCallbackKey) == "" {
		return errors.New("EXECUTION_CALLBACK_KEY is required")
	}
	if cfg.TaskTimeoutSeconds <= 0 {
		return errors.New("EXECUTOR_TASK_TIMEOUT_SECONDS must be greater than 0")
	}
	if !isProd {
		return nil
	}
	if !cfg.ControlGRPCTLSEnabled {
		return errors.New("EXECUTOR_CONTROL_GRPC_TLS_ENABLED must be true for production")
	}
	if strings.TrimSpace(cfg.ControlGRPCTLSServerName) == "" {
		return errors.New("EXECUTOR_CONTROL_GRPC_TLS_SERVER_NAME is required for production")
	}
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(cfg.ControlHTTPBase)), "https://") {
		return errors.New("CONTROL_HTTP_BASE must use https:// for production")
	}
	callbackKey := strings.TrimSpace(cfg.ControlCallbackKey)
	if callbackKey == "change-me-callback" || len(callbackKey) < 32 {
		return errors.New("EXECUTION_CALLBACK_KEY is missing or too weak for production")
	}
	return nil
}

func buildControlGRPCTransportCredentials(cfg common.ExecutorConfig) (credentials.TransportCredentials, error) {
	if !cfg.ControlGRPCTLSEnabled {
		return insecure.NewCredentials(), nil
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if serverName := strings.TrimSpace(cfg.ControlGRPCTLSServerName); serverName != "" {
		tlsCfg.ServerName = serverName
	}

	caFile := strings.TrimSpace(cfg.ControlGRPCTLSCAFile)
	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("read executor grpc ca failed: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("parse executor grpc ca failed")
		}
		tlsCfg.RootCAs = pool
	}

	certFile := strings.TrimSpace(cfg.ExecutorGRPCTLSCertFile)
	keyFile := strings.TrimSpace(cfg.ExecutorGRPCTLSKeyFile)
	if (certFile == "") != (keyFile == "") {
		return nil, errors.New("EXECUTOR_GRPC_TLS_CERT_FILE and EXECUTOR_GRPC_TLS_KEY_FILE must be provided together")
	}
	if certFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("load executor grpc client cert/key failed: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsCfg), nil
}

func NewApp(cfg common.ExecutorConfig) (*App, error) {
	log, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	if err := validateExecutorSecurityConfig(cfg); err != nil {
		return nil, err
	}
	if cfg.TaskTimeoutSeconds <= 0 {
		cfg.TaskTimeoutSeconds = 1800
	}
	if err := ensureNodeAuthKey(&cfg); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.WorkDir, 0o755); err != nil {
		return nil, err
	}
	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&store.ExecutionRecord{}); err != nil {
		return nil, err
	}

	rabbitConn, rabbitCh, err := initRabbit(cfg)
	if err != nil {
		return nil, err
	}

	transportCreds, err := buildControlGRPCTransportCredentials(cfg)
	if err != nil {
		_ = rabbitCh.Close()
		_ = rabbitConn.Close()
		return nil, err
	}

	grpcConn, err := grpc.Dial(cfg.ControlGRPCAddr, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		_ = rabbitCh.Close()
		_ = rabbitConn.Close()
		return nil, err
	}

	a := &App{
		cfg:        cfg,
		log:        log,
		db:         db,
		http:       fiber.New(),
		rabbitConn: rabbitConn,
		rabbitCh:   rabbitCh,
		grpcConn:   grpcConn,
		grpcClient: pb.NewArtifactServiceClient(grpcConn),
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
	a.log.Info("executor node auth key ready",
		zap.String("node_key_file", cfg.NodeAuthKeyFile),
	)
	a.setupRoutes()
	return a, nil
}

func initRabbit(cfg common.ExecutorConfig) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(cfg.RabbitURL)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := ch.ExchangeDeclare(cfg.RabbitExchange, "direct", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	qName := "executor." + cfg.RabbitQueue
	q, err := ch.QueueDeclare(qName, true, false, false, false, nil)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	if err := ch.QueueBind(q.Name, "tasks."+cfg.RabbitQueue, cfg.RabbitExchange, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}

func (a *App) setupRoutes() {
	a.http.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"queue":  a.cfg.RabbitQueue,
		})
	})

	// Public probe endpoint: allows control plane discovery even before pair-key matching.
	a.http.Get("/node/alive", func(c *fiber.Ctx) error {
		hostname, _ := os.Hostname()
		ips := collectLocalIPv4s()
		return c.JSON(fiber.Map{
			"status":    "alive",
			"queue":     a.cfg.RabbitQueue,
			"hostname":  strings.TrimSpace(hostname),
			"machine":   runtime.GOARCH,
			"os":        runtime.GOOS,
			"http_addr": strings.TrimSpace(a.cfg.HTTPAddr),
			"ips":       ips,
		})
	})

	a.http.Use(a.nodeAuthMiddleware)

	a.http.Get("/node/verify", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"queue":  a.cfg.RabbitQueue,
		})
	})

	a.http.Get("/node/capabilities", func(c *fiber.Ctx) error {
		caps := collectRuntimeCapabilities()
		return c.JSON(fiber.Map{
			"status":       "ok",
			"queue":        a.cfg.RabbitQueue,
			"capabilities": caps,
		})
	})
}

func collectLocalIPv4s() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return []string{}
	}
	seen := make(map[string]struct{})
	ips := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP == nil {
			continue
		}
		ip := ipNet.IP.To4()
		if ip == nil {
			continue
		}
		if ip[0] == 127 {
			continue
		}
		text := ip.String()
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		ips = append(ips, text)
	}
	sort.Strings(ips)
	return ips
}

func collectRuntimeCapabilities() []runtimeCapability {
	return []runtimeCapability{
		detectRuntimeCapability("python", "Python", []runtimeCheck{{Command: "python3", Args: []string{"--version"}}, {Command: "python", Args: []string{"--version"}}}),
		detectRuntimeCapability("node", "Node.js", []runtimeCheck{{Command: "node", Args: []string{"--version"}}}),
		detectRuntimeCapability("go", "Go", []runtimeCheck{{Command: "go", Args: []string{"version"}}}),
		detectRuntimeCapability("java", "Java", []runtimeCheck{{Command: "java", Args: []string{"-version"}}}),
	}
}

type runtimeCheck struct {
	Command string
	Args    []string
}

func detectRuntimeCapability(key, name string, checks []runtimeCheck) runtimeCapability {
	for _, check := range checks {
		if _, err := exec.LookPath(check.Command); err != nil {
			continue
		}
		cmd := exec.Command(check.Command, check.Args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			continue
		}
		version := firstNonEmptyLine(string(output))
		if version == "" {
			version = "installed"
		}
		return runtimeCapability{Key: key, Name: name, Available: true, Version: version}
	}
	return runtimeCapability{Key: key, Name: name, Available: false}
}

func firstNonEmptyLine(raw string) string {
	for _, line := range bytes.Split([]byte(raw), []byte("\n")) {
		trimmed := strings.TrimSpace(string(line))
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (a *App) Run(ctx context.Context) error {
	if err := a.startConsumer(ctx); err != nil {
		return err
	}
	errCh := make(chan error, 1)
	go func() {
		a.log.Info("executor http started", zap.String("addr", a.cfg.HTTPAddr), zap.String("queue", a.cfg.RabbitQueue))
		errCh <- a.http.Listen(a.cfg.HTTPAddr)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return a.Shutdown(context.Background())
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	if a.grpcConn != nil {
		_ = a.grpcConn.Close()
	}
	a.closeRabbit()
	return a.http.ShutdownWithContext(ctx)
}

func (a *App) startConsumer(ctx context.Context) error {
	msgs, err := a.openConsumer()
	if err != nil {
		return err
	}
	go a.consumeLoop(ctx, msgs)
	return nil
}

func (a *App) openConsumer() (<-chan amqp.Delivery, error) {
	a.rabbitMu.Lock()
	defer a.rabbitMu.Unlock()

	if a.rabbitConn == nil || a.rabbitConn.IsClosed() || a.rabbitCh == nil {
		conn, ch, err := initRabbit(a.cfg)
		if err != nil {
			return nil, err
		}
		a.rabbitConn = conn
		a.rabbitCh = ch
	}

	if err := a.rabbitCh.Qos(1, 0, false); err != nil {
		a.closeRabbitLocked()
		return nil, err
	}

	qName := "executor." + a.cfg.RabbitQueue
	msgs, err := a.rabbitCh.Consume(qName, "", false, false, false, false, nil)
	if err != nil {
		a.closeRabbitLocked()
		return nil, err
	}
	return msgs, nil
}

func (a *App) consumeLoop(ctx context.Context, msgs <-chan amqp.Delivery) {
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				a.log.Warn("rabbit consumer channel closed; reconnecting", zap.String("queue", a.cfg.RabbitQueue))
				reconnected, err := a.reconnectConsumer(ctx)
				if err != nil {
					return
				}
				msgs = reconnected
				continue
			}
			if err := a.processMessage(ctx, msg.Body); err != nil {
				a.log.Error("process message failed", zap.Error(err))
				_ = msg.Nack(false, false)
				continue
			}
			_ = msg.Ack(false)
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) reconnectConsumer(ctx context.Context) (<-chan amqp.Delivery, error) {
	backoff := time.Second
	for {
		a.closeRabbit()
		msgs, err := a.openConsumer()
		if err == nil {
			a.log.Info("rabbit consumer reconnected", zap.String("queue", a.cfg.RabbitQueue))
			return msgs, nil
		}
		a.log.Warn("rabbit consumer reconnect failed", zap.Error(err), zap.Duration("retry_in", backoff))

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		if backoff < 30*time.Second {
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (a *App) closeRabbit() {
	a.rabbitMu.Lock()
	defer a.rabbitMu.Unlock()
	a.closeRabbitLocked()
}

func (a *App) closeRabbitLocked() {
	if a.rabbitCh != nil {
		_ = a.rabbitCh.Close()
		a.rabbitCh = nil
	}
	if a.rabbitConn != nil {
		_ = a.rabbitConn.Close()
		a.rabbitConn = nil
	}
}

func (a *App) processMessage(ctx context.Context, raw []byte) error {
	var m contracts.QueueTaskMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}
	startedAt := time.Now()
	rec := store.ExecutionRecord{
		RunID:     m.RunID,
		TaskID:    m.TaskID,
		Status:    "running",
		CreatedAt: startedAt,
	}
	if err := a.db.Save(&rec).Error; err != nil {
		return err
	}

	output, exitCode, runDir, execErr := a.executeTask(ctx, m)
	output = truncateOutput(output, maxExecutorOutputBytes)
	sinkSummary, sinkErr := a.processSinkArtifacts(ctx, m, runDir)
	if sinkSummary != "" {
		output = truncateOutput(output+"\n"+sinkSummary, maxExecutorOutputBytes)
	}
	if sinkErr != nil {
		output = truncateOutput(output+"\n"+"sink error: "+sinkErr.Error(), maxExecutorOutputBytes)
		if execErr == nil && a.cfg.SinkStrict {
			execErr = fmt.Errorf("sink transfer failed: %w", sinkErr)
			exitCode = 1
		}
	}
	finishedAt := time.Now()
	status := "success"
	if execErr != nil {
		status = "failed"
		if errors.Is(execErr, context.DeadlineExceeded) {
			execErr = fmt.Errorf("task execution timed out after %ds", a.cfg.TaskTimeoutSeconds)
		}
		output = fmt.Sprintf("%s\nerror: %v", output, execErr)
		output = truncateOutput(output, maxExecutorOutputBytes)
	}

	if err := a.db.Model(&store.ExecutionRecord{}).Where("run_id = ?", m.RunID).Updates(map[string]interface{}{
		"status":      status,
		"output":      output,
		"exit_code":   exitCode,
		"finished_at": finishedAt,
	}).Error; err != nil {
		return err
	}

	if err := a.reportCallback(m.RunID, m.RunToken, m.CorrelationID, contracts.CallbackPayload{
		Status:     status,
		Output:     output,
		ExitCode:   exitCode,
		StartedAt:  &startedAt,
		FinishedAt: &finishedAt,
	}); err != nil {
		a.log.Warn("report callback failed after task completion",
			zap.String("run_id", m.RunID),
			zap.String("task_id", m.TaskID),
			zap.String("status", status),
			zap.Error(err),
		)
		// The task already ran and any sink output has already been processed.
		// Do not Nack the RabbitMQ message, otherwise non-idempotent crawlers or
		// partially successful sink transfers may execute repeatedly.
		_ = a.db.Model(&store.ExecutionRecord{}).Where("run_id = ?", m.RunID).Updates(map[string]interface{}{
			"status": "callback_failed",
			"output": truncateOutput(output+"\ncallback error: "+err.Error(), maxExecutorOutputBytes),
		}).Error
	}
	return nil
}

func (a *App) executeTask(ctx context.Context, msg contracts.QueueTaskMessage) (string, int, string, error) {
	taskType := strings.ToLower(strings.TrimSpace(msg.Type))
	if taskType == "rss" || taskType == "api" {
		return a.executeSourceFetch(ctx, taskType, msg.SourceURL, msg)
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(a.cfg.TaskTimeoutSeconds)*time.Second)
	defer cancel()

	resp, err := a.grpcClient.GetArtifact(
		a.withControlNodeAuth(runCtx, msg.RunID, msg.RunToken, msg.CorrelationID),
		&pb.GetArtifactRequest{ProjectId: msg.ProjectID, VersionId: msg.VersionID},
	)
	if err != nil {
		return "", 1, "", err
	}
	if len(resp.Content) == 0 {
		return "", 1, "", errors.New("empty artifact content")
	}
	sha := runtimeexec.ComputeSHA256(resp.Content)
	if sha != resp.Sha256 {
		return "", 1, "", fmt.Errorf("artifact checksum mismatch expected=%s actual=%s", resp.Sha256, sha)
	}
	runDir := filepath.Join(a.cfg.WorkDir, msg.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", 1, runDir, err
	}
	if err := runtimeexec.UnzipBytes(resp.Content, runDir); err != nil {
		return "", 1, runDir, err
	}
	sinkDir := filepath.Join(runDir, filepath.FromSlash(strings.TrimSpace(a.cfg.SinkDirName)))
	if sinkDir == runDir || strings.TrimSpace(a.cfg.SinkDirName) == "" {
		sinkDir = filepath.Join(runDir, ".araneae", "sink")
	}
	if err := os.MkdirAll(sinkDir, 0o755); err != nil {
		return "", 1, runDir, err
	}
	if err := ensureSinkSDK(runDir); err != nil {
		a.log.Warn("write sink sdk failed", zap.Error(err), zap.String("run_id", msg.RunID))
	}
	env := map[string]string{
		"ARANEAE_RUNTIME":   "1",
		"ARANEAE_SINK_MODE": "araneae",
		"ARANEAE_SINK_DIR":  sinkDir,
	}
	out, code, runErr := runtimeexec.RunCommand(runCtx, runDir, msg.EntryCommand, env)
	return out, code, runDir, runErr
}
