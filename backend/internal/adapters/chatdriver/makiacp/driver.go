// Package makiacp drives Maki Chat sessions over Maki's native ACP server
// (`maki acp`).
package makiacp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	acpsdk "github.com/coder/acp-go-sdk"

	acpdriver "github.com/aoagents/agent-orchestrator/backend/internal/adapters/chatdriver/acp"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/chatdriver/nativeacp"
	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// New returns the Maki ACP chat driver backed by the Maki agent plugin.
func New(plugin nativeacp.Plugin, log *slog.Logger) ports.ChatDriver {
	return nativeacp.New(plugin, nativeacp.Config{
		Harness: domain.HarnessMaki,
		Capabilities: ports.ChatCapabilities{
			ports.ChatCapabilityUsage: true,
			ports.ChatCapabilityDiffs: true,
			ports.ChatCapabilityPlans: true,
		},
		Configure:            configure,
		SessionOptions:       sessionOptions,
		SessionPreamble:      sessionPreamble,
		ValidateTurnSettings: validateTurnSettings,
	}, log)
}

// configure builds `maki acp [--model <m>] [--yolo]`. Maki's ACP server has no
// system-prompt flag, so standing instructions travel as a session preamble.
func configure(_ context.Context, cfg acpdriver.LaunchConfig) ([]string, map[string]string, error) {
	args := []string{"acp"}
	if model := strings.TrimSpace(cfg.Model); model != "" {
		args = append(args, "--model", model)
	}
	if ports.NormalizePermissionMode(cfg.Permissions) == ports.PermissionModeBypassPermissions {
		args = append(args, "--yolo")
	}
	return args, nil, nil
}

func sessionOptions(settings ports.ChatTurnSettings) []acpdriver.SessionOption {
	if model := strings.TrimSpace(settings.Model); model != "" {
		return []acpdriver.SessionOption{{ID: "model", Value: model}}
	}
	return nil
}

func sessionPreamble(cfg acpdriver.LaunchConfig) []acpsdk.ContentBlock {
	if standing := strings.TrimSpace(cfg.SystemPrompt); standing != "" {
		return []acpsdk.ContentBlock{acpdriver.StandingInstructionsBlock(standing)}
	}
	return nil
}

// validateTurnSettings rejects switching into or out of --yolo mid-session:
// Maki fixes it when the ACP process launches.
func validateTurnSettings(initial ports.PermissionMode, settings ports.ChatTurnSettings) error {
	if settings.Approval == "" || isYolo(settings.Approval) == isYolo(initial) {
		return nil
	}
	return fmt.Errorf("%w: Maki ACP --yolo is fixed at process launch; restart Chat to change approval mode",
		acpdriver.ErrACPSetterUnsupported)
}

func isYolo(mode ports.PermissionMode) bool {
	return ports.NormalizePermissionMode(mode) == ports.PermissionModeBypassPermissions
}
