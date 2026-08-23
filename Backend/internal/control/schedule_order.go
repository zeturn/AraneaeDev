package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"araneae-go/internal/common"
)

const canonicalScheduleOrderVersion = "1.0"

// scheduleOrder is the only schedule-chain representation accepted by the
// API, persisted in the database, and consumed by the scheduler.
type scheduleOrder struct {
	Version string              `json:"version"`
	Name    string              `json:"name"`
	Steps   []scheduleOrderStep `json:"steps"`
}

type scheduleOrderStep struct {
	TaskID    string `json:"task_id"`
	Name      string `json:"name,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	NodeQueue string `json:"node_queue,omitempty"`
	Trigger   string `json:"trigger"`
	CronExpr  string `json:"cron_expr,omitempty"`
	RunAt     string `json:"run_at,omitempty"`
	Previous  string `json:"previous,omitempty"`
}

// parseScheduleOrder deliberately accepts only the versioned canonical
// object. Legacy schedule[] payloads are not protocol-compatible input.
func parseScheduleOrder(raw any) (scheduleOrder, bool, error) {
	if raw == nil {
		return scheduleOrder{}, false, nil
	}

	var data []byte
	switch value := raw.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return scheduleOrder{}, false, nil
		}
		data = []byte(value)
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return scheduleOrder{}, false, err
		}
		data = encoded
	}

	var order scheduleOrder
	if err := json.Unmarshal(data, &order); err != nil {
		return scheduleOrder{}, false, fmt.Errorf("order must be valid JSON: %w", err)
	}
	if order.Version != canonicalScheduleOrderVersion {
		return scheduleOrder{}, false, fmt.Errorf("order.version must be %s", canonicalScheduleOrderVersion)
	}
	if order.Steps == nil {
		return scheduleOrder{}, false, errors.New("order.steps must be an array")
	}
	return order, true, nil
}

// validateAndNormalizeScheduleOrder validates task references and normalizes
// all derived step fields before a schedule is written.
func (a *App) validateAndNormalizeScheduleOrder(order scheduleOrder) (scheduleOrder, error) {
	if order.Version != canonicalScheduleOrderVersion {
		return scheduleOrder{}, fmt.Errorf("order.version must be %s", canonicalScheduleOrderVersion)
	}
	if len(order.Steps) == 0 {
		return scheduleOrder{}, errors.New("order.steps must contain at least one task")
	}

	normalized := make([]scheduleOrderStep, 0, len(order.Steps))
	for index, rawStep := range order.Steps {
		step := rawStep
		step.TaskID = strings.TrimSpace(step.TaskID)
		if step.TaskID == "" {
			return scheduleOrder{}, errors.New("each schedule step must reference an existing task_id")
		}

		var task common.Task
		if err := a.db.Where("id = ?", step.TaskID).First(&task).Error; err != nil {
			return scheduleOrder{}, errors.New("schedule step references a task that does not exist")
		}

		step.Name = strings.TrimSpace(step.Name)
		if step.Name == "" {
			step.Name = task.Name
		}
		step.ProjectID = strings.TrimSpace(step.ProjectID)
		if step.ProjectID == "" {
			step.ProjectID = strings.TrimSpace(task.ProjectID)
		}
		step.NodeQueue = sanitizeNodeQueue(strings.TrimSpace(step.NodeQueue))
		if step.NodeQueue == "" {
			step.NodeQueue = sanitizeNodeQueue(strings.TrimSpace(string(task.NodeQueue)))
		}
		if step.NodeQueue == "" {
			step.NodeQueue = "default"
		}

		step.Trigger = strings.ToLower(strings.TrimSpace(step.Trigger))
		step.CronExpr = strings.TrimSpace(step.CronExpr)
		step.RunAt = strings.TrimSpace(step.RunAt)
		if index == 0 {
			if step.Trigger == "" {
				switch {
				case step.CronExpr != "":
					step.Trigger = "crons"
				case step.RunAt != "":
					step.Trigger = "datetime"
				default:
					step.Trigger = "api"
				}
			}
			if step.Trigger != "crons" && step.Trigger != "api" && step.Trigger != "datetime" {
				return scheduleOrder{}, errors.New("first schedule step trigger must be crons, api or datetime")
			}
			switch step.Trigger {
			case "crons":
				if step.CronExpr == "" {
					return scheduleOrder{}, errors.New("cron expression is required when the first step trigger is crons")
				}
				step.RunAt = ""
			case "datetime":
				runAt, err := parseScheduleRunAt(step.RunAt)
				if err != nil {
					return scheduleOrder{}, err
				}
				step.RunAt = runAt.Format(time.RFC3339)
				step.CronExpr = ""
			case "api":
				step.CronExpr = ""
				step.RunAt = ""
			}
			step.Previous = ""
		} else {
			if step.Trigger != "" && step.Trigger != "previous" {
				return scheduleOrder{}, errors.New("only the first schedule step can use crons, api or datetime trigger")
			}
			step.Trigger = "previous"
			step.CronExpr = ""
			step.RunAt = ""
			previous := normalized[index-1].Name
			if previous == "" {
				previous = normalized[index-1].TaskID
			}
			step.Previous = previous
		}

		normalized = append(normalized, step)
	}

	order.Version = canonicalScheduleOrderVersion
	order.Name = strings.TrimSpace(order.Name)
	if order.Name == "" {
		order.Name = normalized[0].Name
	}
	order.Steps = normalized
	return order, nil
}

func marshalScheduleOrder(order scheduleOrder) (string, error) {
	order.Version = canonicalScheduleOrderVersion
	order.Name = strings.TrimSpace(order.Name)
	if order.Steps == nil {
		order.Steps = []scheduleOrderStep{}
	}
	b, err := json.Marshal(order)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
