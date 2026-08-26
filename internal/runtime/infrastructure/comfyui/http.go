package comfyui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/google/uuid"
)

// HTTP talks to a real ComfyUI instance over its REST API.
type HTTP struct {
	BaseURL   string
	Client    *http.Client
	PollEvery time.Duration
	ClientID  string
	MaxWait   time.Duration
}

func NewHTTP(baseURL string) *HTTP {
	return &HTTP{
		BaseURL:   stringsTrimRightSlash(baseURL),
		Client:    &http.Client{Timeout: 30 * time.Second},
		PollEvery: 500 * time.Millisecond,
		ClientID:  uuid.NewString(),
		MaxWait:   10 * time.Minute,
	}
}

func (h *HTTP) Submit(ctx context.Context, graph Graph) (string, error) {
	body, err := json.Marshal(map[string]any{
		"prompt":    graph,
		"client_id": h.clientID(),
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url("/prompt"), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := h.http().Do(req)
	if err != nil {
		return "", fmt.Errorf("comfyui submit: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("comfyui submit: status %d: %s", res.StatusCode, truncate(raw, 256))
	}
	var out struct {
		PromptID string `json:"prompt_id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("comfyui submit decode: %w", err)
	}
	if out.PromptID == "" {
		return "", fmt.Errorf("comfyui submit: empty prompt_id")
	}
	return out.PromptID, nil
}

func (h *HTTP) UploadImage(ctx context.Context, filename, mime string, data []byte) (string, error) {
	if filename == "" {
		filename = "image.png"
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	part, err := w.CreateFormFile("image", filename)
	if err != nil {
		return "", fmt.Errorf("comfyui upload: create part: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("comfyui upload: write part: %w", err)
	}
	if err := w.Close(); err != nil {
		return "", fmt.Errorf("comfyui upload: close multipart: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url("/upload/image"), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := h.http().Do(req)
	if err != nil {
		return "", fmt.Errorf("comfyui upload: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("comfyui upload: status %d: %s", res.StatusCode, truncate(raw, 256))
	}
	var out struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("comfyui upload decode: %w", err)
	}
	if out.Name == "" {
		return "", fmt.Errorf("comfyui upload: empty name")
	}
	return out.Name, nil
}

func (h *HTTP) SystemStats(ctx context.Context) (*SystemStats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url("/system_stats"), nil)
	if err != nil {
		return &SystemStats{Reachable: false, Error: err.Error()}, nil
	}
	res, err := h.http().Do(req)
	if err != nil {
		return &SystemStats{Reachable: false, Error: err.Error()}, nil
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return &SystemStats{
			Reachable: false,
			Error:     fmt.Sprintf("status %d: %s", res.StatusCode, truncate(raw, 256)),
		}, nil
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return &SystemStats{Reachable: false, Error: err.Error()}, nil
	}
	version, _ := body["system"].(map[string]any)["comfyui_version"].(string)
	return &SystemStats{
		Reachable:      true,
		ComfyUIVersion: version,
		Raw:            body,
	}, nil
}

func (h *HTTP) Queue(ctx context.Context) (*QueueView, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url("/queue"), nil)
	if err != nil {
		return &QueueView{Reachable: false, Error: err.Error(), Running: []any{}, Pending: []any{}}, nil
	}
	res, err := h.http().Do(req)
	if err != nil {
		return &QueueView{Reachable: false, Error: err.Error(), Running: []any{}, Pending: []any{}}, nil
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return &QueueView{
			Reachable: false,
			Error:     fmt.Sprintf("status %d: %s", res.StatusCode, truncate(raw, 256)),
			Running:   []any{},
			Pending:   []any{},
		}, nil
	}
	var body struct {
		QueueRunning []any `json:"queue_running"`
		QueuePending []any `json:"queue_pending"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return &QueueView{Reachable: false, Error: err.Error(), Running: []any{}, Pending: []any{}}, nil
	}
	running := body.QueueRunning
	if running == nil {
		running = []any{}
	}
	pending := body.QueuePending
	if pending == nil {
		pending = []any{}
	}
	return &QueueView{Reachable: true, Running: running, Pending: pending}, nil
}

func (h *HTTP) Wait(ctx context.Context, promptID string) (*Result, error) {
	deadline := time.Now().Add(h.maxWait())
	every := h.PollEvery
	if every <= 0 {
		every = 500 * time.Millisecond
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("comfyui wait: timeout for prompt %s", promptID)
		}
		res, done, err := h.pollHistory(ctx, promptID)
		if err != nil {
			return nil, err
		}
		if done {
			return res, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(every):
		}
	}
}

func (h *HTTP) pollHistory(ctx context.Context, promptID string) (*Result, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url("/history/"+url.PathEscape(promptID)), nil)
	if err != nil {
		return nil, false, err
	}
	res, err := h.http().Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("comfyui history: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode == http.StatusNotFound || len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("{}")) {
		return nil, false, nil
	}
	if res.StatusCode >= 300 {
		return nil, false, fmt.Errorf("comfyui history: status %d: %s", res.StatusCode, truncate(raw, 256))
	}
	hist, err := ParseHistory(raw)
	if err != nil {
		return nil, false, err
	}
	var statuses map[string]struct {
		Status struct {
			StatusStr string `json:"status_str"`
			Completed bool   `json:"completed"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &statuses); err != nil {
		return nil, false, fmt.Errorf("comfyui history decode: %w", err)
	}
	entry, ok := statuses[promptID]
	if !ok {
		return nil, false, nil
	}
	if entry.Status.StatusStr == "error" {
		return nil, false, fmt.Errorf("comfyui wait: prompt %s failed", promptID)
	}
	nodeOutputs := HistoryResult{}
	for nodeID, node := range hist {
		out := NodeOutput{}
		for _, img := range node.Images {
			if img.Filename == "" {
				continue
			}
			data, mime, err := h.fetchView(ctx, img.Filename, img.Subfolder, img.Type)
			if err != nil {
				return nil, false, err
			}
			out.Images = append(out.Images, NodeImage{
				OutputFile: OutputFile{Filename: img.Filename, Mime: mime, Data: data},
				Subfolder:  img.Subfolder,
				Type:       img.Type,
			})
		}
		out.Texts = append(out.Texts, node.Texts...)
		if len(out.Images) > 0 || len(out.Texts) > 0 {
			nodeOutputs[nodeID] = out
		}
	}
	if len(nodeOutputs) == 0 && !entry.Status.Completed {
		return nil, false, nil
	}
	if len(nodeOutputs) == 0 {
		return nil, false, fmt.Errorf("comfyui wait: completed without outputs")
	}
	return &Result{PromptID: promptID, Outputs: nodeOutputs}, true, nil
}

func (h *HTTP) fetchView(ctx context.Context, filename, subfolder, typ string) ([]byte, string, error) {
	if typ == "" {
		typ = "output"
	}
	q := url.Values{}
	q.Set("filename", filename)
	q.Set("subfolder", subfolder)
	q.Set("type", typ)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url("/view")+"?"+q.Encode(), nil)
	if err != nil {
		return nil, "", err
	}
	res, err := h.http().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("comfyui view: %w", err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", err
	}
	if res.StatusCode >= 300 {
		return nil, "", fmt.Errorf("comfyui view: status %d: %s", res.StatusCode, truncate(data, 256))
	}
	mime := res.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
		if path.Ext(filename) == ".png" {
			mime = "image/png"
		}
	}
	return data, mime, nil
}

func (h *HTTP) http() *http.Client {
	if h.Client != nil {
		return h.Client
	}
	return http.DefaultClient
}

func (h *HTTP) clientID() string {
	if h.ClientID != "" {
		return h.ClientID
	}
	return uuid.NewString()
}

func (h *HTTP) maxWait() time.Duration {
	if h.MaxWait > 0 {
		return h.MaxWait
	}
	return 10 * time.Minute
}

func (h *HTTP) url(p string) string {
	return h.BaseURL + p
}

func stringsTrimRightSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
