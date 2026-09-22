package controlplane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// successCode 是控制面的成功码，与 internal/apierr.SuccessCode 保持一致。
const successCode = 2000000

// envelope 是控制面所有 JSON 接口的统一响应封装。
type envelope struct {
	Message     string          `json:"message"`
	Code        int             `json:"code"`
	Data        json.RawMessage `json:"data"`
	ErrorDetail string          `json:"error_detail"`
}

// decodeEnvelope 读取响应体并取出 data。
//
// 返回 nil 表示这次调用没有数据（例如没有领到任务）。
// 业务错误码不是成功码时，用服务端给出的可读文案组装 error。
func decodeEnvelope(r io.Reader) (json.RawMessage, error) {
	raw, err := io.ReadAll(io.LimitReader(r, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("pull: read body: %w", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("pull: decode envelope: %w", err)
	}
	if env.Code != successCode {
		reason := env.ErrorDetail
		if reason == "" {
			reason = env.Message
		}
		return nil, fmt.Errorf("pull: server code %d: %s", env.Code, reason)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, nil
	}
	return env.Data, nil
}
