package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

var collectionTokenRE = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

type moiipEnvelope struct {
	MessageType    string         `json:"message_type"`
	SchemaVersion  string         `json:"schema_version"`
	EventID        string         `json:"event_id"`
	TraceID        string         `json:"trace_id"`
	CausationID    string         `json:"causation_id,omitempty"`
	IdempotencyKey string         `json:"idempotency_key"`
	Producer       string         `json:"producer"`
	RetryCount     int            `json:"retry_count"`
	OccurredAt     string         `json:"occurred_at"`
	CreatedAt      string         `json:"created_at"`
	Correlation    map[string]any `json:"correlation"`
	Payload        map[string]any `json:"payload"`
}

func (a *App) publishCrawlSucceededEvent(ctx context.Context, run common.TaskRun) error {
	if !a.cfg.MOIIPEventsEnabled {
		return nil
	}
	envelope, ok, err := a.buildCrawlSucceededEnvelope(ctx, run)
	if err != nil || !ok {
		return err
	}
	return a.publishMOIIPEnvelope(ctx, envelope)
}

func (a *App) buildCrawlSucceededEnvelope(ctx context.Context, run common.TaskRun) (moiipEnvelope, bool, error) {
	var task common.Task
	if err := a.db.WithContext(ctx).Where("id = ?", run.TaskID).First(&task).Error; err != nil {
		return moiipEnvelope{}, false, err
	}
	taskType := strings.ToLower(strings.TrimSpace(task.Type))
	metadata := parseMetadataJSON(task.MetadataJSON)
	hashslipSlot := metadataMap(metadata, "hashslip_slot")
	if taskType != "rss" && taskType != "api" && len(hashslipSlot) == 0 {
		return moiipEnvelope{}, false, nil
	}
	collection := firstNonEmpty(metadataString(metadata, "hashslip_collection"), metadataString(metadata, "collection"), defaultCollectionForTask(task))
	outputCollection := firstNonEmpty(metadataString(metadata, "analysis_collection"), "analysis."+collection)
	missionID := firstNonEmpty(metadataString(metadata, "mission_id"), "mission_"+stableShortHex(collection))
	traceID := firstNonEmpty(metadataString(metadata, "trace_id"), run.CorrelationID, "trace_"+stableShortHex(run.ID))
	now := time.Now().UTC().Format(time.RFC3339Nano)
	eventID := "evt_" + uuid.NewString()
	envelope := moiipEnvelope{
		MessageType:    "araneae.crawl.succeeded",
		SchemaVersion:  "1.0",
		EventID:        eventID,
		TraceID:        traceID,
		CausationID:    run.ID,
		IdempotencyKey: "araneae.crawl.succeeded:" + run.ID,
		Producer:       "araneae",
		RetryCount:     0,
		OccurredAt:     now,
		CreatedAt:      now,
		Correlation: map[string]any{
			"mission_id":  missionID,
			"task_id":     task.ID,
			"run_id":      run.ID,
			"schedule_id": run.ScheduleID,
			"trace_id":    traceID,
			"collection":  collection,
		},
		Payload: map[string]any{
			"task_id":             task.ID,
			"task_name":           task.Name,
			"task_type":           taskType,
			"run_id":              run.ID,
			"schedule_id":         run.ScheduleID,
			"source_url":          task.SourceURL,
			"hashslip_collection": collection,
			"hashslip_slot":       hashslipSlot,
			"schema_id":           firstNonEmpty(metadataString(metadata, "schema_id"), collection),
			"analysis_collection": outputCollection,
			"artifact_type":       firstNonEmpty(metadataString(metadata, "artifact_type"), "analysis_result"),
			"vesper_job":          metadata["vesper_job"],
			"sink_summary":        run.Output,
			"exit_code":           run.ExitCode,
			"finished_at":         timePtrString(run.FinishedAt),
			"metadata":            metadata,
		},
	}
	return envelope, true, nil
}

func (a *App) publishMOIIPEnvelope(ctx context.Context, envelope moiipEnvelope) error {
	url := strings.TrimSpace(a.cfg.MOIIPRabbitURL)
	exchange := strings.TrimSpace(a.cfg.MOIIPEventExchange)
	if url == "" || exchange == "" {
		return nil
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		return err
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
	return ch.PublishWithContext(ctx, exchange, envelope.MessageType, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    envelope.EventID,
		Timestamp:    time.Now().UTC(),
		Type:         envelope.MessageType,
		Body:         body,
	})
}

func (a *App) publishCrawlSucceededEventAsync(run common.TaskRun) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.publishCrawlSucceededEvent(ctx, run); err != nil {
			a.log.Warn("publish MOIIP crawl succeeded event failed", zap.Error(err), zap.String("run_id", run.ID))
		}
	}()
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
