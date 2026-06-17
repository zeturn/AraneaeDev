package control

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"araneae-go/gen/pb"
	"araneae-go/internal/common"
	"araneae-go/internal/control/contracts"
	"araneae-go/internal/control/infra/netx"
	"araneae-go/internal/control/security/password"

	"github.com/glebarez/sqlite"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gorm.io/gorm"
)

type App struct {
	pb.UnimplementedArtifactServiceServer

	cfg             common.ControlConfig
	log             *zap.Logger
	db              *gorm.DB
	http            *fiber.App
	cron            *cron.Cron
	cronEntries     map[string]cron.EntryID
	scheduleEntries map[string]cron.EntryID
	scheduleTimers  map[string]*time.Timer
	oauthCodes      map[string]oauthExchangeState
	cronMu          sync.Mutex
	oauthMu         sync.Mutex
	rabbitMu        sync.Mutex
	rabbitConn      *amqp.Connection
	rabbitCh        *amqp.Channel
	grpcSrv         *grpc.Server
}

type chainRunMeta struct {
	ChainID    string
	ChainIndex int
	ChainTotal int
}

var errQueueUnavailable = errors.New("task queue publisher unavailable")

const maxArtifactUploadBytes = 50 * 1024 * 1024

func randomSecret(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func isWeakSecret(value string, minLen int, denyList ...string) bool {
	v := strings.TrimSpace(value)
	if len(v) < minLen {
		return true
	}
	for _, denied := range denyList {
		if v == denied {
			return true
		}
	}
	return false
}

func isWildcardCORS(raw string) bool {
	for _, origin := range strings.Split(raw, ",") {
		if strings.TrimSpace(origin) == "*" {
			return true
		}
	}
	return false
}

func usesDefaultRabbitCredentials(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User == nil {
		return false
	}
	username := strings.TrimSpace(parsed.User.Username())
	password, hasPassword := parsed.User.Password()
	if !hasPassword {
		return false
	}
	return username == "guest" && strings.TrimSpace(password) == "guest"
}

func validateSecurityConfig(cfg *common.ControlConfig, log *zap.Logger) error {
	cfg.JWTSecret = strings.TrimSpace(cfg.JWTSecret)
	cfg.ExecutionAPIKey = strings.TrimSpace(cfg.ExecutionAPIKey)
	cfg.InitAdminPassword = strings.TrimSpace(cfg.InitAdminPassword)
	cfg.NodeVerifyScheme = strings.ToLower(strings.TrimSpace(cfg.NodeVerifyScheme))
	if cfg.NodeVerifyScheme == "" {
		cfg.NodeVerifyScheme = "http"
	}

	isProd := strings.EqualFold(strings.TrimSpace(cfg.Environment), "production")
	if isWeakSecret(cfg.JWTSecret, 32, "change-me") {
		if isProd {
			return errors.New("CONTROL_JWT_SECRET is missing or too weak for production")
		}
		secret, err := randomSecret(32)
		if err != nil {
			return fmt.Errorf("generate development CONTROL_JWT_SECRET failed: %w", err)
		}
		cfg.JWTSecret = secret
		log.Warn("CONTROL_JWT_SECRET is missing/weak; generated an ephemeral development secret")
	}

	if isWeakSecret(cfg.ExecutionAPIKey, 32, "change-me-callback") {
		if isProd {
			return errors.New("EXECUTION_CALLBACK_KEY is missing or too weak for production")
		}
		secret, err := randomSecret(32)
		if err != nil {
			return fmt.Errorf("generate development EXECUTION_CALLBACK_KEY failed: %w", err)
		}
		cfg.ExecutionAPIKey = secret
		log.Warn("EXECUTION_CALLBACK_KEY is missing/weak; generated an ephemeral development key")
	}

	if isWeakSecret(cfg.InitAdminPassword, 12, "admin123") {
		if isProd {
			return errors.New("INIT_ADMIN_PASSWORD is missing or too weak for production")
		}
		password, err := randomSecret(18)
		if err != nil {
			return fmt.Errorf("generate development INIT_ADMIN_PASSWORD failed: %w", err)
		}
		cfg.InitAdminPassword = password
		log.Warn("INIT_ADMIN_PASSWORD is missing/weak; generated an ephemeral development admin password", zap.String("generated_admin_password", password))
	}

	if !isProd {
		return nil
	}

	if isWildcardCORS(cfg.CORSAllowOrigins) {
		return errors.New("CONTROL_CORS_ALLOW_ORIGINS must not contain wildcard '*' in production")
	}

	if usesDefaultRabbitCredentials(cfg.RabbitURL) {
		return errors.New("RABBITMQ_URL must not use default guest credentials in production")
	}

	if cfg.NodeVerifyScheme != "https" {
		return errors.New("CONTROL_NODE_VERIFY_SCHEME must be https for production")
	}

	if !cfg.GRPCTLSEnabled {
		return errors.New("CONTROL_GRPC_TLS_ENABLED must be true for production")
	}
	if strings.TrimSpace(cfg.GRPCTLSCertFile) == "" || strings.TrimSpace(cfg.GRPCTLSKeyFile) == "" {
		return errors.New("CONTROL_GRPC_TLS_CERT_FILE and CONTROL_GRPC_TLS_KEY_FILE are required for production")
	}

	return nil
}

func (a *App) buildControlGRPCServerOptions() ([]grpc.ServerOption, error) {
	cfg := a.cfg
	opts := []grpc.ServerOption{grpc.UnaryInterceptor(a.nodeAuthUnaryInterceptor)}
	if !cfg.GRPCTLSEnabled {
		return opts, nil
	}

	certFile := strings.TrimSpace(cfg.GRPCTLSCertFile)
	keyFile := strings.TrimSpace(cfg.GRPCTLSKeyFile)
	if certFile == "" || keyFile == "" {
		return nil, errors.New("CONTROL_GRPC_TLS_CERT_FILE and CONTROL_GRPC_TLS_KEY_FILE are required when CONTROL_GRPC_TLS_ENABLED=true")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("load control grpc tls cert/key failed: %w", err)
	}
	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cert},
	}

	clientCAFile := strings.TrimSpace(cfg.GRPCTLSClientCAFile)
	if clientCAFile != "" {
		caPEM, err := os.ReadFile(clientCAFile)
		if err != nil {
			return nil, fmt.Errorf("read control grpc client ca failed: %w", err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, errors.New("parse control grpc client ca failed")
		}
		tlsCfg.ClientCAs = pool
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
	}

	opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	return opts, nil
}

func NewApp(cfg common.ControlConfig) (*App, error) {
	log, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	if err := validateSecurityConfig(&cfg, log); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.ArtifactRoot, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(cfg.RSSRoot, 0o755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(common.AutoMigrateModels()...); err != nil {
		return nil, err
	}
	if err := reconcileSchema(db); err != nil {
		return nil, err
	}

	app := &App{
		cfg:             cfg,
		log:             log,
		db:              db,
		http:            fiber.New(),
		cron:            cron.New(cron.WithSeconds()),
		cronEntries:     make(map[string]cron.EntryID),
		scheduleEntries: make(map[string]cron.EntryID),
		scheduleTimers:  make(map[string]*time.Timer),
		oauthCodes:      make(map[string]oauthExchangeState),
	}

	app.http.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSAllowOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-CSRFToken",
	}))

	if err := app.seedAdmin(); err != nil {
		return nil, err
	}
	if err := app.initRabbit(); err != nil {
		return nil, err
	}
	app.setupRoutes()
	if err := app.loadCronTasks(); err != nil {
		return nil, err
	}
	if err := app.loadCronSchedules(); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	a.cron.Start()

	grpcLis, err := netx.Listen("tcp", a.cfg.GRPCAddr)
	if err != nil {
		return err
	}
	grpcOpts, err := a.buildControlGRPCServerOptions()
	if err != nil {
		return err
	}
	a.grpcSrv = grpc.NewServer(grpcOpts...)
	pb.RegisterArtifactServiceServer(a.grpcSrv, a)
	go func() {
		a.log.Info("control grpc started", zap.String("addr", a.cfg.GRPCAddr))
		if err := a.grpcSrv.Serve(grpcLis); err != nil {
			a.log.Error("grpc serve failed", zap.Error(err))
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		a.log.Info("control http started", zap.String("addr", a.cfg.HTTPAddr))
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
	a.cron.Stop()
	if a.grpcSrv != nil {
		a.grpcSrv.GracefulStop()
	}
	if a.rabbitCh != nil {
		_ = a.rabbitCh.Close()
	}
	if a.rabbitConn != nil {
		_ = a.rabbitConn.Close()
	}
	return a.http.ShutdownWithContext(ctx)
}

func (a *App) seedAdmin() error {
	var count int64
	if err := a.db.Model(&common.User{}).Where("username = ?", "admin").Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	adminPassword := strings.TrimSpace(a.cfg.InitAdminPassword)
	if adminPassword == "" {
		return errors.New("INIT_ADMIN_PASSWORD is required when seeding admin user")
	}
	hash, err := password.Hash(adminPassword)
	if err != nil {
		return err
	}
	return a.db.Create(&common.User{
		ID:           uuid.NewString(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         "admin",
		CreatedAt:    time.Now(),
	}).Error
}

func (a *App) initRabbit() error {
	conn, ch, err := a.openRabbit()
	if err != nil {
		return err
	}
	a.rabbitMu.Lock()
	defer a.rabbitMu.Unlock()
	a.rabbitConn = conn
	a.rabbitCh = ch
	return nil
}

func (a *App) openRabbit() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(a.cfg.RabbitURL)
	if err != nil {
		return nil, nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, err
	}
	if err := ch.ExchangeDeclare(a.cfg.RabbitExchange, "direct", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, err
	}
	return conn, ch, nil
}

func (a *App) rabbitPublisher() (*amqp.Channel, error) {
	a.rabbitMu.Lock()
	if a.rabbitCh == nil && a.rabbitConn == nil {
		a.rabbitMu.Unlock()
		return nil, errQueueUnavailable
	}
	if a.rabbitCh != nil && !a.rabbitCh.IsClosed() && a.rabbitConn != nil && !a.rabbitConn.IsClosed() {
		ch := a.rabbitCh
		a.rabbitMu.Unlock()
		return ch, nil
	}
	if a.rabbitCh != nil {
		_ = a.rabbitCh.Close()
	}
	if a.rabbitConn != nil {
		_ = a.rabbitConn.Close()
	}
	a.rabbitCh = nil
	a.rabbitConn = nil
	a.rabbitMu.Unlock()

	return a.reconnectRabbitPublisher()
}

func (a *App) reconnectRabbitPublisher() (*amqp.Channel, error) {
	conn, ch, err := a.openRabbit()
	if err != nil {
		return nil, err
	}
	a.rabbitMu.Lock()
	defer a.rabbitMu.Unlock()
	a.rabbitConn = conn
	a.rabbitCh = ch
	return ch, nil
}

func (a *App) publishTaskRun(task common.Task, source string, scheduleID string) (*common.TaskRun, error) {
	return a.publishRun(task.ID, scheduleID, source, task.ProjectID, task.VersionID, task.EntryCommand, task.NodeQueue, nil)
}

func (a *App) publishScheduleRun(schedule common.Schedule, source string) (*common.TaskRun, error) {
	steps, err := a.resolveScheduleExecutionSteps(schedule)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		steps = []scheduleExecutionStep{{
			TaskID:       schedule.ID,
			ProjectID:    schedule.ProjectID,
			VersionID:    schedule.VersionID,
			EntryCommand: schedule.EntryCommand,
			NodeQueue:    schedule.NodeQueue,
		}}
	}

	var chainMeta *chainRunMeta
	if len(steps) > 1 {
		chainMeta = &chainRunMeta{
			ChainID:    uuid.NewString(),
			ChainIndex: 0,
			ChainTotal: len(steps),
		}
	}

	first := steps[0]
	run, err := a.publishRun(first.TaskID, schedule.ID, source, first.ProjectID, first.VersionID, first.EntryCommand, first.NodeQueue, chainMeta)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := a.db.Model(&common.Schedule{}).Where("id = ?", schedule.ID).Update("last_triggered_at", &now).Error; err != nil {
		a.log.Warn("failed to update schedule last_triggered_at", zap.Error(err), zap.String("schedule_id", schedule.ID))
	}
	return run, nil
}

func (a *App) publishRun(taskID, scheduleID, source, projectID, versionID, entryCommand, nodeQueue string, chainMeta *chainRunMeta) (*common.TaskRun, error) {
	if taskID == "" {
		taskID = scheduleID
	}
	if nodeQueue == "" {
		nodeQueue = "default"
	}
	run := &common.TaskRun{
		ID:            uuid.NewString(),
		TaskID:        taskID,
		ScheduleID:    scheduleID,
		TriggerSource: source,
		NodeQueue:     nodeQueue,
		Status:        "queued",
		CreatedAt:     time.Now(),
		CorrelationID: uuid.NewString(),
	}
	runToken, err := randomSecret(32)
	if err != nil {
		return nil, err
	}
	run.RunTokenHash = hashNodeKey(runToken)
	if chainMeta != nil {
		run.ChainID = chainMeta.ChainID
		run.ChainIndex = chainMeta.ChainIndex
		run.ChainTotal = chainMeta.ChainTotal
	}
	if err := a.db.Create(run).Error; err != nil {
		return nil, err
	}

	ch, err := a.rabbitPublisher()
	if err != nil {
		_ = a.db.Model(&common.TaskRun{}).Where("id = ?", run.ID).Updates(map[string]interface{}{
			"status":      "failed",
			"output":      err.Error(),
			"finished_at": time.Now(),
		}).Error
		return nil, err
	}

	payload := contracts.QueueTaskMessage{
		RunID:         run.ID,
		TaskID:        taskID,
		ProjectID:     projectID,
		VersionID:     versionID,
		EntryCommand:  entryCommand,
		NodeQueue:     nodeQueue,
		CorrelationID: run.CorrelationID,
		RunToken:      runToken,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	routingKey := "tasks." + nodeQueue
	publishing := amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
		MessageId:    run.CorrelationID,
		Timestamp:    time.Now(),
	}
	err = ch.PublishWithContext(context.Background(), a.cfg.RabbitExchange, routingKey, false, false, publishing)
	if err != nil {
		if retryCh, retryErr := a.reconnectRabbitPublisher(); retryErr == nil {
			err = retryCh.PublishWithContext(context.Background(), a.cfg.RabbitExchange, routingKey, false, false, publishing)
		}
	}
	if err != nil {
		_ = a.db.Model(&common.TaskRun{}).Where("id = ?", run.ID).Updates(map[string]interface{}{
			"status":      "failed",
			"output":      err.Error(),
			"finished_at": time.Now(),
		}).Error
		return nil, err
	}
	return run, nil
}

func computeSHA256(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func writeArtifactFile(baseDir, projectID, versionID, fileName string, data []byte) (string, string, error) {
	safeName := filepath.Base(fileName)
	if safeName == "." || safeName == string(filepath.Separator) || safeName == "" {
		safeName = "artifact.zip"
	}
	projectDir := filepath.Join(baseDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return "", "", err
	}
	path := filepath.Join(projectDir, fmt.Sprintf("%s_%s", versionID, safeName))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", "", err
	}
	return path, computeSHA256(data), nil
}

func loadUploadedFile(c *fiber.Ctx, fieldName string) ([]byte, string, error) {
	h, err := c.FormFile(fieldName)
	if err != nil {
		return nil, "", err
	}
	if h.Size > maxArtifactUploadBytes {
		return nil, "", fmt.Errorf("uploaded file is too large (max %d bytes)", maxArtifactUploadBytes)
	}
	f, err := h.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxArtifactUploadBytes+1))
	if err != nil {
		return nil, "", err
	}
	if int64(len(b)) > maxArtifactUploadBytes {
		return nil, "", fmt.Errorf("uploaded file is too large (max %d bytes)", maxArtifactUploadBytes)
	}
	if len(b) == 0 {
		return nil, "", errors.New("uploaded file is empty")
	}
	return b, h.Filename, nil
}
