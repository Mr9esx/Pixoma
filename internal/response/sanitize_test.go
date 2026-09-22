package response

import "testing"

func TestSanitizeRedactsCredentials(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "DSN 里的密码",
			in:   "dial tcp: postgres://pixoma:s3cret@10.0.0.7:5432/pixoma refused",
			want: "dial tcp: postgres://***:***@10.0.0.7:5432/pixoma refused",
		},
		{
			name: "mysql DSN 里的密码",
			in:   "open mysql://root:hunter2@db.internal:3306/pixoma: access denied",
			want: "open mysql://***:***@db.internal:3306/pixoma: access denied",
		},
		{
			name: "键值形式的密码",
			in:   "blob client: password=hunter2 endpoint=obs.cn-north-4.myhuaweicloud.com",
			want: "blob client: password=*** endpoint=obs.cn-north-4.myhuaweicloud.com",
		},
		{
			name: "JSON 形式的密钥",
			in:   `{"api_key":"sk-abc123","region":"cn"}`,
			want: `{"api_key":"***","region":"cn"}`,
		},
		{
			name: "access key 与 secret key",
			in:   "AccessKey=AKIA123 SecretKey=xyz789",
			want: "AccessKey=*** SecretKey=***",
		},
		{
			name: "Bearer 令牌",
			in:   "upstream returned 401 for Authorization: Bearer eyJhbGciOi.abc-123",
			want: "upstream returned 401 for Authorization: Bearer ***",
		},
		{
			name: "换行与制表符压平",
			in:   "query failed:\n\tSELECT * FROM users\n\tWHERE id = 1",
			want: "query failed: SELECT * FROM users WHERE id = 1",
		},
		{
			name: "空串",
			in:   "   ",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Sanitize(tc.in); got != tc.want {
				t.Fatalf("Sanitize(%q)\n 得到 %q\n 期望 %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSanitizeTruncatesLongDetail(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "错误"
	}
	got := Sanitize(long)
	if len(got) > maxDetailLen+len("…") {
		t.Fatalf("截断后长度 %d 超过上限 %d", len(got), maxDetailLen)
	}
	if got[len(got)-len("…"):] != "…" {
		t.Fatalf("截断后应以省略号结尾，实际 %q", got[len(got)-4:])
	}
	if !isValidUTF8(got) {
		t.Fatal("截断把多字节字符切成了半个")
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}
