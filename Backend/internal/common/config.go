package common

import (
	"os"
	"strconv"
)

type ControlConfig struct {
	HTTPAddr         string
	GRPCAddr         string
	DBPath           string
	RabbitURL        string
	RabbitExchange   string
	JWTSecret        string
	ArtifactRoot     string
	ExecutionAPIKey  string
	CORSAllowOrigins string
}

type ExecutorConfig struct {
	HTTPAddr           string
	DBPath             string
	RabbitURL          string
	RabbitExchange     string
	RabbitQueue        string
	NodeAuthKey        string
	NodeAuthKeyFile    string
	ControlGRPCAddr    string
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

func LoadControlConfig() ControlConfig {
	return ControlConfig{
		HTTPAddr:         GetEnv("CONTROL_HTTP_ADDR", ":8180"),
		GRPCAddr:         GetEnv("CONTROL_GRPC_ADDR", ":9190"),
		DBPath:           GetEnv("CONTROL_DB_PATH", "./data/control.db"),
		RabbitURL:        GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:   GetEnv("RABBITMQ_EXCHANGE", "tasks.direct"),
		JWTSecret:        GetEnv("CONTROL_JWT_SECRET", "change-me"),
		ArtifactRoot:     GetEnv("ARTIFACT_ROOT", "./data/artifacts"),
		ExecutionAPIKey:  GetEnv("EXECUTION_CALLBACK_KEY", "change-me-callback"),
		CORSAllowOrigins: GetEnv("CONTROL_CORS_ALLOW_ORIGINS", "http://localhost:5109,http://127.0.0.1:5109"),
	}
}

func LoadExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		HTTPAddr:           GetEnv("EXECUTOR_HTTP_ADDR", ":4280"),
		DBPath:             GetEnv("EXECUTOR_DB_PATH", "./data/executor.db"),
		RabbitURL:          GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:     GetEnv("RABBITMQ_EXCHANGE", "tasks.direct"),
		RabbitQueue:        GetEnv("EXECUTOR_QUEUE", "default"),
		NodeAuthKey:        GetEnv("EXECUTOR_NODE_KEY", ""),
		NodeAuthKeyFile:    GetEnv("EXECUTOR_NODE_KEY_FILE", "./data/executor.node.key"),
		ControlGRPCAddr:    GetEnv("CONTROL_GRPC_TARGET", "localhost:9190"),
		ControlHTTPBase:    GetEnv("CONTROL_HTTP_BASE", "http://localhost:8180"),
		ControlCallbackKey: GetEnv("EXECUTION_CALLBACK_KEY", "change-me-callback"),
		WorkDir:            GetEnv("EXECUTOR_WORKDIR", "./data/workdir"),
	}
}
