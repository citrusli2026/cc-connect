package acp

import (
	"strings"
	"testing"
)

func TestTurnResultHandler_ObservesActualPromptFailureAndSuccessWithoutChangingEvents(t *testing.T) {
	for _, fail := range []bool{true, false} {
		s, done := startHandshakeServer(t, `{"protocolVersion":1}`, func(req handshakeRequest) handshakeReply {
			if req.Method == "session/new" {
				return handshakeReply{Result: `{"sessionId":"fake-health-session"}`}
			}
			if fail {
				return handshakeReply{Code: -32603, Error: `no API key for provider route "fake-route"; fake-secret`, Stop: true}
			}
			return handshakeReply{Result: `{"stopReason":"end_turn"}`, Stop: true}
		})
		if err := s.handshake("", ""); err != nil {
			t.Fatal(err)
		}
		var observed error
		calls := 0
		s.turnResult = func(err error) { observed = err; calls++ }
		err := s.Send("fake private prompt", "fake-message", nil, nil)
		if calls != 1 || (err != nil) != fail || (observed != nil) != fail {
			t.Fatalf("calls=%d err=%v observed=%v", calls, err, observed)
		}
		if fail && !strings.Contains(observed.Error(), "no API key for provider route") {
			t.Fatal("missing credential error context")
		}
		waitHandshakeServer(t, done)
	}
}
