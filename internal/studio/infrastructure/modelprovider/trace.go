package modelprovider

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// TracePhase describes an observable boundary of one provider request. These
// values intentionally describe transport facts, not UI events, so callers can
// persist them as a diagnostic trace without reconstructing timing later.
type TracePhase string

const (
	TraceRequestStarted  TracePhase = "request_started"
	TraceFirstToken      TracePhase = "first_token"
	TraceRequestFinished TracePhase = "request_finished"
	TraceRequestFailed   TracePhase = "request_failed"
)

// TraceEvent contains a credential-safe snapshot of one provider attempt.
// RequestBody is the actual protocol body after local normalization, with
// secret-shaped fields redacted before it leaves the provider boundary.
type TraceEvent struct {
	Phase             TracePhase
	At                time.Time
	Elapsed           time.Duration
	RequestBody       json.RawMessage
	StatusCode        int
	ProviderRequestID string
	InputTokens       int
	OutputTokens      int
	Error             string
}

// TraceSink is invoked synchronously so a successful model request cannot
// silently lose its source-of-truth diagnostic record.
type TraceSink func(context.Context, TraceEvent) error

type traceAttempt struct {
	sink        TraceSink
	startedAt   time.Time
	requestBody json.RawMessage
	statusCode  int
	requestID   string
}

func beginTrace(ctx context.Context, sink TraceSink, body []byte) (*traceAttempt, error) {
	if sink == nil {
		return nil, nil
	}
	attempt := &traceAttempt{sink: sink, startedAt: time.Now().UTC(), requestBody: redactTraceJSON(body)}
	if err := sink(ctx, TraceEvent{Phase: TraceRequestStarted, At: attempt.startedAt, RequestBody: append(json.RawMessage(nil), attempt.requestBody...)}); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (a *traceAttempt) receivedResponse(statusCode int, requestID string) {
	if a == nil {
		return
	}
	a.statusCode = statusCode
	a.requestID = requestID
}

func (a *traceAttempt) firstToken(ctx context.Context) error {
	if a == nil || a.sink == nil {
		return nil
	}
	now := time.Now().UTC()
	return a.sink(ctx, TraceEvent{Phase: TraceFirstToken, At: now, Elapsed: now.Sub(a.startedAt), StatusCode: a.statusCode, ProviderRequestID: a.requestID})
}

func (a *traceAttempt) finish(ctx context.Context, inputTokens, outputTokens int) error {
	if a == nil || a.sink == nil {
		return nil
	}
	now := time.Now().UTC()
	return a.sink(ctx, TraceEvent{Phase: TraceRequestFinished, At: now, Elapsed: now.Sub(a.startedAt), StatusCode: a.statusCode, ProviderRequestID: a.requestID, InputTokens: inputTokens, OutputTokens: outputTokens})
}

func (a *traceAttempt) fail(ctx context.Context, err error) error {
	if a == nil || a.sink == nil {
		return nil
	}
	now := time.Now().UTC()
	message := "provider request failed"
	if err != nil && strings.TrimSpace(err.Error()) != "" {
		message = err.Error()
	}
	if len(message) > 1024 {
		message = message[:1024]
	}
	return a.sink(ctx, TraceEvent{Phase: TraceRequestFailed, At: now, Elapsed: now.Sub(a.startedAt), StatusCode: a.statusCode, ProviderRequestID: a.requestID, Error: message})
}

func redactTraceJSON(body []byte) json.RawMessage {
	var value any
	if json.Unmarshal(body, &value) != nil {
		return json.RawMessage(`{"redacted":true,"reason":"invalid_json"}`)
	}
	redactTraceValue(value)
	redacted, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"redacted":true,"reason":"encode_failed"}`)
	}
	return redacted
}

func redactTraceValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if isSensitiveTraceKey(key) {
				typed[key] = "[REDACTED]"
				continue
			}
			redactTraceValue(child)
		}
	case []any:
		for _, child := range typed {
			redactTraceValue(child)
		}
	}
}

func isSensitiveTraceKey(key string) bool {
	key = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "-", "_"), " ", "_"))
	return strings.Contains(key, "password") || strings.Contains(key, "secret") || strings.Contains(key, "api_key") || strings.Contains(key, "authorization") || strings.Contains(key, "access_token") || strings.Contains(key, "refresh_token")
}
