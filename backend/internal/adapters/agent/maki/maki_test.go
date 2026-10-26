package maki

import (
	"context"
	"reflect"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

func stubbed() *Plugin {
	return &Plugin{resolvedBinary: "/bin/maki"}
}

func TestGetLaunchCommand(t *testing.T) {
	tests := []struct {
		name string
		cfg  ports.LaunchConfig
		want []string
	}{
		{name: "bare", want: []string{"/bin/maki"}},
		{
			name: "prompt model system prompt bypass",
			cfg: ports.LaunchConfig{
				Prompt: "fix it", SystemPrompt: "AO rules",
				Config:       ports.AgentConfig{Model: "openai/gpt-5.6-luna"},
				Permissions:  ports.PermissionModeBypassPermissions,
				AllowedTools: []string{"Read"},
			},
			want: []string{"/bin/maki", "--append-system-prompt", "AO rules", "--model", "openai/gpt-5.6-luna", "--yolo", "--", "fix it"},
		},
		{
			name: "flag-like task stays a prompt",
			cfg:  ports.LaunchConfig{Prompt: "--help me fix the update command"},
			want: []string{"/bin/maki", "--", "--help me fix the update command"},
		},
		{
			name: "restricted tools",
			cfg: ports.LaunchConfig{
				AllowedTools: []string{"Read", "Grep"}, DisallowedTools: []string{"Edit"},
			},
			want: []string{"/bin/maki", "--allowed-tools", "Read,Grep", "--disallowed-tools", "Edit"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stubbed().GetLaunchCommand(context.Background(), tt.cfg)
			if err != nil {
				t.Fatalf("GetLaunchCommand: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGetRestoreCommandContinuesWorkspaceSession(t *testing.T) {
	got, ok, err := stubbed().GetRestoreCommand(context.Background(), ports.RestoreConfig{
		Config: ports.AgentConfig{Model: "openai/gpt-5.6-luna"},
		Prompt: "next",
	})
	if err != nil || !ok {
		t.Fatalf("GetRestoreCommand: ok=%v err=%v", ok, err)
	}
	want := []string{"/bin/maki", "--model", "openai/gpt-5.6-luna", "--continue", "--", "next"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestGetRestoreCommandUsesReportedSession(t *testing.T) {
	got, ok, err := stubbed().GetRestoreCommand(context.Background(), ports.RestoreConfig{
		Session: ports.SessionRef{Metadata: map[string]string{ports.MetadataKeyAgentSessionID: "CgQtCe1Q"}},
	})
	if err != nil || !ok {
		t.Fatalf("GetRestoreCommand: ok=%v err=%v", ok, err)
	}
	want := []string{"/bin/maki", "--session", "CgQtCe1Q"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestParseAuthStatus(t *testing.T) {
	out := "  \x1b[31m✗\x1b[0m google  Google\n  \x1b[32m✓\x1b[0m openai  OpenAI (oauth)\n"
	if got := parseAuthStatus(out); got != ports.AgentAuthStatusConfigured {
		t.Fatalf("got %q", got)
	}
	if got := parseAuthStatus("  ✗ google  Google\n"); got != ports.AgentAuthStatusUnknown {
		t.Fatalf("got %q", got)
	}
}
