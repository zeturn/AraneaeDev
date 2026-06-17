package runtimeexec

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	maxUnzipEntries = 10000
	maxUnzipBytes   = 500 * 1024 * 1024
)

func ComputeSHA256(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

func UnzipBytes(data []byte, dest string) error {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if len(r.File) > maxUnzipEntries {
		return errors.New("archive contains too many entries")
	}
	var totalUncompressed uint64
	for _, f := range r.File {
		totalUncompressed += f.UncompressedSize64
		if totalUncompressed > maxUnzipBytes {
			return errors.New("archive uncompressed size exceeds limit")
		}
		if err := extractZipEntry(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, dest string) error {
	name := filepath.Clean(f.Name)
	if filepath.IsAbs(name) {
		return os.ErrPermission
	}
	if strings.Contains(name, "..") {
		return os.ErrPermission
	}
	target := filepath.Join(dest, name)
	if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
		return os.ErrPermission
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	src, err := f.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

func RunCommand(ctx context.Context, workDir, command string, extraEnv map[string]string) (string, int, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "bash", "-lc", command)
	}
	cmd.Dir = workDir
	if len(extraEnv) > 0 {
		env := os.Environ()
		for k, v := range extraEnv {
			env = append(env, k+"="+v)
		}
		cmd.Env = env
	}
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	return string(out), exitCode, err
}
