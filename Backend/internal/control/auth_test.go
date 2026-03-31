package control

import (
	"testing"

	"araneae-go/internal/common"
)

func TestIssueAndParseToken(t *testing.T) {
	app := &App{cfg: common.ControlConfig{JWTSecret: "unit-test-secret"}}
	tok, err := app.issueToken("u-1", "admin")
	if err != nil {
		t.Fatalf("issue token failed: %v", err)
	}
	claims, err := app.parseToken(tok)
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if claims.UserID != "u-1" {
		t.Fatalf("unexpected user id: %s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Fatalf("unexpected role: %s", claims.Role)
	}
}

func TestWriteArtifactFile(t *testing.T) {
	tmp := t.TempDir()
	data := []byte("hello-artifact")
	path, sha, err := writeArtifactFile(tmp, "project-x", "version-y", "../payload.zip", data)
	if err != nil {
		t.Fatalf("write artifact file failed: %v", err)
	}
	if sha != computeSHA256(data) {
		t.Fatalf("sha mismatch: got=%s want=%s", sha, computeSHA256(data))
	}
	if path == "" {
		t.Fatal("empty storage path")
	}
}
