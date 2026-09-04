package pull

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/edge"
	"github.com/mr9esx/comfyui_tgbot/internal/platform/queue"
	"github.com/mr9esx/comfyui_tgbot/internal/runtime/infrastructure/actuator"
	"github.com/mr9esx/comfyui_tgbot/internal/sharedkernel"
)

// Job is a claim response from the control-plane Agent API.
type Job struct {
	TaskID     sharedkernel.TaskID  `json:"task_id"`
	EdgeID     sharedkernel.EdgeID  `json:"edge_id"`
	JobRef     sharedkernel.BlobRef `json:"job_ref"`
	LeaseUntil time.Time            `json:"lease_until"`
}

// Client talks to control-plane /agent/v1.
type Client struct {
	BaseURL string
	Token   string
	EdgeID  string
	HTTP    *http.Client
}

func NewClient(baseURL, token, edgeID string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   token,
		EdgeID:  edgeID,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) Claim(ctx context.Context, wait time.Duration) (*Job, error) {
	q := url.Values{}
	q.Set("edge_id", c.EdgeID)
	if wait > 0 {
		q.Set("wait", wait.String())
	} else {
		q.Set("wait", "0s")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/agent/v1/jobs/claim?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if res.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("pull: unauthorized")
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("pull: claim status %d: %s", res.StatusCode, body)
	}
	var job Job
	if err := json.NewDecoder(res.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("pull: decode claim: %w", err)
	}
	if job.TaskID == "" || job.JobRef.Key == "" {
		return nil, nil
	}
	return &job, nil
}

func (c *Client) Heartbeat(ctx context.Context, taskID sharedkernel.TaskID) error {
	body, _ := json.Marshal(map[string]string{"edge_id": c.EdgeID})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/agent/v1/jobs/"+url.PathEscape(string(taskID))+"/heartbeat", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("pull: unauthorized")
	}
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("pull: heartbeat status %d: %s", res.StatusCode, raw)
	}
	return nil
}

func (c *Client) ReportPresence(
	ctx context.Context,
	comfyRunning bool,
	startedAt time.Time,
	comfyVersion string,
	hw *edge.Hardware,
	m *edge.Metrics,
) (refresh bool, consuming bool, err error) {
	payload := map[string]any{
		"edge_id":       c.EdgeID,
		"comfy_running": comfyRunning,
	}
	if !startedAt.IsZero() {
		payload["started_at"] = startedAt
	}
	if comfyVersion != "" {
		payload["comfy_version"] = comfyVersion
	}
	if hw != nil {
		payload["hardware"] = hw
	}
	if m != nil {
		payload["metrics"] = m
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return false, false, fmt.Errorf("pull: encode presence: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/agent/v1/presence", bytes.NewReader(body))
	if err != nil {
		return false, false, err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return false, false, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return false, false, fmt.Errorf("pull: unauthorized")
	}
	if res.StatusCode == http.StatusNoContent {
		return false, false, nil
	}
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return false, false, fmt.Errorf("pull: presence status %d: %s", res.StatusCode, raw)
	}
	var out struct {
		RefreshHardware bool `json:"refresh_hardware"`
		Consuming       bool `json:"consuming"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out); err != nil && !errors.Is(err, io.EOF) {
		return false, false, fmt.Errorf("pull: decode presence: %w", err)
	}
	return out.RefreshHardware, out.Consuming, nil
}

func (c *Client) ReportStatus(ctx context.Context, ev sharedkernel.TaskStatusEvent) error {
	payload := map[string]any{
		"edge_id":    string(ev.EdgeID),
		"status":     string(ev.Status),
		"prompt_id":  ev.PromptID,
		"outputs":    ev.Outputs,
		"error_code": ev.ErrorCode,
		"error_msg":  ev.ErrorMsg,
	}
	if payload["edge_id"] == "" {
		payload["edge_id"] = c.EdgeID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/agent/v1/jobs/"+url.PathEscape(string(ev.TaskID))+"/status", bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.auth(req)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("pull: unauthorized")
	}
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return fmt.Errorf("pull: status report %d: %s", res.StatusCode, raw)
	}
	return nil
}

func (c *Client) auth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.Token)
}

// StatusPublisher adapts Client to queue.Publisher for actuator.Worker.
type StatusPublisher struct {
	Client *Client
}

func NewStatusPublisher(c *Client) *StatusPublisher {
	return &StatusPublisher{Client: c}
}

func (p *StatusPublisher) Publish(ctx context.Context, msg queue.Message) error {
	var ev sharedkernel.TaskStatusEvent
	if err := json.Unmarshal(msg.Payload, &ev); err != nil {
		return fmt.Errorf("pull: status payload: %w", err)
	}
	return p.Client.ReportStatus(ctx, ev)
}

// Loop repeatedly claims jobs and runs the worker.
type Loop struct {
	Client *Client
	Worker *actuator.Worker
	Wait   time.Duration
	Once   bool // for tests: stop after one claim attempt (empty or executed)
}

const DefaultClaimWait = 5 * time.Second

func (l *Loop) Run(ctx context.Context) error {
	if l.Client == nil || l.Worker == nil {
		return fmt.Errorf("pull: loop not configured")
	}
	wait := l.Wait
	if wait <= 0 && !l.Once {
		wait = DefaultClaimWait
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		job, err := l.Client.Claim(ctx, wait)
		if err != nil {
			if l.Once {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Second):
				continue
			}
		}
		if job == nil {
			if l.Once {
				return nil
			}
			continue
		}
		cmd := sharedkernel.DispatchCommand{
			TaskID: job.TaskID,
			EdgeID: job.EdgeID,
			JobRef: job.JobRef,
		}
		if cmd.EdgeID == "" {
			cmd.EdgeID = sharedkernel.EdgeID(l.Client.EdgeID)
		}
		hbCtx, cancel := context.WithCancel(ctx)
		go l.heartbeat(hbCtx, job.TaskID)
		err = l.Worker.HandleDispatch(ctx, cmd)
		cancel()
		if err != nil && l.Once {
			return err
		}
		if l.Once {
			return nil
		}
	}
}

func (l *Loop) heartbeat(ctx context.Context, taskID sharedkernel.TaskID) {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = l.Client.Heartbeat(ctx, taskID)
		}
	}
}
