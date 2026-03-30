#!/usr/bin/env bash
# clank.tmux — persistent sidebar that follows you across windows/sessions
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLANK_BIN="$CURRENT_DIR/bin/clank"
_width=$(tmux show-option -gqv "@clank-width" 2>/dev/null)
WIDTH="${_width:-30}"
HOOK_IDX=100

find_clank_pane() {
  tmux list-panes -a -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
    | awk '$2 == "clank" {print $1; exit}'
}

in_current_window() {
  local pane="$1"
  tmux list-panes -F '#{pane_id}' 2>/dev/null | grep -qF "$pane"
}

ensure() {
  local pane
  pane=$(find_clank_pane)
  [ -z "$pane" ] && return

  if in_current_window "$pane"; then
    return
  fi

  # Target the first pane that ISN'T clank, so clank lands on the far left
  local target_pane
  target_pane=$(tmux list-panes -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
    | awk '$2 != "clank" {print $1; exit}')
  [ -z "$target_pane" ] && return
  tmux join-pane -hb -l "$WIDTH" -t "$target_pane" -s "$pane" 2>/dev/null
}

toggle() {
  local pane
  pane=$(find_clank_pane)

  if [ -n "$pane" ]; then
    # OFF: kill pane, remove hooks
    tmux kill-pane -t "$pane" 2>/dev/null
    tmux set-hook -gu after-select-window[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu client-session-changed[$HOOK_IDX] 2>/dev/null
  else
    # ON: create pane, set hooks, focus main pane
    tmux split-window -hb -l "$WIDTH" "$CLANK_BIN --width $WIDTH"
    tmux select-pane -R
    tmux set-hook -g after-select-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g client-session-changed[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
  fi
}

case "${1:-toggle}" in
  toggle) toggle ;;
  ensure) ensure ;;
  focus)
    ensure
    pane=$(find_clank_pane)
    [ -n "$pane" ] && tmux select-pane -t "$pane"
    ;;
esac
