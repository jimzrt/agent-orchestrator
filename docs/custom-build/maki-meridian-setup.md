# Maki with Meridian as a Claude proxy (macOS)

This guide sets up [Maki](https://maki.sh) and
[Meridian](https://github.com/rynfar/meridian) on a Mac so the custom AO build
can run Maki workers on a Claude subscription. It assumes Claude Code
(`claude`) and Codex (`codex`) are already installed and logged in.

Work through the steps in order. Run every **Check** and fix a failure before
moving on. Steps marked **You** need a browser or a decision and can't be
scripted. Nothing here needs `sudo`; everything installs per user.

Tested versions:

- Maki **v0.6.5**
- Meridian **@rynfar/meridian@1.85.0**

For Linux, see [Linux](#linux) at the end. For the harness itself (binary
lookup, auth probe, activity hooks, Chat mode), see
[docs/harnesses/maki.md](../harnesses/maki.md).

## Background

- **Maki** ([maki.sh](https://maki.sh),
  [tontinton/maki](https://github.com/tontinton/maki)) is a Rust terminal
  coding agent. Config lives in `~/.config/maki/`, data in
  `~/.local/share/maki/`, state in `~/.local/state/maki/`, logs in
  `~/.local/logs/maki/`. macOS uses the same XDG paths.
- **Meridian** runs Claude requests through Anthropic's official Agent SDK with
  your `claude login`, and serves an Anthropic-compatible API on
  `127.0.0.1:3456`. In **passthrough mode** tool calls go back to the client,
  so Maki keeps its own tools.
- **Terms:** using a Claude subscription through Meridian is your decision;
  check Anthropic's current terms. Don't try to work around an Anthropic
  block. If Meridian's requests are refused, stop.
- **Shared limits:** your Claude plan limits are shared by Claude Code, AO's
  Claude workers and Maki through Meridian.
- **AO's side:** the custom build has a `maki` harness. Its adapter installs a
  Lua activity package on first launch (step 7); you don't write it.

## 1. Prerequisites

```sh
sw_vers -productVersion
command -v claude && claude --version
command -v codex && codex --version
command -v node && node --version   # need >= 22
command -v npm
command -v curl git
```

- Node missing or older than 22: `brew install node` if you use Homebrew,
  otherwise install Node 22+ your preferred way (**You**).
- Note the absolute paths. launchd doesn't load the shell `PATH`, so step 5
  needs them:

  ```sh
  NODE_BIN="$(command -v node)"; CLAUDE_BIN="$(command -v claude)"
  echo "$NODE_BIN $CLAUDE_BIN"
  ```

**Check:** `claude auth status` (or `claude -p "reply: ok"`) works. If Claude
isn't logged in, run `claude login` (**You**).

## 2. Install Maki v0.6.5

The official installer downloads the GitHub release into `~/.local/bin` and
takes the tag as an argument. It's short; read it before running it:

```sh
curl -fsSL https://maki.sh/install.sh -o /tmp/maki-install.sh
less /tmp/maki-install.sh
sh /tmp/maki-install.sh v0.6.5
```

Put `~/.local/bin` on `PATH` for future shells (zsh is the macOS default):

```sh
grep -q 'HOME/.local/bin' ~/.zshrc 2>/dev/null || echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

**Check:** `maki --version` prints `maki 0.6.5`.

- Later updates: `maki update` asks `[y/N]` on a terminal; non-interactively
  use `echo y | maki update`. `maki rollback` restores the previous binary.
- If Gatekeeper blocks the binary, `xattr -d com.apple.quarantine ~/.local/bin/maki`
  clears the quarantine flag (**You**: decide whether to trust it).

## 3. OpenAI in Maki (optional)

Maki can use the same ChatGPT subscription as Codex through its own OAuth
login. It doesn't read Codex's credentials:

```sh
maki auth status | grep -i openai
```

If it isn't `✓`, run `maki auth login openai` and sign in with your ChatGPT
account in the browser (**You**).

**Check:** `maki auth status` shows `✓ openai`. A ChatGPT login allows
`openai/gpt-5.6-*`, `openai/gpt-6-*`, `openai/gpt-6.1-*` and the codex models.
API-only models such as `gpt-5.5` are rejected ("not supported when using
Codex with a ChatGPT account").

## 4. Install and test Meridian

```sh
npm install -g @rynfar/meridian@1.85.0
command -v meridian && meridian --version
```

npm may report skipped install scripts for `@anthropic-ai/claude-code`. That's
fine: Meridian uses the `claude` it finds on `PATH`.

Quick foreground test (stop it with Ctrl-C afterwards):

```sh
MERIDIAN_PASSTHROUGH=1 MERIDIAN_DEFAULT_AGENT=passthrough meridian
# in another shell:
curl -s 127.0.0.1:3456/health
```

**Check:** the `/health` JSON contains `"status":"healthy"`,
`"auth":{"loggedIn":true,...}`, `"mode":"passthrough"` and
`"claudeExecutable":{"path": ...}`. On macOS Meridian reads the Claude login
from the Keychain. If `loggedIn` is false, try `meridian refresh-token`, then
`claude login` (**You**).

Meridian also has an optional Mac desktop app that manages the service. This
guide uses launchd instead; install the app only if you prefer it.

## 5. Start Meridian at login (launchd)

Stop the foreground test first. Then write the LaunchAgent with absolute paths.
launchd starts with a minimal `PATH`, so include the directories of `node` and
`claude`:

```sh
NODE_BIN="$(command -v node)"; CLAUDE_BIN="$(command -v claude)"
MERIDIAN_BIN="$(command -v meridian)"
EXTRA_PATH="$(dirname "$NODE_BIN"):$(dirname "$CLAUDE_BIN"):$HOME/.local/bin:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin"
mkdir -p ~/Library/LaunchAgents ~/Library/Logs
cat > ~/Library/LaunchAgents/dev.meridian.bridge.plist <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>dev.meridian.bridge</string>
  <key>ProgramArguments</key>
  <array>
    <string>$NODE_BIN</string>
    <string>$MERIDIAN_BIN</string>
  </array>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key><string>$EXTRA_PATH</string>
    <key>HOME</key><string>$HOME</string>
    <key>MERIDIAN_PASSTHROUGH</key><string>1</string>
    <key>MERIDIAN_DEFAULT_AGENT</key><string>passthrough</string>
  </dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
  <key>StandardOutPath</key><string>$HOME/Library/Logs/meridian.log</string>
  <key>StandardErrorPath</key><string>$HOME/Library/Logs/meridian.log</string>
</dict>
</plist>
EOF
plutil -lint ~/Library/LaunchAgents/dev.meridian.bridge.plist
launchctl bootout gui/$(id -u)/dev.meridian.bridge 2>/dev/null || true
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/dev.meridian.bridge.plist
sleep 6; curl -s 127.0.0.1:3456/health
```

`KeepAlive` uses `SuccessfulExit=false` on purpose. If another Meridian already
holds the port, the new one prints its status and exits with code 0. Find and
stop the other instance (`lsof -nP -iTCP:3456 -sTCP:LISTEN`), then restart the
job with `launchctl kickstart -k gui/$(id -u)/dev.meridian.bridge`.

**Check:** `/health` is healthy with `loggedIn:true` and `mode:passthrough`;
`launchctl print gui/$(id -u)/dev.meridian.bridge | grep state` shows
`running`; `~/Library/Logs/meridian.log` shows
`Meridian running at http://127.0.0.1:3456`.

If your Node path changes later (nvm or Homebrew upgrade), redo this step.

Optional hardening: Meridian has no auth by default, so any local process can
use your Claude plan through it. To require a key, add `MERIDIAN_API_KEY` to
the plist's `EnvironmentVariables` and put the same value in `api_key` in
step 6 (**You**: decide whether you want this).

## 6. Maki provider for Meridian

Create `~/.config/maki/providers.toml`. If the file already exists, merge in the
`[meridian]` table and keep your other entries. Since Maki 0.6.5 **unknown keys
stop Maki from starting**, so use exactly these keys:

```sh
mkdir -p ~/.config/maki
cat > ~/.config/maki/providers.toml <<'EOF'
# Claude via the local Meridian bridge (Claude subscription through the
# official Agent SDK), on 127.0.0.1:3456 in passthrough mode.
[meridian]
display_name = "Claude (Meridian)"
protocol = "anthropic"
base_url = "http://127.0.0.1:3456"
api_key = "x"
default_model = "meridian/claude-sonnet-5-5"
discover_models = true

[[meridian.models]]
id = "claude-opus-5-5"
tier = "strong"
context_window = 200000
max_output_tokens = 32000
supports_thinking = true

[[meridian.models]]
id = "claude-sonnet-5-5"
tier = "medium"
context_window = 200000
max_output_tokens = 32000
supports_thinking = true

[[meridian.models]]
id = "claude-haiku-5-5"
tier = "weak"
context_window = 200000
max_output_tokens = 16000
supports_thinking = true
EOF
```

**Check:**

```sh
maki models | grep meridian          # lists claude-* (declared + discovered; duplicates are harmless)
cd "$(mktemp -d)" && printf 'def add(a,b):\n    return a-b\n' > m.py
maki -p -m meridian/claude-sonnet-5-5 --allowed-tools read,edit,index "Fix the bug in m.py. Reply briefly."
cat m.py                              # must now contain a+b
```

That run exercises the whole path: Maki's own `read` and `edit` tools, called by
Claude through Meridian. `maki auth status` shows `✗ meridian` even though it
works, because the status command doesn't count an `api_key` set in
`providers.toml`.

Don't add `--yolo` to test runs unless you mean to skip Maki's permission
prompts.

## 7. AO integration

The custom AO build includes the `maki` harness, so nothing is installed by
hand here:

- **Binary:** AO finds `maki` on `PATH` or at `~/.local/bin/maki`,
  `~/.cargo/bin/maki`, `/usr/local/bin/maki` or `/opt/homebrew/bin/maki`. GUI
  apps don't get the shell `PATH`, but `~/.local/bin/maki` is in AO's fallback
  list.
- **Activity package:** on the first Maki session AO writes
  `~/.local/share/maki/site/pack/ao/start/ao-activity/{plugin.toml,plugin/ao_activity.lua}`.
  The package does nothing unless `AO_SESSION_ID` is set. It reports turn and
  permission changes through `ao hooks maki <event>` and requests only the
  `run` and `env` permissions. If a non-AO file already sits at that path, AO
  refuses to overwrite it; move it aside yourself (**You**).
- **Worker model:** in AO project settings, choose the **Maki** agent with model
  `meridian/claude-sonnet-5-5`, or an `openai/...` model from step 3.

**Check** (after installing and opening the custom AO build):

```sh
ao doctor | grep -i maki             # maki binary found
```

Then spawn a Maki worker (`ao spawn --agent maki`, or the UI) with a task that
needs one shell command. AO's status should go active → waiting for input (the
permission prompt) → active → idle. Afterwards the package files above exist.

## Troubleshooting

| Symptom | Fix |
|---|---|
| Maki exits at start mentioning `providers.toml` | A typo or unknown key (0.6.5 is strict); compare with step 6. |
| `maki -p ... meridian/...` hangs or gets connection refused | Meridian is down: check `launchctl print gui/$(id -u)/dev.meridian.bridge` and `~/Library/Logs/meridian.log`. |
| `/health` shows `loggedIn:false` | Run `meridian refresh-token`; otherwise `claude login` (**You**). |
| Meridian rate-limit errors | The Claude plan quota is used up (shared with Claude Code). Wait, or use `openai/...` models. |
| OpenAI "rate limited" or "usage limit" | The ChatGPT quota is used up (shared with Codex). |
| AO shows the Maki worker but its status never changes | The activity package is missing or `ao` isn't on the worker's `PATH`; check the step 7 paths and `~/.local/logs/maki/maki.log`. |
| A new Maki version breaks something | `maki rollback`. |

## Linux

The same setup works on Linux with a systemd user service in place of launchd:

- Install Meridian with `npm i -g @rynfar/meridian@1.85.0` and run it as a
  systemd user service, `~/.config/systemd/user/meridian.service`, with
  `MERIDIAN_PASSTHROUGH=1` and `MERIDIAN_DEFAULT_AGENT=passthrough`, listening
  on `127.0.0.1:3456` and enabled to start at login. Use absolute paths to
  `node` and `meridian` in `ExecStart`; with nvm they change when the Node
  version does.
- `~/.config/maki/providers.toml` is the same as in step 6: the `meridian`
  provider (anthropic protocol) with `claude-opus-5-5` (strong),
  `claude-sonnet-5-5` (medium, default) and `claude-haiku-5-5` (weak).
- Maki's OpenAI login (step 3) and AO's activity package (step 7) work the same
  way.
