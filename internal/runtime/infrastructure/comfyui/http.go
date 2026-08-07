package comfyui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/google/uuid"
)

// HTTP talks to a real ComfyUI instance over its REST API.
type HTTP struct {
	BaseURL    string
	Client     *http.Client
	PollEvery  time.Duration
	ClientID   string
	MaxWait    time.Duration
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
	var hist map[string]struct {
		Outputs map[string]struct {
			Images []struct {
				Filename  string `json:"filename"`
				Subfolder string `json:"subfolder"`
				Type      string `json:"type"`
			} `json:"images"`
		} `json:"outputs"`
		Status struct {
			StatusStr string `json:"status_str"`
			Completed bool   `json:"completed"`
		} `json:"status"`
	}
	if err := json.Unmarshal(raw, &hist); err != nil {
		return nil, false, fmt.Errorf("comfyui history decode: %w", err)
	}
	entry, ok := hist[promptID]
	if !ok {
		return nil, false, nil
	}
	if entry.Status.StatusStr == "error" {
		return nil, false, fmt.Errorf("comfyui wait: prompt %s failed", promptID)
	}
	var outs []OutputFile
	for _, node := range entry.Outputs {
		for _, img := range node.Images {
			data, mime, err := h.fetchView(ctx, img.Filename, img.Subfolder, img.Type)
			if err != nil {
				return nil, false, err
			}
			outs = append(outs, OutputFile{Filename: img.Filename, Mime: mime, Data: data})
		}
	}
	if len(outs) == 0 && !entry.Status.Completed {
		return nil, false, nil
	}
	if len(outs) == 0 {
		return nil, false, fmt.Errorf("comfyui wait: completed without image outputs")
	}
	return &Result{PromptID: promptID, Outputs: outs}, true, nil
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
