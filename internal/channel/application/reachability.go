package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ReachabilityKind string

const (
	ReachabilityOK      ReachabilityKind = "ok"
	ReachabilityNetwork ReachabilityKind = "network"
	ReachabilityAuth    ReachabilityKind = "auth"
	ReachabilityOther   ReachabilityKind = "other"
)

type ReachabilityResult struct {
	OK      bool             `json:"ok"`
	Kind    ReachabilityKind `json:"kind"`
	Message string           `json:"message"`
}

const reachabilityTimeout = 3 * time.Second

// telegramGetMeEnvelope mirrors the Bot API getMe response envelope.
// result is kept raw so each platform's identity payload is preserved as-is.
type telegramGetMeEnvelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
}

// fetchTelegramBotInfo calls getMe with the given token and returns the raw
// User object (JSON) exactly as Telegram reported it. Identity fields differ
// per platform/token, so the payload is intentionally not unmarshalled here.
func fetchTelegramBotInfo(ctx context.Context, token string) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, reachabilityTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/bot"+token+"/getMe", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram api status %d", resp.StatusCode)
	}
	var env telegramGetMeEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("telegram getMe decode: %w", err)
	}
	if !env.OK {
		return nil, fmt.Errorf("telegram getMe: ok=false")
	}
	if len(env.Result) == 0 {
		return nil, fmt.Errorf("telegram getMe: empty result")
	}
	return env.Result, nil
}

// telegramBotDisplayName derives a human-friendly name from the getMe
// identity payload (first_name preferred, else the @username).
func telegramBotDisplayName(raw json.RawMessage) string {
	var u struct {
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	}
	if len(raw) == 0 {
		return ""
	}
	if err := json.Unmarshal(raw, &u); err != nil {
		return ""
	}
	if strings.TrimSpace(u.FirstName) != "" {
		return u.FirstName
	}
	return strings.TrimSpace(u.Username)
}

func classifyGetMeError(statusCode int, err error) ReachabilityResult {
	switch {
	case err == nil && statusCode == http.StatusOK:
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}
	case err == nil && (statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden):
		return ReachabilityResult{OK: false, Kind: ReachabilityAuth, Message: "unauthorized"}
	case err != nil && isNetworkError(err):
		return ReachabilityResult{OK: false, Kind: ReachabilityNetwork, Message: err.Error()}
	case err != nil:
		return ReachabilityResult{OK: false, Kind: ReachabilityOther, Message: err.Error()}
	default:
		return ReachabilityResult{OK: false, Kind: ReachabilityOther, Message: fmt.Sprintf("telegram api status %d", statusCode)}
	}
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		return networkMarker(ue.Error()) || networkMarker(ue.Err.Error())
	}
	return networkMarker(err.Error())
}

var networkMarkers = []string{
	"context deadline exceeded",
	"connection refused",
	"connection reset",
	"eof",
	"tls handshake timeout",
	"i/o timeout",
	"no such host",
	"network is unreachable",
	"proxyconnect tcp",
	"client.timeout",
}

func networkMarker(msg string) bool {
	m := strings.ToLower(msg)
	for _, marker := range networkMarkers {
		if strings.Contains(m, marker) {
			return true
		}
	}
	return false
}

func checkTelegramReachability(ctx context.Context, token string) (ReachabilityResult, error) {
	ctx, cancel := context.WithTimeout(ctx, reachabilityTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/bot"+token+"/getMe", nil)
	if err != nil {
		return ReachabilityResult{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return classifyGetMeError(0, err), nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return classifyGetMeError(resp.StatusCode, nil), nil
}
