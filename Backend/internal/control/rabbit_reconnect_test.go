package control

import (
	"errors"
	"testing"

	"araneae-go/internal/common"
)

func TestRabbitPublisherRetriesWhenPreviousReconnectLeftNoConnection(t *testing.T) {
	app := &App{cfg: common.ControlConfig{
		RabbitURL:      "amqp://guest:guest@127.0.0.1:1/",
		RabbitExchange: "tasks.direct",
	}}

	_, err := app.rabbitPublisher()
	if err == nil {
		t.Fatal("expected RabbitMQ dial failure")
	}
	if !errors.Is(err, errQueueUnavailable) {
		t.Fatalf("expected queue-unavailable wrapper for the failed reconnect: %v", err)
	}
	if err == errQueueUnavailable {
		t.Fatalf("expected a reconnect attempt, got permanent queue-unavailable error: %v", err)
	}
}
