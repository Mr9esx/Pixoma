package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestShutdownServer_FreesPortWhileRequestInFlight(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	started := make(chan struct{})
	srv := &http.Server{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		time.Sleep(30 * time.Second)
	})}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	go func() {
		resp, err := http.Get("http://" + addr + "/")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}

	begin := time.Now()
	if err := shutdownServer(srv); err != nil && !errors.Is(err, http.ErrServerClosed) {
		t.Logf("shutdown err: %v", err)
	}
	if took := time.Since(begin); took > 3*time.Second {
		t.Fatalf("shutdown took %s; hung requests must not block reload", took)
	}

	ln2, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("port still busy after shutdown: %v", err)
	}
	_ = ln2.Close()
}

func TestFinishReload_AlwaysReturnsErrRestart(t *testing.T) {
	srv := &http.Server{Handler: http.NewServeMux()}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	err = finishReload(srv, func() {}, nil)
	if !errors.Is(err, errRestart) {
		t.Fatalf("reload must continue even if shutdown is messy, got %v", err)
	}
}
