package sharedkernel

import "time"

const (
	TopicTaskCreated = "task.created"
	TopicTaskStatus  = "task.status"
	TopicNotifyUser  = "notify.user"
)

func TopicDispatch(instanceID InstanceID) string {
	return "dispatch." + string(instanceID)
}

type TaskCreated struct {
	TaskID    TaskID    `json:"task_id"`
	ChatID    ChatID    `json:"chat_id"`
	CaseID    CaseID    `json:"case_id"`
	CreatedAt time.Time `json:"created_at"`
}

type DispatchCommand struct {
	TaskID      TaskID     `json:"task_id"`
	InstanceID  InstanceID `json:"instance_id"`
	InputPrefix string     `json:"input_prefix"`
}

type TaskStatusEvent struct {
	TaskID     TaskID     `json:"task_id"`
	InstanceID InstanceID `json:"instance_id"`
	Status     TaskStatus `json:"status"`
	PromptID   string     `json:"prompt_id,omitempty"`
	Outputs    []BlobRef  `json:"outputs,omitempty"`
	ErrorCode  string     `json:"error_code,omitempty"`
	ErrorMsg   string     `json:"error_msg,omitempty"`
	At         time.Time  `json:"at"`
}

type UserNotify struct {
	ChatID   ChatID    `json:"chat_id"`
	TaskID   TaskID    `json:"task_id"`
	Kind     string    `json:"kind"`
	Title    string    `json:"title,omitempty"`
	Outputs  []BlobRef `json:"outputs,omitempty"`
	ErrorMsg string    `json:"error_msg,omitempty"`
}
