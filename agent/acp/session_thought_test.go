package acp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/chenhg5/cc-connect/core"
)

// Exercise the notification boundary, not only the dedicated thought mapper:
// the generic fallback previously emitted the same reasoning as answer text.
func TestNotificationThoughtChunkEmitsOnlyThinking(t *testing.T) {
	s := &acpSession{ctx: context.Background(), events: make(chan core.Event, 8)}
	s.onNotification("session/update", json.RawMessage(`{"sessionId":"dsh-session","update":{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"private reasoning"}}}`))
	s.onNotification("session/update", json.RawMessage(`{"sessionId":"dsh-session","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"public answer"}}}`))
	if len(s.events) != 2 {
		t.Fatalf("got %d events, want one thought and one answer (no reasoning as text)", len(s.events))
	}
	for _, want := range []core.Event{{Type: core.EventThinking, Content: "private reasoning", SessionID: "dsh-session"}, {Type: core.EventText, Content: "public answer", SessionID: "dsh-session"}} {
		got := <-s.events
		if got.Type != want.Type || got.Content != want.Content || got.SessionID != want.SessionID {
			t.Fatalf("event = %+v, want %+v", got, want)
		}
	}
}

func TestMapSessionUpdateThoughtChunk(t *testing.T) {
	params := json.RawMessage(`{
		"sessionId":"dsh-session",
		"update":{
			"sessionUpdate":"agent_thought_chunk",
			"content":{"type":"text","text":"considering the next step"}
		}
	}`)

	got := mapSessionUpdate("fallback-session", params)
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

func TestMapSessionUpdateThoughtChunkDoesNotSwallowAnswer(t *testing.T) {
	params := json.RawMessage(`{"sessionId":"dsh-session","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"answer"}}}`)
	got := mapSessionUpdate("fallback-session", params)
	if len(got) != 1 || got[0].Type != core.EventText || got[0].Content != "answer" {
		t.Fatalf("answer incorrectly mapped: %+v", got)
	}
}
