package control

import (
	"strings"

	"araneae-go/internal/common"
)

type scheduleExecutionStep struct {
	TaskID       string
	ProjectID    string
	VersionID    string
	EntryCommand string
	NodeQueue    string
	Type         string
	SourceURL    string
}

func (a *App) resolveScheduleExecutionSteps(schedule common.Schedule) ([]scheduleExecutionStep, error) {
	defaultStep := scheduleExecutionStep{
		TaskID:       schedule.ID,
		ProjectID:    strings.TrimSpace(schedule.ProjectID),
		VersionID:    strings.TrimSpace(schedule.VersionID),
		EntryCommand: strings.TrimSpace(schedule.EntryCommand),
		NodeQueue:    strings.TrimSpace(schedule.NodeQueue),
		Type:         "code",
	}
	if defaultStep.NodeQueue == "" {
		defaultStep.NodeQueue = "default"
	}

	parsed, ok := parseLegacyOrder(schedule.OrderJSON)
	if !ok || len(parsed.Schedule) == 0 {
		return []scheduleExecutionStep{defaultStep}, nil
	}

	steps := make([]scheduleExecutionStep, 0, len(parsed.Schedule))
	for _, item := range parsed.Schedule {
		step := defaultStep
		if taskID := strings.TrimSpace(item.TaskID); taskID != "" {
			step.TaskID = taskID
			var task common.Task
			if err := a.db.Where("id = ?", taskID).First(&task).Error; err == nil {
				step.ProjectID = task.ProjectID
				step.VersionID = task.VersionID
				step.EntryCommand = task.EntryCommand
				step.NodeQueue = task.NodeQueue
				step.Type = task.Type
				step.SourceURL = task.SourceURL
			}
		}
		if projectID := strings.TrimSpace(item.ProjectID); projectID != "" {
			step.ProjectID = projectID
		}
		if len(item.Node) > 0 {
			if queue := strings.TrimSpace(item.Node[0]); queue != "" {
				step.NodeQueue = queue
			}
		}
		if step.TaskID == "" {
			step.TaskID = schedule.ID
		}
		if step.Type == "" {
			step.Type = "code"
		}
		if step.NodeQueue == "" {
			step.NodeQueue = "default"
		}
		if step.Type == "rss" || step.Type == "api" {
			if step.SourceURL == "" {
				continue
			}
		} else if step.ProjectID == "" || step.VersionID == "" || step.EntryCommand == "" {
			continue
		}
		steps = append(steps, step)
	}

	if len(steps) == 0 {
		return []scheduleExecutionStep{defaultStep}, nil
	}
	return steps, nil
}

func (a *App) triggerNextScheduleChainRun(prevRun common.TaskRun) error {
	if prevRun.ScheduleID == "" || prevRun.ChainID == "" {
		return nil
	}
	if prevRun.ChainTotal <= 1 || prevRun.ChainIndex+1 >= prevRun.ChainTotal {
		return nil
	}

	var schedule common.Schedule
	if err := a.db.Where("id = ?", prevRun.ScheduleID).First(&schedule).Error; err != nil {
		return err
	}
	steps, err := a.resolveScheduleExecutionSteps(schedule)
	if err != nil {
		return err
	}
	nextIndex := prevRun.ChainIndex + 1
	if nextIndex >= len(steps) {
		return nil
	}
	next := steps[nextIndex]
	metadata := map[string]any{}
	if next.TaskID != "" {
		var task common.Task
		if err := a.db.Where("id = ?", next.TaskID).First(&task).Error; err == nil {
			metadata = parseMetadataJSON(task.MetadataJSON)
		}
	}

	_, err = a.publishRun(
		next.TaskID,
		schedule.ID,
		"schedule_chain",
		next.ProjectID,
		next.VersionID,
		next.EntryCommand,
		next.NodeQueue,
		next.Type,
		next.SourceURL,
		metadata,
		&chainRunMeta{
			ChainID:    prevRun.ChainID,
			ChainIndex: nextIndex,
			ChainTotal: prevRun.ChainTotal,
		},
	)
	return err
}
