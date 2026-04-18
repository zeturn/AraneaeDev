package control

import (
	"testing"

	"araneae-go/internal/common"

	"go.uber.org/zap"
)

func secureControlConfigForTest() common.ControlConfig {
	return common.ControlConfig{
		Environment:       "production",
		JWTSecret:         "0123456789abcdef0123456789abcdef",
		ExecutionAPIKey:   "abcdefghijklmnopqrstuvwxyz012345",
		InitAdminPassword: "very-strong-password",
		NodeVerifyScheme:  "https",
		GRPCTLSEnabled:    true,
		GRPCTLSCertFile:   "/tmp/cert.pem",
		GRPCTLSKeyFile:    "/tmp/key.pem",
		CORSAllowOrigins:  "https://front.example.com",
		RabbitURL:         "amqp://user:strong-pass@rabbitmq:5672/",
	}
}

func TestValidateSecurityConfigRejectsGuestRabbitCredsInProduction(t *testing.T) {
	cfg := secureControlConfigForTest()
	cfg.RabbitURL = "amqp://guest:guest@rabbitmq:5672/"

	err := validateSecurityConfig(&cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected guest rabbit credentials to be rejected")
	}
}

func TestValidateSecurityConfigRejectsWildcardCORSInProduction(t *testing.T) {
	cfg := secureControlConfigForTest()
	cfg.CORSAllowOrigins = "*"

	err := validateSecurityConfig(&cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected wildcard cors to be rejected")
	}
}

func TestValidateSecurityConfigRequiresStrongerSecretsInProduction(t *testing.T) {
	cfg := secureControlConfigForTest()
	cfg.JWTSecret = "short-secret"

	err := validateSecurityConfig(&cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected weak jwt secret to be rejected")
	}

	cfg = secureControlConfigForTest()
	cfg.ExecutionAPIKey = "too-short"
	err = validateSecurityConfig(&cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected weak callback secret to be rejected")
	}
}
