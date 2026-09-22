package apierr

import (
	"strconv"
	"strings"
	"testing"
)

func TestRegistryIsSortedAndComplete(t *testing.T) {
	all := All()
	if len(all) == 0 {
		t.Fatal("注册表是空的")
	}
	if len(all) != len(registry) {
		t.Fatalf("All() 返回 %d 条，注册表有 %d 条", len(all), len(registry))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].Code >= all[i].Code {
			t.Fatalf("All() 没有按码升序：%d 在 %d 之前", all[i-1].Code, all[i].Code)
		}
	}
}

func TestEveryCodeIsSevenDigitsWithHTTPPrefix(t *testing.T) {
	for _, e := range All() {
		if e.Code < 1000000 || e.Code > 5999999 {
			t.Fatalf("码 %d 不是 7 位", e.Code)
		}
		status := e.Code / 10000
		if status < 100 || status > 599 {
			t.Fatalf("码 %d 的前三位 %d 不是合法 HTTP 状态码", e.Code, status)
		}
	}
}

func TestCodeMapsToOneMessageWithinSurface(t *testing.T) {
	// 同一个 (HTTP, 业务域) 内，一个文案只能对应一个码；
	// 跨域或跨状态码复用同一句话是允许的（例如「登录已失效。重新登录。」）。
	bySurface := map[[2]int]map[string]int{}
	for _, e := range All() {
		surface := [2]int{e.Code / 10000, (e.Code / 100) % 100}
		if bySurface[surface] == nil {
			bySurface[surface] = map[string]int{}
		}
		if prev, dup := bySurface[surface][e.Message]; dup {
			t.Fatalf("码 %d 与 %d 同属 HTTP %d 业务域 %d，却共用文案 %q",
				e.Code, prev, surface[0], surface[1], e.Message)
		}
		bySurface[surface][e.Message] = e.Code
	}
}

func TestEveryCodeHasMessageAndI18nKey(t *testing.T) {
	for _, e := range All() {
		if strings.TrimSpace(e.Message) == "" {
			t.Fatalf("码 %d 没有文案", e.Code)
		}
		want := "apiError." + strconv.Itoa(e.Code)
		if e.I18nKey != want {
			t.Fatalf("码 %d 的 I18nKey = %q，期望 %q", e.Code, e.I18nKey, want)
		}
		if got := I18nKeyFor(e.Code); got != want {
			t.Fatalf("I18nKeyFor(%d) = %q，期望 %q", e.Code, got, want)
		}
	}
}

func TestByCodeRoundTrips(t *testing.T) {
	for _, e := range All() {
		got, ok := ByCode(e.Code)
		if !ok {
			t.Fatalf("ByCode(%d) 没找到", e.Code)
		}
		if got != e {
			t.Fatalf("ByCode(%d) 返回的不是注册表里的那一条", e.Code)
		}
	}
	if _, ok := ByCode(4999999); ok {
		t.Fatal("未注册的码不应被找到")
	}
}

func TestErrorMessageIsStable(t *testing.T) {
	// 抽样固定几条，防止改文案时误伤既有码。
	cases := map[int]string{
		4000602: "创建用例失败。检查请求参数后重试。",
		4000612: "请求体格式不正确。检查字段格式后重试。",
		4090604: "用例被菜单或卡片引用。勾选「同时清理引用」后删除。",
		5000005: "服务暂时不可用。重启 `pixoma` 后重试。",
	}
	for code, want := range cases {
		e, ok := ByCode(code)
		if !ok {
			t.Fatalf("码 %d 未注册", code)
		}
		if e.Message != want {
			t.Fatalf("码 %d 文案 = %q，期望 %q", code, e.Message, want)
		}
	}
}

func TestSuccessCodeIsNotInRegistry(t *testing.T) {
	if _, ok := ByCode(SuccessCode); ok {
		t.Fatal("成功码不应出现在错误注册表里")
	}
	if SuccessCode != 2000000 {
		t.Fatalf("成功码 = %d，期望 2000000", SuccessCode)
	}
}
