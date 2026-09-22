package response

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// maxDetailLen 是 error_detail 的字节上限，超出部分截断。
const maxDetailLen = 500

var (
	// scheme://user:password@host —— 只保留协议和用户名位置，密码替换掉。
	reURLCred = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.\-]*://)[^/\s:@]+:[^/\s@]*@`)
	// password=xxx、"secret": "xxx" 之类的键值对。
	// authorization 不在这里处理，否则会把后面的 Bearer 当成密钥值。
	reKVSecret = regexp.MustCompile(
		`(?i)\b(password|passwd|pwd|secret|token|api[_-]?key|access[_-]?key|secret[_-]?key)\b("?\s*[:=]\s*)("?)([^\s,;"'&}]+)`)
	// Authorization 头里的 Bearer / Basic 凭据。
	reBearer = regexp.MustCompile(`(?i)\b(bearer|basic)\s+[A-Za-z0-9+/=._\-]+`)
	// 连续空白（含换行与制表符）压成单个空格。
	reSpace = regexp.MustCompile(`\s+`)
)

// Sanitize 清洗 error_detail：抹掉凭据，压平空白，限制长度。
//
// 原始 Go 错误经常带 DSN、内网地址、SQL 片段和上游密钥，这些不能直接给前端。
// 清洗只在本包做一次，调用点不需要重复处理。
func Sanitize(detail string) string {
	s := strings.TrimSpace(detail)
	if s == "" {
		return ""
	}
	s = reURLCred.ReplaceAllString(s, "${1}***:***@")
	s = reKVSecret.ReplaceAllString(s, "${1}${2}${3}***")
	s = reBearer.ReplaceAllString(s, "${1} ***")
	s = reSpace.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if len(s) > maxDetailLen {
		s = truncateUTF8(s, maxDetailLen) + "…"
	}
	return s
}

// truncateUTF8 在不超过 limit 字节的前提下尽量多截，避免把多字节字符切成半个。
func truncateUTF8(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := s[:limit]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut
}
