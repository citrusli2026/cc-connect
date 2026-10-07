package acp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func startCloseServer(t *testing.T, respond bool) (*acpSession, <-chan string) {
	t.Helper()
	s, wResp, rReq := newTestSession(t, nil)
	s.closeSupported.Store(true)
	methods := make(chan string, 1)
	go func() {
		defer wResp.Close()
		scanner := bufio.NewScanner(rReq)
		if !scanner.Scan() {
			return
		}
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			t.Errorf("decode close request: %v", err)
			return
		}
		methods <- req.Method
		if respond {
			_, _ = fmt.Fprintf(wResp, `{"jsonrpc":"2.0","id":%s,"result":{}}`+"\n", req.ID)
		}
	}()
	return s, methods
}

func TestSession_CloseRequestsServerClose(t *testing.T) {
	s, methods := startCloseServer(t, true)

	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case method := <-methods:
		if method != "session/close" {
			t.Fatalf("method = %q, want session/close", method)
		}
	case <-time.After(time.Second):
		t.Fatal("session/close was not requested")
	}
	if s.Alive() {
		t.Fatal("session should not be alive after Close")
	}
	if _, ok := <-s.Events(); ok {
		t.Fatal("Events channel should be closed after Close")
	}
}

func TestSession_CloseTimeoutStillTerminates(t *testing.T) {
	s, methods := startCloseServer(t, false)
	started := time.Now()
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	elapsed := time.Since(started)
	select {
	case method := <-methods:
		if method != "session/close" {
			t.Fatalf("method = %q, want session/close", method)
		}
	case <-time.After(time.Second):
		t.Fatal("session/close was not requested")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("Close() took %s, want timeout bounded near two seconds", elapsed)
	}
	if s.Alive() {
		t.Fatal("session should not be alive after close timeout")
	}
}

func TestSession_CloseUnsupportedSkipsRPC(t *testing.T) {
	s, _, _ := newTestSession(t, nil)
	started := time.Now()
	if err := s.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("unsupported Close() took %s, want immediate teardown", elapsed)
	}
}

func TestSession_CloseIsIdempotent(t *testing.T) {
	s, methods := startCloseServer(t, true)
	if err := s.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	select {
	case method := <-methods:
		if method != "session/close" {
			t.Fatalf("method = %q, want session/close", method)
		}
	case <-time.After(time.Second):
		t.Fatal("session/close was not requested")
	}
	select {
	case method := <-methods:
		t.Fatalf("Close() requested RPC twice: %q", method)
	case <-time.After(50 * time.Millisecond):
	}
}
