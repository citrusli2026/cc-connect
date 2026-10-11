package core

import (
	"encoding/json"
	"log/slog"
	"os"
)

// HealthEvent is the versioned, content-free Desktop sidecar contract. It is
// deliberately not a log/error payload: no URLs, identities, prompts or secrets.
type HealthEvent struct {
	Version   int    `json:"version"`
	Component string `json:"component"`
	State     string `json:"state"`
	ErrorCode string `json:"errorCode,omitempty"`
}

// ReportHealth is opt-in and does not change the default CLI log protocol.
// A platform/agent adapter reports only facts it owns, never inferred readiness.
func ReportHealth(event HealthEvent) bool {
	if os.Getenv("DSH_CC_CONNECT_HEALTH") != "1" {
		return false
	}
	switch event.Component {
	case "channel":
		if event.State != "connecting" && event.State != "online" && event.State != "offline" {
			return false
		}
	case "agent":
		if event.State != "available" && event.State != "unavailable" {
			return false
		}
	default:
		return false
	}
	switch event.ErrorCode {
	case "", "credentials", "permissions", "network", "model-key-missing", "acp":
	default:
		return false
	}
	event.Version = 1
	payload, err := json.Marshal(event)
	if err != nil {
		return false
	}
	slog.Info("cc-connect health", "event", string(payload))
	return true
}
