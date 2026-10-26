// Package maki implements the Maki agent adapter: launching interactive Maki
// TUI sessions in a worktree and resuming them.
//
// Maki (https://maki.sh) is a Rust TUI agent whose CLI mirrors Claude Code's
// flag surface (--append-system-prompt, --allowed-tools, --yolo). Activity
// comes from a Lua package AO installs into Maki (see hooks.go), which reports
// turn and status changes through `ao hooks maki <event>`.
package maki

import (
	"context"
	"os"
	"strings"
	"sync"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/agentbase"
	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/binaryutil"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// adapterID is the registry id and the value users pass to `ao spawn --agent`.
// It matches domain.HarnessMaki.
const adapterID = "maki"

// Plugin is the Maki agent adapter. It is safe for concurrent use; the binary
// path is resolved once and cached under binaryMu.
type Plugin struct {
	agentbase.Base
	binaryMu       sync.Mutex
	resolvedBinary string
}

// New returns a ready-to-register Maki adapter.
func New() *Plugin {
	return &Plugin{}
}

var _ adapters.Adapter = (*Plugin)(nil)
var _ ports.Agent = (*Plugin)(nil)
var _ ports.AgentBinaryResolver = (*Plugin)(nil)

// Manifest returns the adapter's static self-description.
func (p *Plugin) Manifest() adapters.Manifest {
	return adapters.Manifest{
		ID:          adapterID,
		Name:        "Maki",
		Description: "Run Maki interactive TUI sessions.",
		Version:     "0.0.1",
		Capabilities: []adapters.Capability{
			adapters.CapabilityAgent,
		},
	}
}

// GetConfigSpec reports the model override. Maki model specs carry their
// provider ("<provider>/<model-id>", e.g. "openai/gpt-5.6-luna").
func (p *Plugin) GetConfigSpec(ctx context.Context) (ports.ConfigSpec, error) {
	return agentbase.ModelConfigSpec(ctx, "Model override passed to `maki --model` as \"<provider>/<model-id>\" (e.g. \"openai/gpt-5.6-luna\").")
}

// GetLaunchCommand builds the argv to start an interactive Maki session:
//
//	maki [--append-system-prompt <p>] [--model <m>] [permission flags] [-- prompt]
//
// The runtime sets cwd to the worktree. Maki takes the first task as a
// positional prompt and keeps the TUI open afterwards.
func (p *Plugin) GetLaunchCommand(ctx context.Context, cfg ports.LaunchConfig) ([]string, error) {
	binary, err := p.makiBinary(ctx)
	if err != nil {
		return nil, err
	}
	cmd := []string{binary}
	if err := appendSystemPrompt(&cmd, cfg.SystemPrompt, cfg.SystemPromptFile); err != nil {
		return nil, err
	}
	agentbase.AppendModelFlag(&cmd, cfg.Config, "--model")
	appendPermissionFlags(&cmd, cfg.Permissions, cfg.AllowedTools, cfg.DisallowedTools)
	appendPrompt(&cmd, cfg.Prompt)
	return cmd, nil
}

// GetRestoreCommand resumes the native session the activity package reported
// (`maki --session <id>`). Before any report arrives it falls back to
// `maki --continue`, the most recent session in the working directory, which
// is the one AO launched because each AO session owns its worktree.
func (p *Plugin) GetRestoreCommand(ctx context.Context, cfg ports.RestoreConfig) ([]string, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	binary, err := p.makiBinary(ctx)
	if err != nil {
		return nil, false, err
	}
	cmd := []string{binary}
	if err := appendSystemPrompt(&cmd, cfg.SystemPrompt, cfg.SystemPromptFile); err != nil {
		return nil, false, err
	}
	agentbase.AppendModelFlag(&cmd, cfg.Config, "--model")
	appendPermissionFlags(&cmd, cfg.Permissions, cfg.AllowedTools, cfg.DisallowedTools)
	if id := strings.TrimSpace(cfg.Session.Metadata[ports.MetadataKeyAgentSessionID]); id != "" {
		cmd = append(cmd, "--session", id)
	} else {
		cmd = append(cmd, "--continue")
	}
	appendPrompt(&cmd, cfg.Prompt)
	return cmd, true, nil
}

// SessionInfo reports the title and summary AO has recorded for the session.
func (p *Plugin) SessionInfo(ctx context.Context, session ports.SessionRef) (ports.SessionInfo, bool, error) {
	if err := ctx.Err(); err != nil {
		return ports.SessionInfo{}, false, err
	}
	info, ok := agentbase.StandardSessionInfo(session)
	return info, ok, nil
}

// appendPermissionFlags maps AO permission modes onto Maki's flags. Bypass
// skips every prompt; tool lists only apply outside bypass, matching how the
// reviewer relies on them as an enforced restriction.
func appendPermissionFlags(cmd *[]string, mode ports.PermissionMode, allowed, disallowed []string) {
	if ports.NormalizePermissionMode(mode) == ports.PermissionModeBypassPermissions {
		*cmd = append(*cmd, "--yolo")
		return
	}
	if len(allowed) > 0 {
		*cmd = append(*cmd, "--allowed-tools", strings.Join(allowed, ","))
	}
	if len(disallowed) > 0 {
		*cmd = append(*cmd, "--disallowed-tools", strings.Join(disallowed, ","))
	}
}

// appendPrompt ends option parsing before the task, so a task that starts
// with "-" or equals a subcommand name ("update", "auth") stays a prompt.
func appendPrompt(cmd *[]string, prompt string) {
	if prompt = strings.TrimSpace(prompt); prompt != "" {
		*cmd = append(*cmd, "--", prompt)
	}
}

func appendSystemPrompt(cmd *[]string, inline, file string) error {
	if inline != "" {
		*cmd = append(*cmd, "--append-system-prompt", inline)
		return nil
	}
	if file == "" {
		return nil
	}
	data, err := os.ReadFile(file) //nolint:gosec // path is AO-owned launch config
	if err != nil {
		return err
	}
	*cmd = append(*cmd, "--append-system-prompt", string(data))
	return nil
}

var makiBinarySpec = binaryutil.BinarySpec{
	Label:         "maki",
	Names:         []string{"maki"},
	WinNames:      []string{"maki.exe", "maki"},
	UnixPaths:     []string{"/usr/local/bin/maki", "/opt/homebrew/bin/maki"},
	UnixHomePaths: [][]string{{".local", "bin", "maki"}, {".cargo", "bin", "maki"}},
	WinPaths: []binaryutil.WinPath{
		{Base: binaryutil.WinLocalAppData, Parts: []string{"maki", "maki.exe"}},
		{Base: binaryutil.WinHome, Parts: []string{".local", "bin", "maki.exe"}},
	},
}

// ResolveMakiBinary returns the path to the maki binary, searching PATH then
// the install script's default locations. It returns a wrapped
// ports.ErrAgentBinaryNotFound when maki is absent.
func ResolveMakiBinary(ctx context.Context) (string, error) {
	return binaryutil.ResolveBinary(ctx, makiBinarySpec)
}

// ResolveBinary resolves the executable path for the plugin.
func (p *Plugin) ResolveBinary(ctx context.Context) (string, error) {
	return p.makiBinary(ctx)
}

func (p *Plugin) makiBinary(ctx context.Context) (string, error) {
	p.binaryMu.Lock()
	defer p.binaryMu.Unlock()

	if p.resolvedBinary != "" {
		return p.resolvedBinary, nil
	}
	binary, err := ResolveMakiBinary(ctx)
	if err != nil {
		return "", err
	}
	p.resolvedBinary = binary
	return binary, nil
}
