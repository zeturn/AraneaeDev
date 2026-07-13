package executor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"araneae-go/internal/executor/contracts"

	"go.uber.org/zap"
)

type sinkEnvelope struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
	EmittedAt string          `json:"emitted_at,omitempty"`
}

type sinkStats struct {
	Total      int
	Timeseries int
	Text       int
	Structured int
	Failed     int
}

func (a *App) processSinkArtifacts(ctx context.Context, msg contracts.QueueTaskMessage, runDir string) (string, error) {
	if !a.cfg.SinkEnabled || strings.TrimSpace(a.cfg.HashSlipBaseURL) == "" || runDir == "" {
		return "", nil
	}
	sinkDir := filepath.Join(runDir, filepath.FromSlash(strings.TrimSpace(a.cfg.SinkDirName)))
	if strings.TrimSpace(a.cfg.SinkDirName) == "" || sinkDir == runDir {
		sinkDir = filepath.Join(runDir, ".araneae", "sink")
	}
	if _, err := os.Stat(sinkDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	files, err := os.ReadDir(sinkDir)
	if err != nil {
		return "", err
	}
	stats := sinkStats{}
	var allErrs []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := strings.ToLower(f.Name())
		if !strings.HasSuffix(name, ".jsonl") && !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(sinkDir, f.Name())
		fileStats, err := a.processSinkFile(ctx, msg, path)
		stats.Total += fileStats.Total
		stats.Timeseries += fileStats.Timeseries
		stats.Text += fileStats.Text
		stats.Structured += fileStats.Structured
		stats.Failed += fileStats.Failed
		if err != nil {
			allErrs = append(allErrs, fmt.Sprintf("%s: %v", f.Name(), err))
		}
	}
	if stats.Total == 0 {
		return "", nil
	}
	summary := fmt.Sprintf("sink forwarded total=%d timeseries=%d text=%d structured=%d failed=%d",
		stats.Total, stats.Timeseries, stats.Text, stats.Structured, stats.Failed)
	if len(allErrs) == 0 {
		return summary, nil
	}
	return summary, errors.New(strings.Join(allErrs, "; "))
}

func (a *App) processSinkFile(ctx context.Context, msg contracts.QueueTaskMessage, path string) (sinkStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return sinkStats{}, err
	}
	defer f.Close()

	stats := sinkStats{}
	scanner := bufio.NewScanner(f)
	const maxLine = 4 * 1024 * 1024
	scanner.Buffer(make([]byte, 64*1024), maxLine)
	var fileErrs []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		stats.Total++
		var env sinkEnvelope
		if err := json.Unmarshal([]byte(line), &env); err != nil {
			stats.Failed++
			fileErrs = append(fileErrs, fmt.Sprintf("invalid json line: %v", err))
			continue
		}
		t := strings.ToLower(strings.TrimSpace(env.Type))
		switch t {
		case "timeseries":
			stats.Timeseries++
		case "text":
			stats.Text++
		case "structured":
			stats.Structured++
		default:
			stats.Failed++
			fileErrs = append(fileErrs, "unsupported type "+t)
			continue
		}
		if len(env.Data) == 0 {
			stats.Failed++
			fileErrs = append(fileErrs, "empty data for "+t)
			continue
		}
		if err := a.forwardSinkEvent(ctx, msg, t, env.Data); err != nil {
			stats.Failed++
			fileErrs = append(fileErrs, fmt.Sprintf("%s forward failed: %v", t, err))
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		fileErrs = append(fileErrs, scanErr.Error())
	}
	if len(fileErrs) == 0 {
		return stats, nil
	}
	return stats, errors.New(strings.Join(fileErrs, "; "))
}

func (a *App) forwardSinkEvent(ctx context.Context, msg contracts.QueueTaskMessage, eventType string, payload json.RawMessage) error {
	if eventType == "structured" {
		if datasetID := slotDatasetID(msg.Metadata); datasetID != "" {
			var envelope struct {
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(payload, &envelope); err != nil {
				return fmt.Errorf("decode structured sink event: %w", err)
			}
			if envelope.Data == nil {
				return errors.New("structured sink event has no data")
			}
			payload, _ = json.Marshal(map[string]any{"data": envelope.Data})
			return a.forwardToHashSlip(ctx, msg, "/api/v1/datasets/"+url.PathEscape(datasetID)+"/records", payload)
		}
	}
	var endpoint string
	switch eventType {
	case "timeseries":
		endpoint = strings.TrimSpace(a.cfg.HashSlipTimeseriesPath)
	case "text":
		endpoint = strings.TrimSpace(a.cfg.HashSlipTextPath)
	case "structured":
		endpoint = strings.TrimSpace(a.cfg.HashSlipStructuredPath)
	default:
		return errors.New("unsupported event type")
	}
	if endpoint == "" {
		return errors.New("empty hashslip endpoint path")
	}
	return a.forwardToHashSlip(ctx, msg, endpoint, payload)
}

func slotDatasetID(metadata map[string]any) string {
	slot, ok := metadata["hashslip_slot"].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := slot["dataset_id"].(string)
	return strings.TrimSpace(value)
}

func (a *App) forwardToHashSlip(ctx context.Context, msg contracts.QueueTaskMessage, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.cfg.HashSlipBaseURL, "/")+ensureLeadingSlash(endpoint), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Araneae-Run-Id", msg.RunID)
	req.Header.Set("X-Araneae-Task-Id", msg.TaskID)
	req.Header.Set("X-Araneae-Correlation-Id", msg.CorrelationID)

	token, err := a.getHashSlipBearerToken(ctx)
	if err != nil {
		a.log.Warn("get basalt s2s token failed; continue without bearer", zap.Error(err), zap.String("run_id", msg.RunID))
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := a.httpClient
	if timeoutSec := a.cfg.HashSlipTimeoutSeconds; timeoutSec > 0 {
		client = &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("hashslip status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (a *App) getHashSlipBearerToken(ctx context.Context) (string, error) {
	tokenURL := strings.TrimSpace(a.cfg.BasaltTokenURL)
	clientID := strings.TrimSpace(a.cfg.BasaltClientID)
	clientSecret := strings.TrimSpace(a.cfg.BasaltClientSecret)
	if tokenURL == "" || clientID == "" || clientSecret == "" {
		return "", nil
	}

	a.tokenMu.Lock()
	if a.tokenValue != "" && time.Now().Before(a.tokenUntil) {
		token := a.tokenValue
		a.tokenMu.Unlock()
		return token, nil
	}
	a.tokenMu.Unlock()

	form := url.Values{}
	if subjectToken := strings.TrimSpace(a.cfg.BasaltSubjectToken); subjectToken != "" {
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange")
		form.Set("subject_token", subjectToken)
		form.Set("subject_token_type", "urn:ietf:params:oauth:token-type:access_token")
		form.Set("resource", strings.TrimSpace(a.cfg.BasaltTargetResource))
	} else {
		form.Set("grant_type", "client_credentials")
	}
	if scope := strings.TrimSpace(a.cfg.BasaltClientScope); scope != "" {
		form.Set("scope", scope)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", errors.New("empty access_token from token endpoint")
	}
	expiresIn := tokenResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 300
	}
	a.tokenMu.Lock()
	a.tokenValue = strings.TrimSpace(tokenResp.AccessToken)
	a.tokenUntil = time.Now().Add(time.Duration(expiresIn-30) * time.Second)
	token := a.tokenValue
	a.tokenMu.Unlock()
	return token, nil
}

func ensureLeadingSlash(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if strings.HasPrefix(trimmed, "/") {
		return trimmed
	}
	return "/" + trimmed
}

func ensureSinkSDK(runDir string) error {
	target := filepath.Join(runDir, "araneae_sink.py")
	return os.WriteFile(target, []byte(araneaeSinkSDKPython), 0o644)
}

const araneaeSinkSDKPython = `import json
import os
from datetime import datetime, timezone
from pathlib import Path

_MODE = os.getenv("ARANEAE_SINK_MODE", "auto").strip().lower()
if _MODE == "auto":
    _MODE = "araneae" if os.getenv("ARANEAE_RUNTIME", "") == "1" else "local"
_SINK_DIR = os.getenv("ARANEAE_SINK_DIR", "./.araneae/sink").strip() or "./.araneae/sink"
_LOCAL_FILE = os.getenv("ARANEAE_SINK_FILE", "./araneae_sink.local.jsonl").strip() or "./araneae_sink.local.jsonl"


def _now_iso():
    return datetime.now(timezone.utc).isoformat()


def _append_jsonl(path: Path, payload: dict):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as f:
        f.write(json.dumps(payload, ensure_ascii=False))
        f.write("\n")


def emit(event_type: str, data: dict):
    env = {
        "type": str(event_type).strip().lower(),
        "data": data or {},
        "emitted_at": _now_iso(),
    }
    if _MODE == "araneae":
        _append_jsonl(Path(_SINK_DIR) / "events.jsonl", env)
        print(f"[araneae_sink] queued type={env['type']}")
    else:
        _append_jsonl(Path(_LOCAL_FILE), env)
        print(f"[araneae_sink] local type={env['type']} data={json.dumps(env['data'], ensure_ascii=False)}")


def emit_timeseries(source: str, metric: str, timestamp: str, value, tags=None, payload=None, hash_key: str = "", bucket_date: str = ""):
    emit("timeseries", {
        "source": source,
        "metric": metric,
        "timestamp": timestamp,
        "value": value,
        "tags": tags or {},
        "payload": payload or {},
        "hash_key": hash_key,
        "bucket_date": bucket_date,
    })


def emit_text(source: str, title: str, url: str, content: str, published_at: str = "", tags=None, metadata=None):
    emit("text", {
        "source": source,
        "title": title,
        "url": url,
        "content": content,
        "published_at": published_at,
        "tags": tags or {},
        "metadata": metadata or {},
    })


def emit_structured(instance_id: str, schema_id: str, data: dict):
    emit("structured", {
        "instance_id": instance_id,
        "schema_id": schema_id,
        "data": data or {},
    })
`
