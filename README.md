# clank

A tmux sidebar that shows your AI agent fleet at a glance. Sessions are
projects, windows are workstreams, and every window shows whether its agent is
running, waiting for input, or done.

## Installation

### Homebrew

```bash
brew install voska/tap/clank
```

### TPM (tmux Plugin Manager)

Add to `~/.config/tmux/tmux.conf`:

```
set -g @plugin 'voska/clank'
```

Then press `prefix + I` to install.

### From source

```bash
git clone https://github.com/voska/clank.git
cd clank
make install
```

## Quick Start

1. Install clank via TPM or Homebrew
2. Press `prefix + Ctrl-s` to toggle the sidebar
3. Clank automatically detects Claude Code and Droid instances

No configuration required for basic agent detection. For richer metadata
(model name, context usage), set up the hook bridge:

```bash
cp agent-hook-bridge ~/.local/bin/
chmod +x ~/.local/bin/agent-hook-bridge
```

Then add hooks to `~/.claude/settings.json` — see [Hook Bridge](#hook-bridge).

## Features

- **Per-window agent state**: running, waiting, done, error
- **Zero-config detection**: identifies agents from process name + terminal content
- **Collapsible sessions**: `Space` to collapse, number keys to jump
- **Unseen markers**: dot indicator when state changes while you're elsewhere
- **Ghostty notifications**: OSC 777 + bell, works over SSH
- **Hook bridge**: optional richer metadata from Claude Code hooks

## Navigation

| Key        | Action                            |
|------------|-----------------------------------|
| `j` / `↓`  | Move down                        |
| `k` / `↑`  | Move up                          |
| `Enter`    | Switch to window / toggle collapse |
| `Space`    | Toggle session collapse           |
| `1`-`9`    | Jump to session by index          |
| `q`        | Hide sidebar                      |

## Configuration

Set options in `tmux.conf`:

```
set -g @clank-width 30        # sidebar width (default: 30)
set -g @clank-position left   # left or right (default: left)
```

## Hook Bridge

For Claude Code integration with model/context metadata:

```json
{
  "hooks": {
    "UserPromptSubmit": [{"matcher": "", "hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge submit"}]}],
    "Notification": [{"matcher": "", "hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge notification"}]}],
    "Stop": [{"hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge stop"}]}],
    "StatusLine": [{"hooks": [{"type": "command", "command": "$HOME/.local/bin/agent-hook-bridge statusline"}]}]
  }
}
```

## tmux Requirements

```
set -g allow-passthrough all    # for Ghostty notifications over SSH
set -g monitor-bell on
set -g bell-action other
```

## License

[MIT](LICENSE)
