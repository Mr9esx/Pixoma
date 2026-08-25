package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ReachabilityKind string

const (
	ReachabilityOK      ReachabilityKind = "ok"
	ReachabilityNetwork ReachabilityKind = "network"
	ReachabilityAuth    ReachabilityKind = "auth"
	ReachabilityOther   ReachabilityKind = "other"
)

type ReachabilityResult struct {
	OK      bool             `json:"ok"`
	Kind    ReachabilityKind `json:"kind"`
	Message string           `json:"message"`
}

const reachabilityTimeout = 3 * time.Second

func classifyGetMeError(statusCode int, err error) ReachabilityResult {
	switch {
	case err == nil && statusCode == http.StatusOK:
		return ReachabilityResult{OK: true, Kind: ReachabilityOK}
	case err == nil && (statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden):
		return ReachabilityResult{OK: false, Kind: ReachabilityAuth, Message: "unauthorized"}
	case err != nil && isNetworkError(err):
		return ReachabilityResult{OK: false, Kind: ReachabilityNetwork, Message: err.Error()}
	case err != nil:
		return ReachabilityResult{OK: false, Kind: ReachabilityOther, Message: err.Error()}
	default:
		return ReachabilityResult{OK: false, Kind: ReachabilityOther, Message: fmt.Sprintf("telegram api status %d", statusCode)}
	}
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		return networkMarker(ue.Error()) || networkMarker(ue.Err.Error())
	}
	return networkMarker(err.Error())
}

var networkMarkers = []string{
	"context deadline exceeded",
	"connection refused",
	"connection reset",
	"eof",
	"tls handshake timeout",
	"i/o timeout",
	"no such host",
	"network is unreachable",
	"proxyconnect tcp",
	"client.timeout",
}

func networkMarker(msg string) bool {
	m := strings.ToLower(msg)
	for _, marker := range networkMarkers {
		if strings.Contains(m, marker) {
			return true
		}
	}
	return false
}

func checkTelegramReachability(ctx context.Context, token string) (ReachabilityResult, error) {
	ctx, cancel := context.WithTimeout(ctx, reachabilityTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.telegram.org/bot"+token+"/getMe", nil)
	if err != nil {
		return ReachabilityResult{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return classifyGetMeError(0, err), nil
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return classifyGetMeError(resp.StatusCode, nil), nil
}
