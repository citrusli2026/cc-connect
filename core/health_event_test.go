package core

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

type healthLogBuffer struct {
	sync.Mutex
	bytes.Buffer
}

func (b *healthLogBuffer) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.Buffer.Write(p)
}

func TestHealthEvent_OptInVersionedAndContentFree(t *testing.T) {
	previous := slog.Default()
	var buffer healthLogBuffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buffer, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	t.Setenv("DSH_CC_CONNECT_HEALTH", "0")
	if ReportHealth(HealthEvent{Component: "channel", State: "online"}) {
		t.Fatal("health contract must be opt-in")
	}
	t.Setenv("DSH_CC_CONNECT_HEALTH", "1")
	for _, event := range []HealthEvent{
		{Component: "channel", State: "connecting"}, {Component: "channel", State: "online"},
		{Component: "channel", State: "offline", ErrorCode: "network"},
		{Component: "agent", State: "available"}, {Component: "agent", State: "unavailable", ErrorCode: "model-key-missing"},
	} {
		if !ReportHealth(event) {
			t.Fatalf("valid event rejected: %+v", event)
		}
	}
	for _, event := range []HealthEvent{
		{Component: "fake-secret", State: "online"}, {Component: "channel", State: "fake-secret"},
		{Component: "agent", State: "online"}, {Component: "agent", State: "available", ErrorCode: "fake-secret"},
	} {
		if ReportHealth(event) {
			t.Fatal("unbounded event accepted")
		}
	}
	buffer.Lock()
	output := buffer.Buffer.String()
	buffer.Unlock()
	if strings.Count(output, `msg="cc-connect health"`) != 5 || !strings.Contains(output, `\"version\":1`) || strings.Contains(output, "fake-secret") {
		t.Fatalf("invalid contract output: %s", output)
	}
}
