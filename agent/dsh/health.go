package dsh

import (
	"context"
	"strings"

	"github.com/chenhg5/cc-connect/core"
)

func reportDSHResult(err error) {
	event := core.HealthEvent{Component: "agent", State: "available"}
	if err != nil {
		event.State = "unavailable"
		event.ErrorCode = "acp"
		// This is the DSH credentials-service error, not a robot credential error.
		// Only the category is emitted; provider route/key/error text stays private.
		if strings.Contains(err.Error(), "no API key for provider route") {
			event.ErrorCode = "model-key-missing"
		}
	}
	core.ReportHealth(event)
}

func (a *Agent) StartSession(ctx context.Context, sessionID string) (core.AgentSession, error) {
	session, err := a.Agent.StartSession(ctx, sessionID)
	if err != nil {
		reportDSHResult(err)
		return nil, err
	}
	// A handshake alone does not prove that a model route can run a turn.
	return session, nil
}
