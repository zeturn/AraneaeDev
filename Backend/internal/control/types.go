package control

import "time"

type queueTaskMessage struct {
	RunID        string `json:"run_id"`
	TaskID       string `json:"task_id"`
	ProjectID    string `json:"project_id"`
	VersionID    string `json:"version_id"`
	EntryCommand string `json:"entry_command"`
	NodeQueue    string `json:"node_queue"`
}

type runCallbackPayload struct {
	Status     string     `json:"status"`
	Output     string     `json:"output"`
	ExitCode   int        `json:"exit_code"`
	StartedAt  *time.Time `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}
