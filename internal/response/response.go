// Package response 写出 Pixoma 的统一响应封装。
//
// 成功：{"message":"success","code":2000000,"data":...}
// 失败：{"message":"...","code":4xxxxxx,"data":null,"error_detail":"..."}
//
// 所有 JSON 接口都必须经过本包，前端才能按同一个形状解包。
package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Mr9esx/Pixoma/internal/apierr"
)

// SuccessMessage 是成功响应固定的 message。
const SuccessMessage = "success"

// Envelope 是所有 JSON 接口的统一响应体。
//
// Data 在错误响应里固定为 null；ErrorDetail 只在错误响应里出现，
// 携带脱敏后的原始技术原因，由前端直接展示。
type Envelope struct {
	Message     string `json:"message"`
	Code        int    `json:"code"`
	Data        any    `json:"data"`
	ErrorDetail string `json:"error_detail,omitempty"`
}

// OK 写出 200 成功响应。
func OK(w http.ResponseWriter, data any) {
	OKStatus(w, http.StatusOK, data)
}

// OKStatus 写出指定状态码的成功响应。
//
// 204 没有响应体，统一改成 200 加封装，避免前端拿到空 body 无法解包。
func OKStatus(w http.ResponseWriter, status int, data any) {
	if status == http.StatusNoContent {
		status = http.StatusOK
	}
	write(w, status, Envelope{
		Message: SuccessMessage,
		Code:    apierr.SuccessCode,
		Data:    data,
	})
}

// Fail 写出错误响应。
//
// detail 是原始技术原因，会先脱敏再放进 error_detail；为空则省略该字段。
// 401 与 403 一律不带 error_detail，避免从技术原因反推资源是否存在。
func Fail(w http.ResponseWriter, e *apierr.Error, detail string) {
	if e == nil {
		e = apierr.ErrCommonInternal
	}
	env := Envelope{Message: e.Message, Code: e.Code}
	if detailAllowed(e.Code) {
		env.ErrorDetail = Sanitize(detail)
	}
	write(w, StatusOf(e.Code), env)
}

// FailErr 用 err.Error() 作为 error_detail 写出错误响应。
func FailErr(w http.ResponseWriter, e *apierr.Error, err error) {
	if err == nil {
		Fail(w, e, "")
		return
	}
	Fail(w, e, err.Error())
}

// FailCode 按错误码写出错误响应。码未注册时退化为通用 500。
func FailCode(w http.ResponseWriter, code int, detail string) {
	e, ok := apierr.ByCode(code)
	if !ok {
		e = apierr.ErrCommonInternal
	}
	Fail(w, e, detail)
}

// StatusOf 从 7 位错误码取出 HTTP 状态码。
//
// 码位是 [HTTP:3][业务域:2][明细:2]，所以前三位就是状态码。
func StatusOf(code int) int {
	status := code / 10000
	if status < 100 || status > 599 {
		return http.StatusInternalServerError
	}
	return status
}

// detailAllowed 决定哪些错误可以带 error_detail。
func detailAllowed(code int) bool {
	switch StatusOf(code) {
	case http.StatusUnauthorized, http.StatusForbidden:
		return false
	}
	return true
}

func write(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(env); err != nil {
		// 响应头已经写出，这里只能记录。客户端会看到截断的 body。
		slog.Error("写出响应失败", "code", env.Code, "error", err)
	}
}
