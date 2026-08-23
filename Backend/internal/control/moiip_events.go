package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"araneae-go/internal/common"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var collectionTokenRE = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

const pipelineEnvelopeSchemaVersion = "1.0"

type pipelineEnvelope struct {
	MessageType    string         `json:"message_type"`
	SchemaVersion  string         `json:"schema_version"`
	EventID        string         `json:"event_id"`
	TraceID        string         `json:"trace_id"`
	CausationID    string         `json:"causation_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key"`
	Producer       string         `json:"producer"`
	RetryCount     int            `json:"retry_count"`
	OccurredAt     string         `json:"occurred_at"`
	Correlation    map[string]any `json:"correlation"`
	Payload        map[string]any `json:"payload"`
}

func (a *App) publishCrawlSucceededEvent(ctx context.Context, run common.TaskRun) error {
	return a.publishCrawlTerminalEvent(ctx, run)
}

func (a *App) publishCrawlTerminalEvent(ctx context.Context, run common.TaskRun) error {
	if !a.cfg.PipelineEventsEnabled {
		return nil
	}
	envelope, ok, err := a.buildCrawlTerminalEnvelope(ctx, run, run.Status)
	if err != nil || !ok {
		return err
	}
	return a.enqueuePipelineEnvelope(envelope)
}

func (a *App) buildCrawlSucceededEnvelope(ctx context.Context, run common.TaskRun) (pipelineEnvelope, bool, error) {
	return a.buildCrawlTerminalEnvelope(ctx, run, "success")
}

func (a *App) buildCrawlTerminalEnvelope(ctx context.Context, run common.TaskRun, status string) (pipelineEnvelope, bool, error) {
	var task common.Task
	if err := a.db.WithContext(ctx).Where("id = ?", run.TaskID).First(&task).Error; err != nil {
		return pipelineEnvelope{}, false, err
	}
	taskType := strings.ToLower(strings.TrimSpace(task.Type))
	metadata := parseMetadataJSON(task.MetadataJSON)
	hashslipSlot := metadataMap(metadata, "hashslip_slot")
	if taskType != "rss" && taskType != "api" && taskType != "page" && len(hashslipSlot) == 0 {
		return pipelineEnvelope{}, false, nil
	}
	status = normalizeCrawlStatus(status)
	messageStatus := map[string]string{"success": "succeeded", "canceled": "canceled", "failed": "failed"}[status]
	if messageStatus == "" {
		messageStatus = status
	}
	messageType := "araneae.crawl." + messageStatus
	collection := firstNonEmpty(metadataString(metadata, "hashslip_collection"), metadataString(metadata, "collection"), defaultCollectionForTask(task))
	outputCollection := firstNonEmpty(metadataString(metadata, "analysis_collection"), "analysis."+collection)
	missionID := firstNonEmpty(metadataString(metadata, "mission_id"), "mission_"+stableShortHex(collection))
	traceID := firstNonEmpty(metadataString(metadata, "trace_id"), run.CorrelationID, "trace_"+stableShortHex(run.ID))
	datasetID := firstNonEmpty(metadataString(hashslipSlot, "dataset_id"), metadataString(hashslipSlot, "hashslip_dataset_id"), metadataString(metadata, "hashslip_dataset_id"))
	chunkID := firstNonEmpty(metadataString(hashslipSlot, "chunk_id"), metadataString(hashslipSlot, "hashslip_chunk_id"))
	dataType := firstNonEmpty(metadataString(hashslipSlot, "data_type"), metadataString(metadata, "data_type"))
	var project common.Project
	projectPayload := map[string]any{}
	if task.ProjectID != "" && a.db.WithContext(ctx).Where("id = ?", task.ProjectID).First(&project).Error == nil {
		projectPayload = map[string]any{
			"id": project.ID, "name": project.Name, "description": project.Description,
			"language": project.Language, "workplace_id": project.WorkplaceID,
		}
	}
	var schedule common.Schedule
	schedulePayload := map[string]any{}
	if run.ScheduleID != "" && a.db.WithContext(ctx).Where("id = ?", run.ScheduleID).First(&schedule).Error == nil {
		schedulePayload = map[string]any{
			"id": schedule.ID, "name": schedule.Name, "description": schedule.Description,
			"task_id": schedule.TaskID, "project_id": schedule.ProjectID, "version_id": schedule.VersionID,
			"entry_command": schedule.EntryCommand, "cron_expr": schedule.CronExpr,
			"trigger_type": schedule.TriggerType, "run_at": schedule.RunAt, "node_queue": schedule.NodeQueue,
			"enabled": schedule.Enabled, "created_at": schedule.CreatedAt, "updated_at": schedule.UpdatedAt,
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	eventID := "evt_" + stableShortHex(messageType+":"+run.ID)
	envelope := pipelineEnvelope{
		MessageType:    messageType,
		SchemaVersion:  pipelineEnvelopeSchemaVersion,
		EventID:        eventID,
		TraceID:        traceID,
		CausationID:    run.ID,
		IdempotencyKey: messageType + ":" + run.ID,
		Producer:       "araneae",
		RetryCount:     0,
		OccurredAt:     now,
		Correlation: map[string]any{
			"mission_id":   missionID,
			"task_id":      task.ID,
			"run_id":       run.ID,
			"schedule_id":  run.ScheduleID,
			"project_id":   task.ProjectID,
			"version_id":   task.VersionID,
			"objective_id": metadataString(metadata, "cis_objective_id"),
			"trace_id":     traceID,
			"collection":   collection,
		},
		Payload: map[string]any{
			"task_id":             task.ID,
			"task_name":           task.Name,
			"task_type":           taskType,
			"status":              status,
			"run_id":              run.ID,
			"schedule_id":         run.ScheduleID,
			"project_id":          task.ProjectID,
			"version_id":          task.VersionID,
			"entry_command":       task.EntryCommand,
			"node_queue":          run.NodeQueue,
			"trigger_source":      run.TriggerSource,
			"chain_id":            run.ChainID,
			"chain_index":         run.ChainIndex,
			"chain_total":         run.ChainTotal,
			"correlation_id":      run.CorrelationID,
			"source_url":          task.SourceURL,
			"hashslip_collection": collection,
			"hashslip_dataset_id": datasetID,
			"dataset": map[string]any{
				"dataset_id": datasetID,
				"chunk_id":   chunkID,
				"data_type":  dataType,
				"collection": collection,
			},
			"hashslip_slot":       hashslipSlot,
			"schema_id":           firstNonEmpty(metadataString(metadata, "schema_id"), collection),
			"analysis_collection": outputCollection,
			"artifact_type":       firstNonEmpty(metadataString(metadata, "artifact_type"), "analysis_result"),
			"vesper_job":          metadata["vesper_job"],
			"cis_objective_id":    metadataString(metadata, "cis_objective_id"),
			"cis_artifact_id":     metadataString(metadata, "cis_artifact_id"),
			"cis_plan_id":         metadataString(metadata, "cis_plan_id"),
			"project":             projectPayload,
			"schedule":            schedulePayload,
			"started_at":          timePtrString(run.StartedAt),
			"sink_summary":        run.Output,
			"output":              run.Output,
			"exit_code":           run.ExitCode,
			"finished_at":         timePtrString(run.FinishedAt),
			"created_at":          run.CreatedAt.UTC().Format(time.RFC3339Nano),
			"metadata":            metadata,
		},
	}
	return envelope, true, nil
}

func (a *App) publishPipelineEnvelope(ctx context.Context, envelope pipelineEnvelope) error {
	url := strings.TrimSpace(a.cfg.PipelineRabbitURL)
	exchange := strings.TrimSpace(a.cfg.PipelineEventExchange)
	if url == "" || exchange == "" {
		return nil
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return a.publishPipelineBody(ctx, envelope.MessageType, envelope.EventID, body)
}

func (a *App) publishPipelineBody(ctx context.Context, messageType, eventID string, body []byte) error {
	url := strings.TrimSpace(a.cfg.PipelineRabbitURL)
	exchange := strings.TrimSpace(a.cfg.PipelineEventExchange)
	if url == "" || exchange == "" {
		return nil
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	if err := ch.PublishWithContext(ctx, exchange, messageType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    eventID,
		Timestamp:    time.Now().UTC(),
		Type:         messageType,
		Body:         body,
	}); err != nil {
		return err
	}
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	select {
	case confirmation, ok := <-confirms:
		if !ok || !confirmation.Ack {
			return fmt.Errorf("rabbitmq negatively acknowledged message %s", eventID)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("timed out waiting for rabbitmq confirm for %s", eventID)
	}
}

func (a *App) enqueuePipelineEnvelope(envelope pipelineEnvelope) error {
	if !a.cfg.PipelineEventsEnabled {
		return nil
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	var existing common.PipelineEventOutbox
	lookupErr := a.db.Where("event_id = ?", envelope.EventID).First(&existing).Error
	if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return lookupErr
	}
	if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		if err := a.db.Create(&common.PipelineEventOutbox{
			ID: uuidForEvent(envelope.EventID), EventID: envelope.EventID,
			MessageType: envelope.MessageType, Payload: string(body), Status: "pending",
			NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			// A terminal callback can be delivered more than once. A stable event
			// id turns that race into an idempotent no-op.
			if lookup := a.db.Where("event_id = ?", envelope.EventID).First(&existing).Error; lookup != nil {
				return err
			}
		}
	}
	return a.drainPipelineEventOutbox()
}

func (a *App) drainPipelineEventOutbox() error {
	a.pipelineOutboxMu.Lock()
	defer a.pipelineOutboxMu.Unlock()
	var entries []common.PipelineEventOutbox
	if err := a.db.Where("status IN ? AND next_attempt_at <= ?", []string{"pending", "retrying"}, time.Now()).Order("created_at ASC").Limit(25).Find(&entries).Error; err != nil {
		return err
	}
	var firstErr error
	for _, entry := range entries {
		attempts := entry.Attempts + 1
		err := a.publishPipelineBody(context.Background(), entry.MessageType, entry.EventID, []byte(entry.Payload))
		if err == nil {
			now := time.Now().UTC()
			if updateErr := a.db.Model(&common.PipelineEventOutbox{}).Where("id = ?", entry.ID).Updates(map[string]interface{}{
				"status": "published", "published_at": now, "attempts": attempts, "last_error": "", "updated_at": now,
			}).Error; updateErr != nil && firstErr == nil {
				firstErr = updateErr
			}
			continue
		}
		wait := time.Duration(1<<minOutboxInt(attempts, 6)) * time.Second
		next := time.Now().Add(wait)
		if updateErr := a.db.Model(&common.PipelineEventOutbox{}).Where("id = ?", entry.ID).Updates(map[string]interface{}{
			"status": "retrying", "attempts": attempts, "next_attempt_at": next, "last_error": err.Error(), "updated_at": time.Now(),
		}).Error; updateErr != nil && firstErr == nil {
			firstErr = updateErr
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func uuidForEvent(eventID string) string {
	return "outbox_" + stableShortHex(eventID)
}

func (a *App) publishCrawlSucceededEventAsync(run common.TaskRun) {
	a.publishCrawlTerminalEventAsync(run)
}

func (a *App) publishCrawlTerminalEventAsync(run common.TaskRun) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.publishCrawlTerminalEvent(ctx, run); err != nil {
			a.log.Warn("publish pipeline crawl terminal event failed", zap.Error(err), zap.String("run_id", run.ID), zap.String("status", run.Status))
		}
	}()
}

func normalizeCrawlStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "completed":
		return "success"
	case "cancelled":
		return "canceled"
	case "":
		return "unknown"
	default:
		return strings.ToLower(strings.TrimSpace(status))
	}
}

func defaultCollectionForTask(task common.Task) string {
	base := strings.TrimSpace(task.Name)
	if base == "" {
		base = strings.TrimSpace(task.SourceURL)
	}
	base = strings.Trim(collectionTokenRE.ReplaceAllString(strings.ToLower(base), "_"), "_.-")
	if base == "" {
		base = "feed_" + stableShortHex(task.ID+task.SourceURL)
	}
	if !strings.HasPrefix(base, "documents.") {
		base = "documents." + base
	}
	return base
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func metadataMap(metadata map[string]any, key string) map[string]any {
	if metadata == nil {
		return nil
	}
	value, ok := metadata[key]
	if !ok {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func stableShortHex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}

func timePtrString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
