#!/usr/bin/env bash
# clank.tmux — TPM plugin bootstrap
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLANK_BIN="$CURRENT_DIR/bin/clank"
CLANK_WIDTH=$(tmux show-option -gqv "@clank-width" 2>/dev/null || echo "30")
CLANK_POSITION=$(tmux show-option -gqv "@clank-position" 2>/dev/null || echo "left")
STASH_SESSION="_clank"

if [ ! -x "$CLANK_BIN" ]; then
  "$CURRENT_DIR/scripts/install.sh" "$CURRENT_DIR/bin"
fi

if ! tmux has-session -t "$STASH_SESSION" 2>/dev/null; then
  tmux new-session -d -s "$STASH_SESSION" -x 200 -y 50 "$CLANK_BIN --width $CLANK_WIDTH"
fi

toggle_sidebar() {
  local clank_pane
  clank_pane=$(tmux list-panes -F '#{pane_id}:#{pane_current_command}' | grep ':clank$' | head -1 | cut -d: -f1)

  if [ -n "$clank_pane" ]; then
    tmux join-pane -d -t "${STASH_SESSION}:0.0" -s "$clank_pane"
  else
    local stash_pane="${STASH_SESSION}:0.0"
    if [ "$CLANK_POSITION" = "right" ]; then
      tmux join-pane -h -l "$CLANK_WIDTH" -t "$(tmux display-message -p '#{pane_id}')" -s "$stash_pane"
    else
      tmux join-pane -hb -l "$CLANK_WIDTH" -t "$(tmux display-message -p '#{pane_id}')" -s "$stash_pane"
    fi
  fi
}

case "${1:-}" in
  toggle)
    toggle_sidebar
    ;;
  focus)
    toggle_sidebar
    clank_pane=$(tmux list-panes -F '#{pane_id}:#{pane_current_command}' | grep ':clank$' | head -1 | cut -d: -f1)
    [ -n "$clank_pane" ] && tmux select-pane -t "$clank_pane"
    ;;
  *)
    tmux bind-key -T prefix C-s run-shell "$CURRENT_DIR/clank.tmux toggle"
    tmux bind-key -T prefix C-f run-shell "$CURRENT_DIR/clank.tmux focus"
    ;;
esac
