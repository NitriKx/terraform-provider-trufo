package trufo

import "context"

// Task is the status of an async operation (the only GET endpoint:
// GET /tasks/{task_id}).
type Task struct {
	TaskID         string         `json:"task_id"`
	TaskType       string         `json:"task_type"`
	Status         string         `json:"status"` // queued, running, succeeded, failed, expired
	StartbyTS      string         `json:"startby_ts,omitempty"`
	TimeoutSeconds int            `json:"timeout_seconds,omitempty"`
	CreateTS       string         `json:"create_ts,omitempty"`
	UpdateTS       string         `json:"update_ts,omitempty"`
	StartTS        string         `json:"start_ts,omitempty"`
	FinishTS       string         `json:"finish_ts,omitempty"`
	DurationMS     int64          `json:"duration_ms,omitempty"`
	Progress       map[string]any `json:"progress,omitempty"`
	FileSizeBytes  int64          `json:"file_size_bytes,omitempty"`
	FileMIMEType   string         `json:"file_mime_type,omitempty"`
	Result         map[string]any `json:"result,omitempty"`
	Error          map[string]any `json:"error,omitempty"`
}

// GetTask calls GET /tasks/{task_id}.
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	var out Task
	if err := c.get(ctx, TasksPath+taskID, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
