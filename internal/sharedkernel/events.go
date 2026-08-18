package sharedkernel

import "time"

const (
	TopicTaskCreated = "task.created"
	TopicTaskStatus  = "task.status"
	TopicNotifyUser  = "notify.user"
)

func TopicDispatch(edgeID EdgeID) string {
	return "dispatch." + string(edgeID)
}

type TaskCreated struct {
	TaskID    TaskID    `json:"task_id"`
	ChatID    ChatID    `json:"chat_id"`
	CaseID    CaseID    `json:"case_id"`
	CreatedAt time.Time `json:"created_at"`
}

type DispatchCommand struct {
	TaskID      TaskID  `json:"task_id"`
	EdgeID      EdgeID  `json:"edge_id"`
	InputPrefix string  `json:"input_prefix,omitempty"` // legacy; success path uses JobRef
	JobRef      BlobRef `json:"job_ref"`
}

type TaskStatusEvent struct {
	TaskID    TaskID     `json:"task_id"`
	EdgeID    EdgeID     `json:"edge_id"`
	Status    TaskStatus `json:"status"`
	PromptID  string     `json:"prompt_id,omitempty"`
	Outputs   []BlobRef  `json:"outputs,omitempty"`
	ErrorCode string     `json:"error_code,omitempty"`
	ErrorMsg  string     `json:"error_msg,omitempty"`
	At        time.Time  `json:"at"`
}

type UserNotify struct {
	ChatID   ChatID    `json:"chat_id"`
	TaskID   TaskID    `json:"task_id"`
	Kind     string    `json:"kind"`
	Title    string    `json:"title,omitempty"`
	Outputs  []BlobRef `json:"outputs,omitempty"`
	ErrorMsg string    `json:"error_msg,omitempty"`
}
