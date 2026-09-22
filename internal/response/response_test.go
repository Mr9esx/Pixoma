package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mr9esx/Pixoma/internal/apierr"
)

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("响应不是合法 JSON：%v（body=%q）", err, rec.Body.String())
	}
	return body
}

func TestOKWrapsData(t *testing.T) {
	rec := httptest.NewRecorder()
	OK(rec, map[string]any{"id": "case-1"})

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 %d，期望 200", rec.Code)
	}
	body := decode(t, rec)
	if body["message"] != SuccessMessage {
		t.Fatalf("message = %v，期望 %q", body["message"], SuccessMessage)
	}
	if body["code"] != float64(apierr.SuccessCode) {
		t.Fatalf("code = %v，期望 %d", body["code"], apierr.SuccessCode)
	}
	data, ok := body["data"].(map[string]any)
	if !ok || data["id"] != "case-1" {
		t.Fatalf("data = %v，期望透传原始数据", body["data"])
	}
	if _, exists := body["error_detail"]; exists {
		t.Fatal("成功响应不应出现 error_detail")
	}
}

func TestOKStatusTurns204Into200(t *testing.T) {
	rec := httptest.NewRecorder()
	OKStatus(rec, http.StatusNoContent, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 %d，期望 204 被改写成 200", rec.Code)
	}
	body := decode(t, rec)
	if body["code"] != float64(apierr.SuccessCode) {
		t.Fatalf("code = %v，期望成功码", body["code"])
	}
	if body["data"] != nil {
		t.Fatalf("data = %v，期望 null", body["data"])
	}
}

func TestOKStatusKeepsCreated(t *testing.T) {
	rec := httptest.NewRecorder()
	OKStatus(rec, http.StatusCreated, map[string]any{"id": "edge-1"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("状态码 %d，期望 201", rec.Code)
	}
}

func TestFailUsesStatusFromCode(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, apierr.ErrCaseGetNotFound, "case abc not found")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("状态码 %d，期望 404", rec.Code)
	}
	body := decode(t, rec)
	if body["message"] != apierr.ErrCaseGetNotFound.Message {
		t.Fatalf("message = %v，期望注册表文案", body["message"])
	}
	if body["code"] != float64(apierr.ErrCaseGetNotFound.Code) {
		t.Fatalf("code = %v，期望 %d", body["code"], apierr.ErrCaseGetNotFound.Code)
	}
	if body["data"] != nil {
		t.Fatalf("data = %v，期望 null", body["data"])
	}
	if body["error_detail"] != "case abc not found" {
		t.Fatalf("error_detail = %v，期望原始原因", body["error_detail"])
	}
}

func TestFailOmitsEmptyDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, apierr.ErrCaseCreateInvalidJSON, "")

	if _, exists := decode(t, rec)["error_detail"]; exists {
		t.Fatal("detail 为空时不应出现 error_detail")
	}
}

func TestFailNeverLeaksDetailOn401And403(t *testing.T) {
	for _, e := range []*apierr.Error{apierr.ErrSetupSessionUnauthorized, apierr.ErrSetupSessionForbidden} {
		rec := httptest.NewRecorder()
		Fail(rec, e, "session abc123 belongs to admin@corp but was revoked")

		body := decode(t, rec)
		if _, exists := body["error_detail"]; exists {
			t.Fatalf("码 %d 不应带 error_detail，实际 %v", e.Code, body["error_detail"])
		}
	}
}

func TestFailSanitizesDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, apierr.ErrSetupTestDatabaseFailed, "postgres://pixoma:s3cret@10.0.0.7:5432 refused")

	got := decode(t, rec)["error_detail"]
	if got != "postgres://***:***@10.0.0.7:5432 refused" {
		t.Fatalf("error_detail = %v，密码没有被抹掉", got)
	}
}

func TestFailNilErrorFallsBackToInternal(t *testing.T) {
	rec := httptest.NewRecorder()
	Fail(rec, nil, "boom")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("状态码 %d，期望 500", rec.Code)
	}
	if decode(t, rec)["code"] != float64(apierr.ErrCommonInternal.Code) {
		t.Fatal("应退化为通用 500")
	}
}

func TestFailCodeUnknownFallsBackToInternal(t *testing.T) {
	rec := httptest.NewRecorder()
	FailCode(rec, 4999999, "unregistered")

	if decode(t, rec)["code"] != float64(apierr.ErrCommonInternal.Code) {
		t.Fatal("未注册的码应退化为通用 500")
	}
}

func TestFailErrPassesErrorText(t *testing.T) {
	rec := httptest.NewRecorder()
	FailErr(rec, apierr.ErrChannelListFailed, errString("db is gone"))

	if decode(t, rec)["error_detail"] != "db is gone" {
		t.Fatal("FailErr 应把 err.Error() 放进 error_detail")
	}
}

func TestStatusOfReadsLeadingDigits(t *testing.T) {
	cases := map[int]int{
		2000000: 200,
		4000602: 400,
		4090604: 409,
		5000005: 500,
		5021219: 502,
		0:       http.StatusInternalServerError,
		9999999: http.StatusInternalServerError,
	}
	for code, want := range cases {
		if got := StatusOf(code); got != want {
			t.Fatalf("StatusOf(%d) = %d，期望 %d", code, got, want)
		}
	}
}

func TestContentTypeIsJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	OK(rec, nil)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
