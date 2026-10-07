// Package dsh integrates DeepSeek Harness as a first-class ACP agent.
package dsh

import (
	"strings"

	"github.com/chenhg5/cc-connect/agent/acp"
	"github.com/chenhg5/cc-connect/core"
)

func init() {
	core.RegisterAgent("dsh", New)
}

// Agent embeds the generic ACP implementation and supplies DSH-specific
// identity and defaults.
type Agent struct {
	*acp.Agent
}

// Name returns the stable agent type identifier used in configuration and
// session store keys.
func (a *Agent) Name() string { return "dsh" }

// New builds a DSH ACP agent. All generic ACP options are passed through
// unchanged after DSH defaults are applied.
func New(opts map[string]any) (core.Agent, error) {
	a, err := acp.New(applyDSHDefaults(opts))
	if err != nil {
		return nil, err
	}
	base, ok := a.(*acp.Agent)
	if !ok {
		return a, nil
	}
	return &Agent{Agent: base}, nil
}

func applyDSHDefaults(opts map[string]any) map[string]any {
	if opts == nil {
		opts = make(map[string]any)
	}
	if command, _ := opts["command"].(string); strings.TrimSpace(command) == "" {
		opts["command"] = "dsh"
	}
	if _, ok := opts["args"]; !ok {
		opts["args"] = []string{"--profile", "acp"}
	}
	if displayName, _ := opts["display_name"].(string); strings.TrimSpace(displayName) == "" {
		opts["display_name"] = "DeepSeek Harness"
	}
	return opts
}
