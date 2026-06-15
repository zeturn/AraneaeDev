package common

import (
	"os"
	"strconv"
	"strings"
)

type ControlConfig struct {
	Environment           string
	HTTPAddr              string
	GRPCAddr              string
	GRPCTLSEnabled        bool
	GRPCTLSCertFile       string
	GRPCTLSKeyFile        string
	GRPCTLSClientCAFile   string
	NodeVerifyScheme      string
	DBPath                string
	RabbitURL             string
	RabbitExchange        string
	InitAdminPassword     string
	JWTSecret             string
	ArtifactRoot          string
	RSSRoot               string
	ExecutionAPIKey       string
	CORSAllowOrigins      string
	FrontendBaseURL       string
	BasaltBaseURL         string
	BasaltInternalBaseURL string
	BasaltOAuthEnabled    bool
	BasaltClientID        string
	BasaltClientSecret    string
	BasaltRedirectURI     string
	BasaltScope           string
	BasaltCallbackPath    string
	BasaltRoleClaimKeys   string
	BasaltGroupClaimKeys  string
	BasaltAdminEmails     string
	BasaltS2SScopes       string
	BasaltTeamSyncEnabled bool
	BasaltTeamSyncPrune   bool
	BasaltTeamPrefix      string
}

type ExecutorConfig struct {
	Environment              string
	HTTPAddr                 string
	DBPath                   string
	RabbitURL                string
	RabbitExchange           string
	RabbitQueue              string
	NodeAuthKey              string
	NodeAuthKeyFile          string
	ControlGRPCAddr          string
	ControlGRPCTLSEnabled    bool
	ControlGRPCTLSCAFile     string
	ControlGRPCTLSServerName string
	ExecutorGRPCTLSCertFile  string
	ExecutorGRPCTLSKeyFile   string
	ControlHTTPBase          string
	ControlCallbackKey       string
	TaskTimeoutSeconds       int
	WorkDir                  string
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
		Environment:           GetEnv("ARANEAE_ENV", "development"),
		HTTPAddr:              GetEnv("CONTROL_HTTP_ADDR", ":8180"),
		GRPCAddr:              GetEnv("CONTROL_GRPC_ADDR", ":9190"),
		GRPCTLSEnabled:        GetEnvBool("CONTROL_GRPC_TLS_ENABLED", false),
		GRPCTLSCertFile:       GetEnv("CONTROL_GRPC_TLS_CERT_FILE", ""),
		GRPCTLSKeyFile:        GetEnv("CONTROL_GRPC_TLS_KEY_FILE", ""),
		GRPCTLSClientCAFile:   GetEnv("CONTROL_GRPC_TLS_CLIENT_CA_FILE", ""),
		NodeVerifyScheme:      GetEnv("CONTROL_NODE_VERIFY_SCHEME", "http"),
		DBPath:                GetEnv("CONTROL_DB_PATH", "./data/control.db"),
		RabbitURL:             GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:        GetEnv("RABBITMQ_EXCHANGE", "tasks.direct"),
		InitAdminPassword:     GetEnv("INIT_ADMIN_PASSWORD", ""),
		JWTSecret:             GetEnv("CONTROL_JWT_SECRET", ""),
		ArtifactRoot:          GetEnv("ARTIFACT_ROOT", "./data/artifacts"),
		RSSRoot:               GetEnv("RSS_ROOT", "./data/rss"),
		ExecutionAPIKey:       GetEnv("EXECUTION_CALLBACK_KEY", ""),
		CORSAllowOrigins:      GetEnv("CONTROL_CORS_ALLOW_ORIGINS", "http://localhost:5109,http://127.0.0.1:5109"),
		FrontendBaseURL:       GetEnv("FRONTEND_BASE_URL", "http://localhost:5109"),
		BasaltBaseURL:         GetEnv("BASALTPASS_BASE_URL", "http://localhost:8101"),
		BasaltInternalBaseURL: GetEnv("BASALTPASS_INTERNAL_BASE_URL", GetEnv("BASALTPASS_BASE_URL", "http://localhost:8101")),
		BasaltOAuthEnabled:    GetEnvBool("BASALTPASS_OAUTH_ENABLED", false),
		BasaltClientID:        GetEnv("BASALTPASS_OAUTH_CLIENT_ID", ""),
		BasaltClientSecret:    GetEnv("BASALTPASS_OAUTH_CLIENT_SECRET", ""),
		BasaltRedirectURI:     GetEnv("BASALTPASS_OAUTH_REDIRECT_URI", "http://localhost:8180/api/auth/basaltpass/callback/"),
		BasaltScope:           GetEnv("BASALTPASS_OAUTH_SCOPE", "openid profile email"),
		BasaltCallbackPath:    GetEnv("BASALTPASS_FRONTEND_CALLBACK_PATH", "/oauth/callback"),
		BasaltRoleClaimKeys:   GetEnv("BASALTPASS_ROLE_CLAIM_KEYS", "roles,role,app_roles"),
		BasaltGroupClaimKeys:  GetEnv("BASALTPASS_GROUP_CLAIM_KEYS", "groups,group,teams,team"),
		BasaltAdminEmails:     GetEnv("BASALTPASS_ADMIN_EMAILS", "hrzh@ucdavis.edu"),
		BasaltS2SScopes:       GetEnv("BASALTPASS_S2S_SCOPES", "s2s.user.read s2s.team.read"),
		BasaltTeamSyncEnabled: GetEnvBool("BASALTPASS_TEAM_SYNC_ENABLED", true),
		BasaltTeamSyncPrune:   GetEnvBool("BASALTPASS_TEAM_SYNC_PRUNE", false),
		BasaltTeamPrefix:      GetEnv("BASALTPASS_TEAM_PREFIX", "Basalt::"),
	}
}

func LoadExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Environment:              GetEnv("ARANEAE_ENV", "development"),
		HTTPAddr:                 GetEnv("EXECUTOR_HTTP_ADDR", ":4280"),
		DBPath:                   GetEnv("EXECUTOR_DB_PATH", "./data/executor.db"),
		RabbitURL:                GetEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		RabbitExchange:           GetEnv("RABBITMQ_EXCHANGE", "tasks.direct"),
		RabbitQueue:              GetEnv("EXECUTOR_QUEUE", "default"),
		NodeAuthKey:              GetEnv("EXECUTOR_NODE_KEY", ""),
		NodeAuthKeyFile:          GetEnv("EXECUTOR_NODE_KEY_FILE", "./data/executor.node.key"),
		ControlGRPCAddr:          GetEnv("CONTROL_GRPC_TARGET", "localhost:9190"),
		ControlGRPCTLSEnabled:    GetEnvBool("EXECUTOR_CONTROL_GRPC_TLS_ENABLED", false),
		ControlGRPCTLSCAFile:     GetEnv("EXECUTOR_CONTROL_GRPC_TLS_CA_FILE", ""),
		ControlGRPCTLSServerName: GetEnv("EXECUTOR_CONTROL_GRPC_TLS_SERVER_NAME", ""),
		ExecutorGRPCTLSCertFile:  GetEnv("EXECUTOR_GRPC_TLS_CERT_FILE", ""),
		ExecutorGRPCTLSKeyFile:   GetEnv("EXECUTOR_GRPC_TLS_KEY_FILE", ""),
		ControlHTTPBase:          GetEnv("CONTROL_HTTP_BASE", "http://localhost:8180"),
		ControlCallbackKey:       GetEnv("EXECUTION_CALLBACK_KEY", ""),
		TaskTimeoutSeconds:       GetEnvInt("EXECUTOR_TASK_TIMEOUT_SECONDS", 1800),
		WorkDir:                  GetEnv("EXECUTOR_WORKDIR", "./data/workdir"),
	}
}
