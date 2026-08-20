package db

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestGormLogger_IgnoresRecordNotFound(t *testing.T) {
	var buf bytes.Buffer
	l := newGormLogger(false, &buf)
	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM `tasks` WHERE status = queued", 0
	}, gorm.ErrRecordNotFound)
	got := buf.String()
	if strings.Contains(got, "record not found") || strings.Contains(got, "SELECT") {
		t.Fatalf("empty claim must not log, got %q", got)
	}
}

func TestGormLogger_StillLogsRealErrors(t *testing.T) {
	var buf bytes.Buffer
	l := newGormLogger(false, &buf)
	l.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT 1", 0
	}, gorm.ErrInvalidDB)
	if !strings.Contains(buf.String(), "invalid db") && !strings.Contains(buf.String(), "SELECT 1") {
		t.Fatalf("real errors must still log, got %q", buf.String())
	}
}
