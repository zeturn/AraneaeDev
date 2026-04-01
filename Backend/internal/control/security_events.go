package control

import (
	"strings"
	"time"

	"araneae-go/internal/common"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const maxSecurityEventDetailLength = 1024

func normalizeSecuritySeverity(raw string) string {
	severity := strings.ToLower(strings.TrimSpace(raw))
	switch severity {
	case "info", "warning", "critical":
		return severity
	default:
		return "info"
	}
}

func truncateSecurityEventDetail(detail string) string {
	clean := strings.TrimSpace(detail)
	if len(clean) <= maxSecurityEventDetailLength {
		return clean
	}
	return clean[:maxSecurityEventDetailLength]
}

func (a *App) recordSecurityEventWithUser(c *fiber.Ctx, userID, eventType, severity, detail string) {
	if a == nil || a.db == nil {
		return
	}
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return
	}

	event := common.SecurityEvent{
		ID:        uuid.NewString(),
		EventType: eventType,
		Severity:  normalizeSecuritySeverity(severity),
		UserID:    strings.TrimSpace(userID),
		Detail:    truncateSecurityEventDetail(detail),
		CreatedAt: time.Now(),
	}
	if c != nil {
		event.IPAddress = strings.TrimSpace(c.IP())
		event.Method = strings.TrimSpace(c.Method())
		event.Path = strings.TrimSpace(c.Path())
	}

	if err := a.db.Create(&event).Error; err != nil {
		if a.log != nil {
			a.log.Warn("failed to persist security event", zap.Error(err), zap.String("event_type", eventType))
		}
		return
	}

	if a.log != nil && (event.Severity == "critical" || event.Severity == "warning") {
		a.log.Warn("security event", zap.String("event_type", event.EventType), zap.String("severity", event.Severity), zap.String("user_id", event.UserID), zap.String("path", event.Path))
	}
}

func (a *App) recordSecurityEvent(c *fiber.Ctx, eventType, severity, detail string) {
	uid := ""
	if c != nil {
		uid, _ = c.Locals("uid").(string)
	}
	a.recordSecurityEventWithUser(c, uid, eventType, severity, detail)
}
