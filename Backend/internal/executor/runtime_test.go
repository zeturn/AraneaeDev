package executor

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"araneae-go/internal/executor/runtimeexec"
)

func zipWithEntries(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry failed: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry failed: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}
	return b.Bytes()
}

func TestUnzipBytesRejectsTraversal(t *testing.T) {
	archive := zipWithEntries(t, map[string]string{"../evil.sh": "echo hacked"})
	err := runtimeexec.UnzipBytes(archive, t.TempDir())
	if err == nil {
		t.Fatal("expected traversal error, got nil")
	}
}

func TestUnzipBytesRejectsAbsolutePath(t *testing.T) {
	entry := "/etc/passwd"
	if runtime.GOOS == "windows" {
		entry = `C:\Windows\system32\drivers\etc\hosts`
	}
	archive := zipWithEntries(t, map[string]string{entry: "blocked"})
	err := runtimeexec.UnzipBytes(archive, t.TempDir())
	if err == nil {
		t.Fatal("expected absolute path error, got nil")
	}
}

func TestUnzipBytesRejectsTooManyEntries(t *testing.T) {
	var b bytes.Buffer
	zw := zip.NewWriter(&b)
	for i := 0; i < 10001; i++ {
		name := fmt.Sprintf("f_%05d.txt", i)
		if _, err := zw.Create(name); err != nil {
			t.Fatalf("create zip entry failed: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer failed: %v", err)
	}

	err := runtimeexec.UnzipBytes(b.Bytes(), t.TempDir())
	if err == nil {
		t.Fatal("expected too many entries error, got nil")
	}
}

func TestUnzipBytesExtractsFiles(t *testing.T) {
	dir := t.TempDir()
	archive := zipWithEntries(t, map[string]string{"run.sh": "echo ok"})
	if err := runtimeexec.UnzipBytes(archive, dir); err != nil {
		t.Fatalf("unzip failed: %v", err)
	}
	p := filepath.Join(dir, "run.sh")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("expected extracted file: %v", err)
	}
}

func TestRunCommand(t *testing.T) {
	dir := t.TempDir()
	out, code, err := runtimeexec.RunCommand(context.Background(), dir, "echo test-run", nil)
	if err != nil {
		t.Fatalf("runCommand failed: %v", err)
	}
	if code != 0 {
		t.Fatalf("unexpected exit code: %d", code)
	}
	if !bytes.Contains([]byte(out), []byte("test-run")) {
		t.Fatalf("unexpected output: %s", out)
	}
}
