//go:build integration
// +build integration

package control

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"araneae-go/internal/common"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestControlIntegration_TriggerQueueCallbackFlow(t *testing.T) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	probeConn, err := amqp.Dial(rabbitURL)
	if err != nil {
		t.Skipf("rabbitmq unavailable at %s: %v", rabbitURL, err)
	}
	_ = probeConn.Close()

	app, err := NewApp(testControlConfigWithRabbit(t, rabbitURL))
	if err != nil {
		t.Fatalf("new app failed: %v", err)
	}
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	conn, ch := mustRabbitConsumerChannel(t, rabbitURL, app.cfg.RabbitExchange)
	defer func() {
		_ = ch.Close()
		_ = conn.Close()
	}()

	msgCh := mustBindAndConsumeQueue(t, ch, app.cfg.RabbitExchange, "tasks.default")
	token := loginAndGetToken(t, app)
	_, _, task := createProjectVersionTask(t, app, token, "")

	triggerRec := doJSONRequest(t, app, http.MethodPost, "/api/v1/tasks/"+task.ID+"/trigger", token, nil)
	if triggerRec.Code != http.StatusOK {
		t.Fatalf("trigger task failed: status=%d body=%s", triggerRec.Code, triggerRec.Body.String())
	}

	var run map[string]any
	if err := json.Unmarshal(triggerRec.Body.Bytes(), &run); err != nil {
		t.Fatalf("decode trigger response: %v", err)
	}
	runID, _ := run["id"].(string)
	if runID == "" {
		t.Fatalf("missing run id in trigger response: %s", triggerRec.Body.String())
	}

	select {
	case delivery := <-msgCh:
		var queued queueTaskMessage
		if err := json.Unmarshal(delivery.Body, &queued); err != nil {
			t.Fatalf("decode queued message: %v", err)
		}
		if queued.RunID != runID {
			t.Fatalf("run id mismatch queued=%s trigger=%s", queued.RunID, runID)
		}
		if queued.TaskID != task.ID {
			t.Fatalf("task id mismatch queued=%s task=%s", queued.TaskID, task.ID)
		}
	case <-time.After(8 * time.Second):
		t.Fatal("timed out waiting for rabbit queue message")
	}

	callbackReq := httptest.NewRequest(http.MethodPost, "/api/v1/runs/"+runID+"/callback", bytes.NewReader([]byte(`{"status":"success","output":"integration-ok","exit_code":0}`)))
	callbackReq.Header.Set("Content-Type", "application/json")
	callbackReq.Header.Set("X-Execution-Key", app.cfg.ExecutionAPIKey)
	callbackResp, err := app.http.Test(callbackReq, -1)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	if callbackResp.StatusCode != http.StatusOK {
		b := bytes.NewBuffer(nil)
		_, _ = b.ReadFrom(callbackResp.Body)
		t.Fatalf("callback failed: status=%d body=%s", callbackResp.StatusCode, b.String())
	}
	_ = callbackResp.Body.Close()

	runsRec := doJSONRequest(t, app, http.MethodGet, "/api/v1/tasks/"+task.ID+"/runs", token, nil)
	if runsRec.Code != http.StatusOK {
		t.Fatalf("list runs failed: status=%d body=%s", runsRec.Code, runsRec.Body.String())
	}

	var runs []map[string]any
	if err := json.Unmarshal(runsRec.Body.Bytes(), &runs); err != nil {
		t.Fatalf("decode runs response: %v", err)
	}
	if len(runs) == 0 {
		t.Fatalf("expected at least one run, got 0")
	}
	if runs[0]["status"] != "success" {
		t.Fatalf("expected run status success, got %v", runs[0]["status"])
	}
}

func testControlConfigWithRabbit(t *testing.T, rabbitURL string) common.ControlConfig {
	t.Helper()
	tmp := t.TempDir()
	return common.ControlConfig{
		HTTPAddr:        ":0",
		GRPCAddr:        ":0",
		DBPath:          tmp + "/control.db",
		RabbitURL:       rabbitURL,
		RabbitExchange:  "tasks.direct",
		JWTSecret:       "integration-jwt-secret",
		ArtifactRoot:    tmp + "/artifacts",
		ExecutionAPIKey: "integration-callback-key",
	}
}

func mustRabbitConsumerChannel(t *testing.T, rabbitURL, exchange string) (*amqp.Connection, *amqp.Channel) {
	t.Helper()
	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		t.Fatalf("dial rabbitmq failed: %v", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		t.Fatalf("open rabbitmq channel failed: %v", err)
	}
	if err := ch.ExchangeDeclare(exchange, "direct", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		t.Fatalf("declare exchange failed: %v", err)
	}
	return conn, ch
}

func mustBindAndConsumeQueue(t *testing.T, ch *amqp.Channel, exchange, routingKey string) <-chan amqp.Delivery {
	t.Helper()
	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		t.Fatalf("declare temp queue failed: %v", err)
	}
	if err := ch.QueueBind(q.Name, routingKey, exchange, false, nil); err != nil {
		t.Fatalf("bind queue failed: %v", err)
	}
	msgs, err := ch.Consume(q.Name, "", true, true, false, false, nil)
	if err != nil {
		t.Fatalf("consume queue failed: %v", err)
	}
	return msgs
}
