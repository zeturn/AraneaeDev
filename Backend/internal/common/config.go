package common

import (
	"os"
	"strconv"
	"strings"
)

type ControlConfig struct {
	Environment       string
	HTTPAddr          string
	GRPCAddr          string
	GRPCTLSEnabled    bool
	GRPCTLSCertFile   string
	GRPCTLSKeyFile    string
	GRPCTLSClientCAFile string
	DBPath            string
	RabbitURL         string
	RabbitExchange    string
	InitAdminPassword string
	JWTSecret         string
	ArtifactRoot      string
	ExecutionAPIKey   string
	CORSAllowOrigins  string
}

type ExecutorConfig struct {
	Environment        string
	HTTPAddr           string
	DBPath             string
	RabbitURL          string
	RabbitExchange     string
	RabbitQueue        string
	NodeAuthKey        string
	NodeAuthKeyFile    string
	ControlGRPCAddr    string
	ControlGRPCTLSEnabled bool
	ControlGRPCTLSCAFile string
	ControlGRPCTLSServerName string
	ExecutorGRPCTLSCertFile string
	ExecutorGRPCTLSKeyFile string
	ControlHTTPBase    string
	ControlCallbackKey string
	WorkDir            string
}

func GetEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func GetEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func GetEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return fallback
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func LoadControlConfig() ControlConfig {
	return ControlConfig{
		Environment:       GetEnv("ARANEAE_ENV", "development"),
		HTTPAddr:          GetEnv("CONTROL_HTTP_ADDR", ":8180"),
		GRPCAddr:          GetEnv("CONTROL_GRPC_ADDR", ":9190"),
		GRPCTLSEnabled:    GetEnvBool("CONTROL_GRPC_TLS_ENABLED", false),
		GRPCTLSCertFile:   GetEnv("CONTROL_GRPC_TLS_CERT_FILE", ""),
		GRPCTLSKeyFile:    GetEnv("CONTROL_GRPC_TLS_KEY_FILE", ""),
		GRPCTLSClientCAFile: GetEnv("CONTROL_GRPC_TLS_CLIENT_CA_FILE", ""),
		DBPath:            GetEnv("CONTROL_DB_PATH", "./data/control.db"),
		RabbitURL:         GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:    GetEnv("RABBITMQ_EXCHANGE", "tasks.direct"),
		InitAdminPassword: GetEnv("INIT_ADMIN_PASSWORD", "admin123"),
		JWTSecret:         GetEnv("CONTROL_JWT_SECRET", "change-me"),
		ArtifactRoot:      GetEnv("ARTIFACT_ROOT", "./data/artifacts"),
		ExecutionAPIKey:   GetEnv("EXECUTION_CALLBACK_KEY", "change-me-callback"),
		CORSAllowOrigins:  GetEnv("CONTROL_CORS_ALLOW_ORIGINS", "http://localhost:5109,http://127.0.0.1:5109"),
	}
}

func LoadExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Environment:        GetEnv("ARANEAE_ENV", "development"),
		HTTPAddr:           GetEnv("EXECUTOR_HTTP_ADDR", ":4280"),
		DBPath:             GetEnv("EXECUTOR_DB_PATH", "./data/executor.db"),
		RabbitURL:          GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:     GetEnv("RABBITMQ_EXCHANGE", "tasks.direct"),
		RabbitQueue:        GetEnv("EXECUTOR_QUEUE", "default"),
		NodeAuthKey:        GetEnv("EXECUTOR_NODE_KEY", ""),
		NodeAuthKeyFile:    GetEnv("EXECUTOR_NODE_KEY_FILE", "./data/executor.node.key"),
		ControlGRPCAddr:    GetEnv("CONTROL_GRPC_TARGET", "localhost:9190"),
		ControlGRPCTLSEnabled: GetEnvBool("EXECUTOR_CONTROL_GRPC_TLS_ENABLED", false),
		ControlGRPCTLSCAFile: GetEnv("EXECUTOR_CONTROL_GRPC_TLS_CA_FILE", ""),
		ControlGRPCTLSServerName: GetEnv("EXECUTOR_CONTROL_GRPC_TLS_SERVER_NAME", ""),
		ExecutorGRPCTLSCertFile: GetEnv("EXECUTOR_GRPC_TLS_CERT_FILE", ""),
		ExecutorGRPCTLSKeyFile: GetEnv("EXECUTOR_GRPC_TLS_KEY_FILE", ""),
		ControlHTTPBase:    GetEnv("CONTROL_HTTP_BASE", "http://localhost:8180"),
		ControlCallbackKey: GetEnv("EXECUTION_CALLBACK_KEY", "change-me-callback"),
		WorkDir:            GetEnv("EXECUTOR_WORKDIR", "./data/workdir"),
	}
}
