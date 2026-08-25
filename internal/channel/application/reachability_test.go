package application

import (
	"context"
	"errors"
	"net/url"
	"testing"
)

func TestClassifyGetMeError(t *testing.T) {
	cases := []struct {
		name     string
		status   int
		err      error
		wantOK   bool
		wantKind ReachabilityKind
	}{
		{"ok", 200, nil, true, ReachabilityOK},
		{"unauthorized", 401, nil, false, ReachabilityAuth},
		{"forbidden", 403, nil, false, ReachabilityAuth},
		{"deadline", 0, context.DeadlineExceeded, false, ReachabilityNetwork},
		{"connection refused", 0, &url.Error{Op: "Post", URL: "https://api.telegram.org", Err: errors.New("dial tcp 149.154.167.220:443: connect: connection refused")}, false, ReachabilityNetwork},
		{"eof", 0, errors.New("unexpected EOF"), false, ReachabilityNetwork},
		{"no such host", 0, &url.Error{Err: errors.New("dial tcp: lookup api.telegram.org: no such host")}, false, ReachabilityNetwork},
		{"tls timeout", 0, &url.Error{Err: errors.New("net/http: TLS handshake timeout")}, false, ReachabilityNetwork},
		{"other error", 0, errors.New("boom"), false, ReachabilityOther},
		{"server error", 500, nil, false, ReachabilityOther},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyGetMeError(tc.status, tc.err)
			if got.OK != tc.wantOK || got.Kind != tc.wantKind {
				t.Fatalf("got %+v", got)
			}
		})
	}
}
