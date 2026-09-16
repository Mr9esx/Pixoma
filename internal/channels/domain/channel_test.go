package domain

import "testing"

func TestValidPlatformRejectsRetiredEnterpriseWeChat(t *testing.T) {
	if ValidPlatform(Platform("wecom")) {
		t.Fatal("wecom must not remain a creatable message platform")
	}
}
