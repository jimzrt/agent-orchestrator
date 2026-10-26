# Maki Adapter

[Maki](https://maki.sh) is a Rust terminal coding agent. AO integrates it as an
interactive Terminal UI harness (`maki`) and as a Chat harness over Maki's
native ACP server (`maki acp`). AO launches the user's own `maki` binary inside
the session worktree.

## Install

Install Maki with its official installer, which places the binary in
`~/.local/bin` (override with `MAKI_INSTALL_DIR`):

```bash
curl -fsSL https://maki.sh/install.sh | sh
```

On Windows, `irm https://maki.sh/install.ps1 | iex` installs to
`%LOCALAPPDATA%\maki`. AO's one-click installer runs the same scripts, and
updates go through `maki update --yes`. AO compares the installed version with
the latest `tontinton/maki` GitHub release.

AO resolves the binary from:

- `PATH`: `maki` (`maki.exe` on Windows)
- `/usr/local/bin/maki`, `/opt/homebrew/bin/maki`
- `~/.local/bin/maki`, `~/.cargo/bin/maki`
- `%LOCALAPPDATA%\maki\maki.exe`, `~/.local/bin/maki.exe` on Windows

## Auth

Maki keeps provider credentials itself. Settings → Harness offers **Open setup**,
which runs `maki auth login` (OAuth or API key per provider) in a terminal.

AO probes `maki auth status`. A provider marked `✓` counts as configured
(not verified: a later model call can still fail on quota or plan). With no
marked provider the status stays unknown rather than unauthorized, because
Maki also reads provider keys from the environment and from custom providers
in `~/.config/maki/providers.toml`, which `auth status` does not report as
signed in.

## Models

The model picker lists the output of `maki models`, one `provider/model` spec
per line (for example `openai/gpt-5.6-luna`). Custom entry is allowed: AO passes
the value unchanged to `--model`.

## Terminal UI

A fresh session launches as:

```bash
maki [--append-system-prompt <standing instructions>] [--model <provider/model>] [permission flags] [-- task]
```

Maki takes the first task as a positional prompt and keeps the TUI open. The
`--` keeps a task that starts with `-` or matches a subcommand name from being
parsed as an option.

| AO permission mode | Maki flags |
| --- | --- |
| default, accept-edits, auto | none (Maki's own permission prompts and `permissions.toml`) |
| bypass-permissions | `--yolo` |

Outside bypass mode, AO's allowed and disallowed tool lists are forwarded as
`--allowed-tools` and `--disallowed-tools`.

### Activity tracking

Maki loads Lua packages from its data directory, not from the workspace. AO
installs a small package at
`<maki-data>/site/pack/ao/start/ao-activity/` (`~/.local/share/maki/...` or
`$XDG_DATA_HOME/maki/...` on Linux and macOS, `%APPDATA%\maki\...` on Windows).
AO never overwrites a file there that lacks its managed-file marker.

The package is inert unless `AO_SESSION_ID` is set, so it does not affect Maki
sessions started outside AO. It needs Maki's `run` and `env` plugin permissions
and reports through `ao hooks maki <event>` with Maki's native session id on
stdin:

| Maki event | AO event | Activity |
| --- | --- | --- |
| `TurnStart` | `user-prompt-submit` | active |
| `SessionStatusChanged` to `needs_input` | `permission-request` | waiting for input |
| `needs_input` back to `working` | `permission-resolved` | active |
| `TurnEnd`, `TurnError` | `stop` | idle |
| `SessionEnd` with reason `shutdown` | `session-end` | exited |

A workspace `.maki/init.lua` was not used: Maki asks before loading shared
project configuration, and `--trust` would also load anything else the
repository ships in `.maki/`.

### Restore

AO restores with `maki --session <id>` once the activity package has reported
the native session id. Before the first report it falls back to
`maki --continue`, the most recent session in the worktree, which is the one AO
launched because each AO session owns its worktree.

## Chat

AO launches `maki acp [--model <provider/model>] [--yolo]` and speaks the Agent
Client Protocol over stdio. A handshake against Maki 0.5.5 on 2026-10-26
reported:

```text
protocolVersion   1
loadSession       true
promptCapabilities{image: true, embeddedContext: true}
mcpCapabilities   {http: true}
modes             build, plan
configOptions     model (select, provider/model specs)
```

The Chat model picker uses the `model` config option. `maki acp` has no
system-prompt flag, so standing instructions travel as an embedded resource in
the first prompt. `--yolo` is fixed when the ACP process starts, so changing
between bypass and a prompting mode mid-chat is refused; restart the Chat to
change it.

The Chat binding lives in `backend/internal/adapters/chatdriver/makiacp/`.

## Limits

- `user-prompt-submit` carries no prompt text: the Maki TUI does not expose it
  to plugins.
- Maki fires no turn events under ACP, so Chat sessions get no hook activity;
  Chat status comes from the ACP stream like other Chat drivers.
- Waiting-for-input follows only the focused Maki tab. AO workers use one tab.
- The activity package is installed once per machine, not per workspace.
- The legacy `~/.maki/` layout is not handled; only XDG paths and `%APPDATA%`.
- Unattended harness updates while sessions run are off
  (`SupportsTerminalVersionRefresh` is false).
- Maki is not offered as a Chat or Terminal switch target until a real switch
  has been verified.
