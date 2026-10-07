package acp

import (
	"encoding/json"
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

func TestMapDSHThoughtChunk(t *testing.T) {
	params := json.RawMessage(`{
		"sessionId":"dsh-session",
		"update":{
			"sessionUpdate":"agent_thought_chunk",
			"content":{"type":"text","text":"considering the next step"}
		}
	}`)

	got := mapDSHThoughtChunk("fallback-session", params)
	if len(got) != 1 {
		t.Fatalf("got %d events, want one: %+v", len(got), got)
	}
	if got[0].Type != core.EventThinking {
		t.Fatalf("event type = %v, want EventThinking", got[0].Type)
	}
	if got[0].SessionID != "dsh-session" || got[0].Content != "considering the next step" {
		t.Fatalf("event = %+v, want DSH thought content and session", got[0])
	}
}

func TestMapDSHThoughtChunk_IgnoresOtherUpdates(t *testing.T) {
	params := json.RawMessage(`{"sessionId":"dsh-session","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"answer"}}}`)
	if got := mapDSHThoughtChunk("fallback-session", params); got != nil {
		t.Fatalf("got events for non-thought update: %+v", got)
	}
}
