package app

import (
	"os"
	"strings"

	"github.com/mr9esx/comfyui_tgbot/internal/platform/settings"
)

const lanNoProxy = "localhost,127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"

// ApplyHTTPProxy copies settings/env proxy into process env for Telegram and other HTTPS.
func ApplyHTTPProxy(cfg settings.Settings) {
	envProxy := firstNonEmpty(
		os.Getenv("HTTPS_PROXY"),
		os.Getenv("https_proxy"),
		os.Getenv("HTTP_PROXY"),
		os.Getenv("http_proxy"),
		os.Getenv("ALL_PROXY"),
		os.Getenv("all_proxy"),
	)
	proxy := envProxy
	if strings.TrimSpace(proxy) == "" {
		proxy = cfg.ProxyURL()
	}
	if strings.TrimSpace(proxy) == "" {
		return
	}
	setIfEmpty := func(key, val string) {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return
		}
		_ = os.Setenv(key, val)
	}
	setIfEmpty("HTTP_PROXY", proxy)
	setIfEmpty("HTTPS_PROXY", proxy)
	setIfEmpty("ALL_PROXY", proxy)
	mergeNoProxy(lanNoProxy)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func mergeNoProxy(extra string) {
	cur := firstNonEmpty(os.Getenv("NO_PROXY"), os.Getenv("no_proxy"))
	seen := map[string]bool{}
	var parts []string
	for _, p := range strings.Split(cur+","+extra, ",") {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		parts = append(parts, p)
	}
	_ = os.Setenv("NO_PROXY", strings.Join(parts, ","))
}
