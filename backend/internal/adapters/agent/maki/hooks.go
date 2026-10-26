package maki

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aoagents/agent-orchestrator/backend/internal/adapters/agent/hookutil"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// makiPackageSentinel marks files AO owns, so install never overwrites a
// package the user placed under the same name.
const makiPackageSentinel = "agent-orchestrator: managed maki activity package"

//go:embed assets/ao_activity.lua
var activityPluginSource []byte

//go:embed assets/plugin.toml
var activityPluginManifest []byte

// GetAgentHooks installs AO's Maki activity package. Maki loads Lua packages
// from its data dir rather than the workspace, so this is one machine-wide
// install. The package stays inert unless AO_SESSION_ID is set, which keeps
// it from touching non-AO sessions and avoids project folder trust: a
// workspace .maki/init.lua would need --trust, which would also load whatever
// the repository ships in .maki/.
func (p *Plugin) GetAgentHooks(ctx context.Context, cfg ports.WorkspaceHookConfig) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dir, err := activityPackageDir(cfg.Env)
	if err != nil {
		return fmt.Errorf("maki.GetAgentHooks: %w", err)
	}
	files := []struct {
		path string
		data []byte
	}{
		{filepath.Join(dir, "plugin.toml"), activityPluginManifest},
		{filepath.Join(dir, "plugin", "ao_activity.lua"), activityPluginSource},
	}
	for _, f := range files {
		if err := installManagedFile(f.path, f.data); err != nil {
			return fmt.Errorf("maki.GetAgentHooks: %w", err)
		}
	}
	return nil
}

func installManagedFile(path string, data []byte) error {
	existing, err := os.ReadFile(path) //nolint:gosec // AO-owned package path
	switch {
	case err == nil:
		if !strings.Contains(string(existing), makiPackageSentinel) {
			return fmt.Errorf("refusing to overwrite non-AO file at %s", path)
		}
		if bytes.Equal(existing, data) {
			return nil
		}
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	return hookutil.AtomicWriteFile(path, data, 0o600)
}

// activityPackageDir is <maki-data>/site/pack/ao/start/ao-activity, where
// Maki loads hand-installed packages at startup.
func activityPackageDir(env map[string]string) (string, error) {
	data, err := makiDataDir(env)
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "site", "pack", "ao", "start", "ao-activity"), nil
}

// makiDataDir mirrors Maki's directory layout: XDG data on Linux and macOS,
// Roaming AppData on Windows.
func makiDataDir(env map[string]string) (string, error) {
	lookup := func(key string) string {
		if v := strings.TrimSpace(env[key]); v != "" {
			return v
		}
		return strings.TrimSpace(os.Getenv(key))
	}
	if runtime.GOOS == "windows" {
		if appData := lookup("APPDATA"); appData != "" {
			return filepath.Join(appData, "maki"), nil
		}
		return "", errors.New("APPDATA is not set")
	}
	if xdg := lookup("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "maki"), nil
	}
	home := lookup("HOME")
	if home == "" {
		var err error
		if home, err = os.UserHomeDir(); err != nil {
			return "", err
		}
	}
	return filepath.Join(home, ".local", "share", "maki"), nil
}
