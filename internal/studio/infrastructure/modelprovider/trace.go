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
	ResponseBody      json.RawMessage
	StatusCode        int
	ProviderRequestID string
	InputTokens       int
	OutputTokens      int
	UsageReported     bool
	Error             string
}

// TraceSink is invoked synchronously so a successful model request cannot
// silently lose its source-of-truth diagnostic record.
type TraceSink func(context.Context, TraceEvent) error

type traceAttempt struct {
	sink        TraceSink
	startedAt   time.Time
	credential  string
	requestBody json.RawMessage
	statusCode  int
	requestID   string
	chunks      []traceChunk
}

type traceChunk struct {
	At        time.Time       `json:"at"`
	ElapsedMS int64           `json:"elapsed_ms"`
	Data      json.RawMessage `json:"data"`
}

func beginTrace(ctx context.Context, sink TraceSink, body []byte, credential string) (*traceAttempt, error) {
	if sink == nil {
		return nil, nil
	}
	attempt := &traceAttempt{sink: sink, startedAt: time.Now().UTC(), credential: credential, requestBody: redactTraceJSON(body, credential)}
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

func (a *traceAttempt) recordChunk(data []byte) {
	if a == nil {
		return
	}
	now := time.Now().UTC()
	a.chunks = append(a.chunks, traceChunk{At: now, ElapsedMS: now.Sub(a.startedAt).Milliseconds(), Data: redactTraceJSON(data, a.credential)})
}

func (a *traceAttempt) finish(ctx context.Context, inputTokens, outputTokens int, usageReported bool, responseBody []byte) error {
	if a == nil || a.sink == nil {
		return nil
	}
	now := time.Now().UTC()
	if len(a.chunks) > 0 {
		responseBody, _ = json.Marshal(map[string]any{"stream": a.chunks})
	} else if len(responseBody) > 0 {
		responseBody = redactTraceJSON(responseBody, a.credential)
	}
	return a.sink(ctx, TraceEvent{Phase: TraceRequestFinished, At: now, Elapsed: now.Sub(a.startedAt), StatusCode: a.statusCode, ProviderRequestID: a.requestID, InputTokens: inputTokens, OutputTokens: outputTokens, UsageReported: usageReported, ResponseBody: append(json.RawMessage(nil), responseBody...)})
}

func (a *traceAttempt) fail(ctx context.Context, err error) error {
	return a.failWithBody(ctx, err, nil)
}

func (a *traceAttempt) failWithBody(ctx context.Context, err error, body []byte) error {
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
	if a.credential != "" {
		message = strings.ReplaceAll(message, a.credential, "[REDACTED]")
	}
	var responseBody json.RawMessage
	if len(body) > 0 {
		responseBody = redactTraceJSON(body, a.credential)
	} else if len(a.chunks) > 0 {
		responseBody, _ = json.Marshal(map[string]any{"stream": a.chunks})
	}
	return a.sink(ctx, TraceEvent{Phase: TraceRequestFailed, At: now, Elapsed: now.Sub(a.startedAt), StatusCode: a.statusCode, ProviderRequestID: a.requestID, ResponseBody: responseBody, Error: message})
}

func redactTraceJSON(body []byte, credential string) json.RawMessage {
	if credential != "" {
		body = []byte(strings.ReplaceAll(string(body), credential, "[REDACTED]"))
	}
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
		if typed["type"] == "base64" {
			if mediaType, ok := typed["media_type"].(string); ok && strings.HasPrefix(mediaType, "image/") {
				typed["data"] = "[IMAGE DATA]"
			}
		}
		for key, child := range typed {
			if key == "image_url" || key == "url" {
				if url, ok := child.(string); ok && strings.HasPrefix(url, "data:image/") {
					typed[key] = "[IMAGE DATA]"
					continue
				}
			}
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
	return strings.Contains(key, "password") || strings.Contains(key, "secret") || strings.Contains(key, "authorization") || strings.Contains(key, "credential") || key == "token" || strings.HasSuffix(key, "_token") || key == "key" || strings.HasSuffix(key, "_key")
}
