package acp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type handshakeRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type handshakeReply struct {
	Result string
	Code   int
	Error  string
	Stop   bool
}

func startHandshakeServer(t *testing.T, initResult string, handler func(handshakeRequest) handshakeReply) (*acpSession, <-chan struct{}) {
	t.Helper()
	s, wResp, rReq := newTestSession(t, nil)
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer wResp.Close()
		scanner := bufio.NewScanner(rReq)
		for scanner.Scan() {
			var req handshakeRequest
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}
			if req.Method == "initialize" {
				_, _ = fmt.Fprintf(wResp, `{"jsonrpc":"2.0","id":%s,"result":%s}`+"\n", req.ID, initResult)
				continue
			}
			reply := handler(req)
			if reply.Code != 0 {
				_, _ = fmt.Fprintf(wResp, `{"jsonrpc":"2.0","id":%s,"error":{"code":%d,"message":%q}}`+"\n", req.ID, reply.Code, reply.Error)
			} else {
				result := reply.Result
				if result == "" {
					result = `{}`
				}
				_, _ = fmt.Fprintf(wResp, `{"jsonrpc":"2.0","id":%s,"result":%s}`+"\n", req.ID, result)
			}
			if reply.Stop {
				return
			}
		}
	}()
	t.Cleanup(func() {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Errorf("fake ACP server did not stop")
		}
	})
	return s, done
}

func assertResumeParams(t *testing.T, req handshakeRequest, method string, sessionID string) {
	t.Helper()
	if req.Method != method {
		t.Fatalf("method = %q, want %q", req.Method, method)
	}
	var params struct {
		SessionID  string `json:"sessionId"`
		Cwd        string `json:"cwd"`
		MCPServers []any  `json:"mcpServers"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("decode %s params: %v", method, err)
	}
	if params.SessionID != sessionID {
		t.Fatalf("%s sessionId = %q, want %q", method, params.SessionID, sessionID)
	}
	if !filepath.IsAbs(params.Cwd) {
		t.Fatalf("%s cwd = %q, want absolute path", method, params.Cwd)
	}
	if params.MCPServers == nil || len(params.MCPServers) != 0 {
		t.Fatalf("%s mcpServers = %#v, want empty array", method, params.MCPServers)
	}
}

func waitHandshakeServer(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("fake ACP server did not receive the expected handshake")
	}
}

func TestHandshake_UsesLegacyLoadBeforeResume(t *testing.T) {
	s, done := startHandshakeServer(t, `{"protocolVersion":1,"agentCapabilities":{"loadSession":true,"sessionCapabilities":{"resume":{}}}}`, func(req handshakeRequest) handshakeReply {
		assertResumeParams(t, req, "session/load", "persisted-id")
		return handshakeReply{Result: `{"sessionId":"loaded-id"}`, Stop: true}
	})

	if err := s.handshake("persisted-id", ""); err != nil {
		t.Fatalf("handshake() error = %v", err)
	}
	waitHandshakeServer(t, done)
	if got := s.currentACPSessionID(); got != "loaded-id" {
		t.Fatalf("session id = %q, want loaded-id", got)
	}
}

func TestHandshake_UsesStandardResumeWhenLegacyLoadUnsupported(t *testing.T) {
	s, done := startHandshakeServer(t, `{"protocolVersion":1,"agentCapabilities":{"sessionCapabilities":{"resume":{}}}}`, func(req handshakeRequest) handshakeReply {
		assertResumeParams(t, req, "session/resume", "persisted-id")
		return handshakeReply{Result: `{"sessionId":"resumed-id"}`, Stop: true}
	})

	if err := s.handshake("persisted-id", ""); err != nil {
		t.Fatalf("handshake() error = %v", err)
	}
	waitHandshakeServer(t, done)
	if got := s.currentACPSessionID(); got != "resumed-id" {
		t.Fatalf("session id = %q, want resumed-id", got)
	}
}

func TestHandshake_CreatesNewWhenResumeAndLoadUnsupported(t *testing.T) {
	s, done := startHandshakeServer(t, `{"protocolVersion":1,"agentCapabilities":{"sessionCapabilities":{}}}`, func(req handshakeRequest) handshakeReply {
		if req.Method != "session/new" {
			t.Fatalf("method = %q, want session/new", req.Method)
		}
		return handshakeReply{Result: `{"sessionId":"new-id"}`, Stop: true}
	})

	if err := s.handshake("persisted-id", ""); err != nil {
		t.Fatalf("handshake() error = %v", err)
	}
	waitHandshakeServer(t, done)
	if got := s.currentACPSessionID(); got != "new-id" {
		t.Fatalf("session id = %q, want new-id", got)
	}
}

func TestHandshake_ResumeErrorDoesNotCreateNewSession(t *testing.T) {
	s, done := startHandshakeServer(t, `{"protocolVersion":1,"agentCapabilities":{"sessionCapabilities":{"resume":{}}}}`, func(req handshakeRequest) handshakeReply {
		assertResumeParams(t, req, "session/resume", "persisted-id")
		return handshakeReply{Code: -32001, Error: "resume failed", Stop: true}
	})

	err := s.handshake("persisted-id", "")
	waitHandshakeServer(t, done)
	if err == nil || !strings.Contains(err.Error(), "session/resume") || !strings.Contains(err.Error(), "resume failed") {
		t.Fatalf("handshake() error = %v, want contextual session/resume failure", err)
	}
}

func TestHandshake_ResumeResponseWithoutSessionIDUsesRequestedID(t *testing.T) {
	s, done := startHandshakeServer(t, `{"protocolVersion":1,"agentCapabilities":{"sessionCapabilities":{"resume":{}}}}`, func(req handshakeRequest) handshakeReply {
		assertResumeParams(t, req, "session/resume", "persisted-id")
		return handshakeReply{Result: `{}`, Stop: true}
	})

	if err := s.handshake("persisted-id", ""); err != nil {
		t.Fatalf("handshake() error = %v", err)
	}
	waitHandshakeServer(t, done)
	if got := s.currentACPSessionID(); got != "persisted-id" {
		t.Fatalf("session id = %q, want requested persisted-id", got)
	}
}

func TestHandshake_LegacyLoadErrorFallsBackToNew(t *testing.T) {
	s, done := startHandshakeServer(t, `{"protocolVersion":1,"agentCapabilities":{"loadSession":true,"sessionCapabilities":{"resume":{}}}}`, func(req handshakeRequest) handshakeReply {
		switch req.Method {
		case "session/load":
			assertResumeParams(t, req, "session/load", "persisted-id")
			return handshakeReply{Code: -32001, Error: "legacy load failed"}
		case "session/new":
			return handshakeReply{Result: `{"sessionId":"new-after-load-error"}`, Stop: true}
		default:
			t.Fatalf("unexpected method %q", req.Method)
			return handshakeReply{Stop: true}
		}
	})

	if err := s.handshake("persisted-id", ""); err != nil {
		t.Fatalf("handshake() error = %v, want legacy fallback to succeed", err)
	}
	waitHandshakeServer(t, done)
	if got := s.currentACPSessionID(); got != "new-after-load-error" {
		t.Fatalf("session id = %q, want new-after-load-error", got)
	}
}
