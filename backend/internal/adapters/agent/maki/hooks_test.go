package maki

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func TestGetAgentHooksInstallsActivityPackage(t *testing.T) {
	data := t.TempDir()
	cfg := ports.WorkspaceHookConfig{Env: map[string]string{"XDG_DATA_HOME": data}}
	for i := 0; i < 2; i++ { // second run must be a no-op, not a refusal
		if err := stubbed().GetAgentHooks(context.Background(), cfg); err != nil {
			t.Fatalf("GetAgentHooks run %d: %v", i, err)
		}
	}
	dir := filepath.Join(data, "maki", "site", "pack", "ao", "start", "ao-activity")
	lua, err := os.ReadFile(filepath.Join(dir, "plugin", "ao_activity.lua"))
	if err != nil {
		t.Fatalf("read plugin: %v", err)
	}
	if !strings.Contains(string(lua), `ao hooks maki`) || !strings.Contains(string(lua), "AO_SESSION_ID") {
		t.Fatalf("plugin source missing hook call or session gate")
	}
	manifest, err := os.ReadFile(filepath.Join(dir, "plugin.toml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if !strings.Contains(string(manifest), "run = true") || !strings.Contains(string(manifest), "env = true") {
		t.Fatalf("manifest = %q", manifest)
	}
}

func TestGetAgentHooksRefusesForeignPackage(t *testing.T) {
	data := t.TempDir()
	path := filepath.Join(data, "maki", "site", "pack", "ao", "start", "ao-activity", "plugin.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("[permissions]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := stubbed().GetAgentHooks(context.Background(), ports.WorkspaceHookConfig{Env: map[string]string{"XDG_DATA_HOME": data}})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("err = %v", err)
	}
}

func TestDeriveActivityState(t *testing.T) {
	tests := map[string]domain.ActivityState{
		"user-prompt-submit":  domain.ActivityActive,
		"permission-resolved": domain.ActivityActive,
		"permission-request":  domain.ActivityWaitingInput,
		"stop":                domain.ActivityIdle,
		"session-end":         domain.ActivityExited,
	}
	for event, want := range tests {
		if got, ok := DeriveActivityState(event, nil); !ok || got != want {
			t.Errorf("%s = %q, %v; want %q", event, got, ok, want)
		}
	}
	if _, ok := DeriveActivityState("tool-start", nil); ok {
		t.Error("unknown event reported activity")
	}
}
