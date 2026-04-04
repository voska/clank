# clank

A tmux sidebar that shows your AI agent fleet at a glance. Sessions are
projects, windows are workstreams, and every window shows whether its agent is
running, waiting for input, or done.

Built for [Claude Code](https://code.claude.com) and [Factory Droid](https://factory.ai).

## How it works

Clank uses a three-tier detection stack, prioritized by reliability:

1. **JSONL transcript watching** — Claude Code writes structured transcripts to
   `~/.claude/projects/`. Clank watches these files via fsnotify for near-instant,
   definitive state detection. Zero configuration needed.
2. **Hook bridge** — optional bash script that receives Claude Code lifecycle
   events (prompt submitted, tool running, agent stopped) and pushes them to
   clank's Unix socket for instant state transitions and Ghostty notifications.
3. **Content scraping** — fallback for agents without transcripts (Droid, Aider,
   Codex). Captures tmux pane content and pattern-matches against known TUI
   indicators.

## Installation

### Homebrew

```bash
brew install voska/tap/clank
```

### From source

```bash
git clone https://github.com/voska/clank.git
cd clank
make build
```

### TPM (tmux plugin manager)

Add to `~/.config/tmux/tmux.conf`:

```tmux
set -g @plugin 'voska/clank'
set -g @clank-width "22"     # optional, default 30
set -g @clank-key "o"        # optional, default o
set -g @clank-focus-key "O"  # optional, default O
```

### Manual tmux configuration

Add to `~/.config/tmux/tmux.conf`:

```tmux
set -g @clank-width "22"
bind o run-shell "$HOME/Developer/clank/clank.tmux toggle"
bind O run-shell "$HOME/Developer/clank/clank.tmux focus"
```

Reload with `prefix + r`, then `prefix + o` to toggle the sidebar.

## Features

- **Per-window agent state** — running (spinner + timer), done (checkmark), waiting (needs permission)
- **JSONL transcript watching** — primary detection for Claude Code, event-driven via fsnotify
- **Persistent sidebar** — follows you across windows and sessions via tmux hooks
- **Mouse clickable** — click a window to switch to it, focus goes to the main pane
- **Unseen markers** — checkmark appears when an agent finishes; clears when you visit the window
- **State persistence** — timers and states survive clank restarts
- **Desktop notifications** — Ghostty (OSC 777), iTerm2 (OSC 9), Kitty (OSC 99), macOS native (osascript)
- **Multi-client aware** — auto-resizes when terminal dimensions change

## Navigation

| Key        | Action                           |
|------------|----------------------------------|
| `j` / `↓`  | Move down                       |
| `k` / `↑`  | Move up                         |
| `Enter`    | Switch to window                 |
| `1`-`9`    | Jump to session by index         |
| `f`        | Filter: show only agent sessions |
| `q`        | Hide sidebar                     |
| Mouse click | Switch to window                |

## Hook Bridge (optional)

The hook bridge adds instant state transitions and Ghostty desktop notifications.

```bash
cp agent-hook-bridge ~/.local/bin/
chmod +x ~/.local/bin/agent-hook-bridge
```

Add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [{"matcher": "", "hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge submit"}]}],
    "Notification": [{"matcher": "", "hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge notification"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge stop"}]}],
    "SessionStart": [{"hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge session-start"}]}],
    "SessionEnd": [{"hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge session-end"}]}],
    "SubagentStop": [{"hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge subagent-stop"}]}]
  }
}
```

## tmux Requirements

```tmux
set -g allow-passthrough all    # for Ghostty notifications over SSH
set -g monitor-bell on
set -g bell-action other
set -g mouse on                 # for mouse click support
```

## Architecture

```
Claude Code JSONL ──▶ fsnotify ──▶ transcript watcher ──▶ state store ──▶ TUI
Claude Code hooks ──▶ hook bridge ──▶ Unix socket ──────▶ state store
Droid/other agents ──▶ capture-pane ──▶ content scrape ──▶ state store
```

Single Go binary (~4MB). Bubbletea v1 + Lipgloss. Catppuccin Mocha palette.

## License

[MIT](LICENSE)
