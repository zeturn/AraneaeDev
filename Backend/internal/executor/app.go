package executor

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	rabbitConn *amqp.Connection
	rabbitCh   *amqp.Channel
	grpcConn   *grpc.ClientConn
	grpcClient pb.ArtifactServiceClient
	httpClient *http.Client
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
	if callbackKey == "change-me-callback" || len(callbackKey) < 24 {
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
	a.http.Use(a.nodeAuthMiddleware)

	a.http.Get("/healthz", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"queue":  a.cfg.RabbitQueue,
		})
	})

	a.http.Get("/node/verify", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"queue":  a.cfg.RabbitQueue,
		})
	})
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
	if a.rabbitCh != nil {
		_ = a.rabbitCh.Close()
	}
	if a.rabbitConn != nil {
		_ = a.rabbitConn.Close()
	}
	return a.http.ShutdownWithContext(ctx)
}

func (a *App) startConsumer(ctx context.Context) error {
	qName := "executor." + a.cfg.RabbitQueue
	msgs, err := a.rabbitCh.Consume(qName, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	go func() {
		for {
			select {
			case msg, ok := <-msgs:
				if !ok {
					return
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
	}()
	return nil
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

	output, exitCode, execErr := a.executeTask(ctx, m)
	output = truncateOutput(output, maxExecutorOutputBytes)
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

	return a.reportCallback(m.RunID, m.RunToken, m.CorrelationID, contracts.CallbackPayload{
		Status:     status,
		Output:     output,
		ExitCode:   exitCode,
		StartedAt:  &startedAt,
		FinishedAt: &finishedAt,
	})
}

func (a *App) executeTask(ctx context.Context, msg contracts.QueueTaskMessage) (string, int, error) {
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(a.cfg.TaskTimeoutSeconds)*time.Second)
	defer cancel()

	resp, err := a.grpcClient.GetArtifact(
		a.withControlNodeAuth(runCtx, msg.RunID, msg.RunToken, msg.CorrelationID),
		&pb.GetArtifactRequest{ProjectId: msg.ProjectID, VersionId: msg.VersionID},
	)
	if err != nil {
		return "", 1, err
	}
	if len(resp.Content) == 0 {
		return "", 1, errors.New("empty artifact content")
	}
	sha := runtimeexec.ComputeSHA256(resp.Content)
	if sha != resp.Sha256 {
		return "", 1, fmt.Errorf("artifact checksum mismatch expected=%s actual=%s", resp.Sha256, sha)
	}
	runDir := filepath.Join(a.cfg.WorkDir, msg.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return "", 1, err
	}
	if err := runtimeexec.UnzipBytes(resp.Content, runDir); err != nil {
		return "", 1, err
	}
	return runtimeexec.RunCommand(runCtx, runDir, msg.EntryCommand)
}
