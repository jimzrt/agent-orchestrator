package maki

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
	aoprocess "github.com/aoagents/agent-orchestrator/backend/internal/process"
)

var _ ports.AgentAuthChecker = (*Plugin)(nil)

var makiTerminalEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// AuthStatus reports configured when `maki auth status` lists at least one
// provider with stored credentials. Maki can also read provider keys from the
// environment, so an empty listing is unknown rather than unauthorized.
func (p *Plugin) AuthStatus(ctx context.Context) (ports.AgentAuthStatus, error) {
	binary, err := p.ResolveBinary(ctx)
	if err != nil {
		return ports.AgentAuthStatusUnknown, err
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := aoprocess.CommandContext(probeCtx, binary, "auth", "status").CombinedOutput() //nolint:gosec // binary is adapter-resolved, args are static.
	if err != nil {
		return ports.AgentAuthStatusUnknown, nil
	}
	return parseAuthStatus(string(out)), nil
}

func parseAuthStatus(output string) ports.AgentAuthStatus {
	plain := makiTerminalEscape.ReplaceAllString(output, "")
	for _, line := range strings.Split(plain, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "✓") {
			return ports.AgentAuthStatusConfigured
		}
	}
	return ports.AgentAuthStatusUnknown
}
