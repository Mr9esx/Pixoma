package sharedkernel

type (
	CaseID     string
	TaskID     string
	SessionID  string
	ChatID     int64
	InstanceID string
)

type BlobRef struct {
	Bucket string `json:"bucket,omitempty"`
	Key    string `json:"key"`
	MIME   string `json:"mime,omitempty"`
	Size   int64  `json:"size,omitempty"`
}

type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskQueued    TaskStatus = "queued"
	TaskRunning   TaskStatus = "running"
	TaskSucceeded TaskStatus = "succeeded"
	TaskFailed    TaskStatus = "failed"
	TaskCancelled TaskStatus = "cancelled"
)
