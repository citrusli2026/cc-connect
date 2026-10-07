package acp

import (
	"testing"
	"time"
)

func TestConvertSessionList_DSHCreatedAtMilliseconds(t *testing.T) {
	createdAtMillis := int64(1791379200123)
	got := convertSessionList([]acpSessionListEntry{
		{SessionID: "dsh-session", Cwd: "/tmp/project", CreatedAt: &createdAtMillis},
	}, "/tmp/project")
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want one", len(got))
	}
	want := time.UnixMilli(createdAtMillis).UTC()
	if !got[0].ModifiedAt.Equal(want) {
		t.Fatalf("ModifiedAt = %s, want DSH Date.now milliseconds %s", got[0].ModifiedAt, want)
	}
}

func TestConvertSessionList_UpdatedAtTakesPrecedence(t *testing.T) {
	createdAtMillis := int64(1791379200123)
	got := convertSessionList([]acpSessionListEntry{
		{
			SessionID: "session",
			Cwd:       "/tmp/project",
			UpdatedAt: "2026-10-07T00:00:00Z",
			CreatedAt: &createdAtMillis,
		},
	}, "/tmp/project")
	if len(got) != 1 {
		t.Fatalf("got %d sessions, want one", len(got))
	}
	want, _ := time.Parse(time.RFC3339, "2026-10-07T00:00:00Z")
	if !got[0].ModifiedAt.Equal(want) {
		t.Fatalf("ModifiedAt = %s, want updatedAt %s", got[0].ModifiedAt, want)
	}
}
