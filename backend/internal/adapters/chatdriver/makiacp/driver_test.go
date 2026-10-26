package makiacp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	acpdriver "github.com/aoagents/agent-orchestrator/backend/internal/adapters/chatdriver/acp"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestConfigure(t *testing.T) {
	got, env, err := configure(context.Background(), acpdriver.LaunchConfig{
		Model: "openai/gpt-5.6-luna", Permissions: ports.PermissionModeBypassPermissions,
	})
	if err != nil || env != nil {
		t.Fatalf("configure: %v %v", env, err)
	}
	want := []string{"acp", "--model", "openai/gpt-5.6-luna", "--yolo"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	got, _, _ = configure(context.Background(), acpdriver.LaunchConfig{Permissions: ports.PermissionModeAcceptEdits})
	if !reflect.DeepEqual(got, []string{"acp"}) {
		t.Fatalf("non-bypass got %#v", got)
	}
}

func TestSessionPreambleCarriesSystemPrompt(t *testing.T) {
	if got := sessionPreamble(acpdriver.LaunchConfig{}); got != nil {
		t.Fatalf("empty = %#v", got)
	}
	if got := sessionPreamble(acpdriver.LaunchConfig{SystemPrompt: "AO rules"}); len(got) != 1 {
		t.Fatalf("got %#v", got)
	}
}

func TestValidateTurnSettingsRejectsYoloChange(t *testing.T) {
	if err := validateTurnSettings(ports.PermissionModeDefault, ports.ChatTurnSettings{Approval: ports.PermissionModeAcceptEdits}); err != nil {
		t.Fatalf("non-yolo change: %v", err)
	}
	err := validateTurnSettings(ports.PermissionModeDefault, ports.ChatTurnSettings{Approval: ports.PermissionModeBypassPermissions})
	if !errors.Is(err, acpdriver.ErrACPSetterUnsupported) {
		t.Fatalf("yolo change err = %v", err)
	}
}
