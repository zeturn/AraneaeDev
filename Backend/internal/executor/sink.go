package executor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

	"github.com/google/uuid"
)

type sinkEnvelope struct {
	MessageType    string          `json:"message_type"`
	SchemaVersion  string          `json:"schema_version"`
	EventID        string          `json:"event_id"`
	TraceID        string          `json:"trace_id"`
	CausationID    string          `json:"causation_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key"`
	Producer       string          `json:"producer"`
	RetryCount     int             `json:"retry_count"`
	OccurredAt     string          `json:"occurred_at"`
	Correlation    map[string]any  `json:"correlation"`
	Data           json.RawMessage `json:"data"`
	Metadata       map[string]any  `json:"metadata"`
	Source         map[string]any  `json:"source,omitempty"`
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
		if err := validateSinkEnvelope(env); err != nil {
			stats.Failed++
			fileErrs = append(fileErrs, err.Error())
			continue
		}
		t := strings.ToLower(strings.TrimSpace(sinkStringValue(env.Metadata["record_type"])))
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
	datasetID := slotDatasetID(msg.Metadata)
	if datasetID == "" {
		return errors.New("sink requires hashslip_slot.dataset_id")
	}
	var record map[string]any
	if err := json.Unmarshal(payload, &record); err != nil {
		return fmt.Errorf("decode %s sink event: %w", eventType, err)
	}
	if eventType == "structured" {
		// Structured records store the source record below instance_id/schema_id/
		// data. HashSlip slots are schema-bearing, so forward the source record
		// itself; wrapping the envelope would make schema fields and dedupe keys
		// appear null to downstream readers.
		record = mapFromAny(record["data"])
		if len(record) == 0 {
			return errors.New("structured sink event has no data")
		}
	}
	envelope := map[string]any{
		"data":     record,
		"metadata": canonicalRecordMetadata(msg, eventType, record),
	}
	if source := canonicalRecordSource(msg, record); len(source) > 0 {
		envelope["source"] = source
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode %s sink event: %w", eventType, err)
	}
	return a.forwardToHashSlip(ctx, msg, "/api/v1/datasets/"+url.PathEscape(datasetID)+"/records", payload)
}

func validateSinkEnvelope(env sinkEnvelope) error {
	if env.MessageType != "araneae.record.ready" {
		return fmt.Errorf("unsupported sink message_type %q", env.MessageType)
	}
	if env.SchemaVersion != "1.0" {
		return fmt.Errorf("unsupported sink schema_version %q", env.SchemaVersion)
	}
	for name, value := range map[string]string{
		"event_id": env.EventID, "trace_id": env.TraceID,
		"idempotency_key": env.IdempotencyKey, "producer": env.Producer,
		"occurred_at": env.OccurredAt,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("sink envelope %s is required", name)
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, env.OccurredAt); err != nil {
		return fmt.Errorf("sink envelope occurred_at is invalid: %w", err)
	}
	if env.Correlation == nil || env.Metadata == nil {
		return errors.New("sink envelope correlation and metadata are required")
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return errors.New("sink envelope data is required")
	}
	return nil
}

func appendSinkEnvelope(sinkDir, recordType string, data json.RawMessage, metadata map[string]any, source string) error {
	eventID := "evt_" + uuid.NewString()
	traceID := firstNonEmptySink(os.Getenv("ARANEAE_TRACE_ID"), eventID)
	runID := strings.TrimSpace(os.Getenv("ARANEAE_RUN_ID"))
	taskID := strings.TrimSpace(os.Getenv("ARANEAE_TASK_ID"))
	correlation := map[string]any{"trace_id": traceID, "run_id": runID, "task_id": taskID}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["record_type"] = strings.ToLower(strings.TrimSpace(recordType))
	env := sinkEnvelope{
		MessageType:    "araneae.record.ready",
		SchemaVersion:  "1.0",
		EventID:        eventID,
		TraceID:        traceID,
		CausationID:    runID,
		IdempotencyKey: "araneae.record:" + eventID,
		Producer:       "araneae",
		RetryCount:     0,
		OccurredAt:     time.Now().UTC().Format(time.RFC3339Nano),
		Correlation:    correlation,
		Data:           data,
		Metadata:       metadata,
	}
	if strings.TrimSpace(source) != "" {
		env.Source = map[string]any{"name": strings.TrimSpace(source)}
	}
	line, err := json.Marshal(env)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(sinkDir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(sinkDir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(line, '\n'))
	return err
}

func firstNonEmptySink(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func sinkStringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

// canonicalRecordSource projects the stable source identity into HashSlip's
// data envelope. The source identity deliberately excludes run-specific
// values so source_count remains a count of sources rather than deliveries.
func canonicalRecordSource(msg contracts.QueueTaskMessage, record map[string]any) map[string]any {
	name := firstNonEmptyRecordString(
		recordString(record, "source"),
		recordString(record, "source_name"),
		recordString(record, "source_url"),
		recordString(record, "url"),
		recordString(record, "link"),
		strings.TrimSpace(msg.SourceURL),
		strings.TrimSpace(msg.ProjectID),
	)
	if name == "" {
		return nil
	}
	return map[string]any{"name": name}
}

func canonicalRecordMetadata(msg contracts.QueueTaskMessage, eventType string, record map[string]any) map[string]any {
	metadata := map[string]any{"record_type": eventType}
	set := func(key, value string) {
		if strings.TrimSpace(value) != "" {
			metadata[key] = strings.TrimSpace(value)
		}
	}
	set("run_id", msg.RunID)
	set("task_id", msg.TaskID)
	set("correlation_id", msg.CorrelationID)
	set("project_id", msg.ProjectID)
	set("version_id", msg.VersionID)
	set("source_url", firstNonEmptyRecordString(
		recordString(record, "source_url"),
		recordString(record, "url"),
		recordString(record, "link"),
		msg.SourceURL,
	))
	return metadata
}

func firstNonEmptyRecordString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func recordString(record map[string]any, key string) string {
	if record == nil {
		return ""
	}
	value, ok := record[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func mapFromAny(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func slotDatasetID(metadata map[string]any) string {
	for _, slot := range sinkSlots(metadata) {
		if value, ok := slot["dataset_id"].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (a *App) forwardToHashSlip(ctx context.Context, msg contracts.QueueTaskMessage, endpoint string, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.cfg.HashSlipBaseURL, "/")+ensureLeadingSlash(endpoint), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// X-Pipeline-* is the sole HTTP projection of the canonical pipeline
	// envelope. Product-specific fields remain in the request body or slot
	// metadata; these headers carry only transport provenance.
	// HashSlip uses this value to make a retried HTTP transfer resolve to the
	// same durable event. The payload hash is stable for a given sink result.
	idempotency := sha256.Sum256(append([]byte(msg.RunID+"\n"+msg.TaskID+"\n"+endpoint+"\n"), payload...))
	idempotencyKey := "araneae:" + hex.EncodeToString(idempotency[:])
	eventID := "evt_" + hex.EncodeToString(idempotency[:16])
	traceID := strings.TrimSpace(msg.CorrelationID)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	req.Header.Set("X-Pipeline-Schema-Version", "1.0")
	req.Header.Set("X-Pipeline-Message-Type", "hashslip.record.ingest")
	req.Header.Set("X-Pipeline-Event-Id", eventID)
	req.Header.Set("X-Pipeline-Trace-Id", traceID)
	req.Header.Set("X-Pipeline-Causation-Id", strings.TrimSpace(msg.RunID))
	req.Header.Set("X-Pipeline-Correlation-Id", traceID)
	req.Header.Set("X-Pipeline-Idempotency-Key", idempotencyKey)
	req.Header.Set("X-Pipeline-Producer", "araneae")
	req.Header.Set("X-Pipeline-Occurred-At", now)
	setPipelineHeader := func(name string, value string) {
		if value != "" {
			req.Header.Set(name, value)
		}
	}
	setPipelineHeader("X-Pipeline-Run-Id", strings.TrimSpace(msg.RunID))
	setPipelineHeader("X-Pipeline-Task-Id", strings.TrimSpace(msg.TaskID))
	setPipelineHeader("X-Pipeline-Schedule-Id", metadataString(msg.Metadata, "hashslip_schedule_id", "schedule_id"))
	setPipelineHeader("X-Pipeline-Objective-Id", metadataString(msg.Metadata, "cis_objective_id", "objective_id"))
	setPipelineHeader("X-Pipeline-Artifact-Id", metadataString(msg.Metadata, "cis_artifact_id", "artifact_id"))
	setPipelineHeader("X-Pipeline-Project-Id", strings.TrimSpace(msg.ProjectID))
	setPipelineHeader("X-Pipeline-Version-Id", strings.TrimSpace(msg.VersionID))
	setPipelineHeader("X-Pipeline-Record-Type", strings.TrimSpace(msg.Type))
	if slots := sinkSlots(msg.Metadata); len(slots) > 0 {
		if raw, marshalErr := json.Marshal(slots); marshalErr == nil {
			req.Header.Set("X-Pipeline-Slots", string(raw))
		}
	}

	token, err := a.getHashSlipBearerToken(ctx)
	if err != nil {
		return fmt.Errorf("get basalt s2s token: %w", err)
	}
	if token == "" {
		return errors.New("get basalt s2s token: empty token")
	}
	req.Header.Set("Authorization", "Bearer "+token)

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

func sinkSlots(metadata map[string]any) []map[string]any {
	if metadata == nil {
		return nil
	}
	if raw, ok := metadata["hashslip_slots"].([]any); ok {
		slots := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if slot, ok := item.(map[string]any); ok && len(slot) > 0 {
				slots = append(slots, slot)
			}
		}
		if len(slots) > 0 {
			return slots
		}
	}
	if slot, ok := metadata["hashslip_slot"].(map[string]any); ok && len(slot) > 0 {
		return []map[string]any{slot}
	}
	return nil
}

func metadataString(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
import uuid
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
    event_id = "evt_" + uuid.uuid4().hex
    trace_id = os.getenv("ARANEAE_TRACE_ID", "").strip() or event_id
    run_id = os.getenv("ARANEAE_RUN_ID", "").strip()
    task_id = os.getenv("ARANEAE_TASK_ID", "").strip()
    source_name = str((data or {}).get("source") or "").strip()
    env = {
        "message_type": "araneae.record.ready",
        "schema_version": "1.0",
        "event_id": event_id,
        "trace_id": trace_id,
        "causation_id": run_id,
        "idempotency_key": "araneae.record:" + event_id,
        "producer": "araneae",
        "retry_count": 0,
        "occurred_at": _now_iso(),
        "correlation": {"trace_id": trace_id, "run_id": run_id, "task_id": task_id},
        "data": data or {},
        "metadata": {"record_type": str(event_type).strip().lower()},
    }
    if source_name:
        env["source"] = {"name": source_name}
    if _MODE == "araneae":
        _append_jsonl(Path(_SINK_DIR) / "events.jsonl", env)
        print(f"[araneae_sink] queued type={env['metadata']['record_type']}")
    else:
        _append_jsonl(Path(_LOCAL_FILE), env)
        print(f"[araneae_sink] local type={env['metadata']['record_type']} data={json.dumps(env['data'], ensure_ascii=False)}")


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
