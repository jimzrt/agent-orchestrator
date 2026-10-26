package maki

import "github.com/aoagents/agent-orchestrator/backend/internal/domain"

// DeriveActivityState maps the events AO's Maki activity package reports
// (see assets/ao_activity.lua) onto AO activity states.
func DeriveActivityState(event string, _ []byte) (domain.ActivityState, bool) {
	switch event {
	case "user-prompt-submit", "permission-resolved":
		return domain.ActivityActive, true
	case "permission-request":
		return domain.ActivityWaitingInput, true
	case "stop":
		return domain.ActivityIdle, true
	case "session-end":
		return domain.ActivityExited, true
	default:
		return "", false
	}
}
