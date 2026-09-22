package apierr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// localeDir 相对本包目录定位前端词条。
const localeDir = "../../web/admin/src/lib/i18n/locales"

// TestEveryCodeHasFrontendCopy 保证每个错误码在中文与英文词条里都有对应文案。
//
// 前端按 code 查词条，缺条目时会退回后端文案或者通用提示，用户看不到具体原因。
// 这条测试把「后端加码但忘记补词条」拦在提交之前。
func TestEveryCodeHasFrontendCopy(t *testing.T) {
	for _, locale := range []string{"zh", "en"} {
		path := filepath.Join(localeDir, locale+".json")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", path, err)
		}
		var doc struct {
			ApiError map[string]string `json:"apiError"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("解析 %s 失败: %v", path, err)
		}
		if len(doc.ApiError) == 0 {
			t.Fatalf("%s 里没有 apiError 词条", path)
		}

		for _, e := range All() {
			key := strconv.Itoa(e.Code)
			text, ok := doc.ApiError[key]
			if !ok {
				t.Errorf("%s 缺少码 %d 的词条", locale, e.Code)
				continue
			}
			if text == "" {
				t.Errorf("%s 里码 %d 的词条是空的", locale, e.Code)
			}
		}

		// 反向检查：词条里的码必须在注册表里，避免删除错误码后留下死词条。
		for key := range doc.ApiError {
			code, err := strconv.Atoi(key)
			if err != nil {
				t.Errorf("%s 里的词条键 %q 不是数字码", locale, key)
				continue
			}
			if _, ok := ByCode(code); !ok {
				t.Errorf("%s 里的词条 %d 在注册表中不存在", locale, code)
			}
		}
	}
}

// TestEnglishCopyHasNoChinese 保证英文词条确实是英文。
func TestEnglishCopyHasNoChinese(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(localeDir, "en.json"))
	if err != nil {
		t.Fatalf("读取 en.json 失败: %v", err)
	}
	var doc struct {
		ApiError map[string]string `json:"apiError"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("解析 en.json 失败: %v", err)
	}
	for key, text := range doc.ApiError {
		for _, r := range text {
			if r >= 0x4e00 && r <= 0x9fff {
				t.Errorf("英文词条 %s 含中文: %q", key, text)
				break
			}
		}
	}
}
