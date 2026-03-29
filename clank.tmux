#!/usr/bin/env bash
# clank.tmux — toggle clank sidebar pane
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLANK_BIN="$CURRENT_DIR/bin/clank"

_width=$(tmux show-option -gqv "@clank-width" 2>/dev/null)
WIDTH="${_width:-30}"

# Find existing clank pane in current window
clank_pane=$(tmux list-panes -F '#{pane_id} #{pane_current_command}' | awk '$2 == "clank" {print $1; exit}')

case "${1:-toggle}" in
  toggle)
    if [ -n "$clank_pane" ]; then
      tmux kill-pane -t "$clank_pane"
    else
      tmux split-window -hb -l "$WIDTH" "$CLANK_BIN --width $WIDTH"
    fi
    ;;
  focus)
    if [ -z "$clank_pane" ]; then
      tmux split-window -hb -l "$WIDTH" "$CLANK_BIN --width $WIDTH"
    fi
    clank_pane=$(tmux list-panes -F '#{pane_id} #{pane_current_command}' | awk '$2 == "clank" {print $1; exit}')
    [ -n "$clank_pane" ] && tmux select-pane -t "$clank_pane"
    ;;
esac
