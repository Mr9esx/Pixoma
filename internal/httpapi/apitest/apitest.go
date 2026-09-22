// Package apitest 提供测试用的响应解包辅助。
//
// 所有 JSON 接口都返回 {message, code, data} 封装，测试要断言业务数据时
// 先取出 data。直接把整个 body 解到业务结构体会得到「cannot unmarshal object」
// 这类错误，所以统一走这里的辅助函数。
package apitest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
)

// DataBytes 返回封装里 data 字段的原始 JSON。
//
// 空 body 返回 null；不是封装（二进制预览、SSE 等）时原样返回，
// 这样同一个辅助函数也能用在非 JSON 接口上。
func DataBytes(rec *httptest.ResponseRecorder) []byte {
	return dataOf(rec.Body.Bytes())
}

// DataReader 返回 data 字段的读取器，配合 json.NewDecoder 使用。
func DataReader(res *http.Response) io.Reader {
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		panic("apitest: 读取响应体失败: " + err.Error())
	}
	_ = res.Body.Close()
	return bytes.NewReader(dataOf(raw))
}

func dataOf(raw []byte) []byte {
	if len(bytes.TrimSpace(raw)) == 0 {
		return []byte("null")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		// 不是 JSON，例如媒体二进制预览。
		return raw
	}
	data, ok := probe["data"]
	if !ok {
		// 不是统一封装，例如 AGUI 的 SSE 流。
		return raw
	}
	if len(data) == 0 {
		return []byte("null")
	}
	return data
}
