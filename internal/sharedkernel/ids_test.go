package sharedkernel

import "testing"

func TestChatIDRoundTrip(t *testing.T) {
	addr := ChannelAddr{ChannelID: "tg-default", ExternalChatID: "123456789"}
	got, err := ParseChatID(FormatChatID(addr))
	if err != nil {
		t.Fatal(err)
	}
	if got != addr {
		t.Fatalf("roundtrip: %+v != %+v", got, addr)
	}
}

func TestParseChatIDInvalid(t *testing.T) {
	if _, err := ParseChatID("noprefix"); err == nil {
		t.Fatal("want error")
	}
	if _, err := ParseChatID("tg:"); err == nil {
		t.Fatal("want error for empty external id")
	}
}
