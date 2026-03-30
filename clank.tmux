#!/usr/bin/env bash
# clank.tmux — persistent sidebar that pre-splits windows to avoid resize flash
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

# Find a placeholder pane (running 'cat' or 'sleep') in the current window's leftmost slot
find_placeholder() {
  tmux list-panes -F '#{pane_id} #{pane_left} #{pane_current_command}' 2>/dev/null \
    | awk '$2 == "0" && ($3 == "cat" || $3 == "sleep") {print $1; exit}'
}

ensure() {
  local clank_pane
  clank_pane=$(find_clank_pane)

  if [ -n "$clank_pane" ] && in_current_window "$clank_pane"; then
    return # already here
  fi

  # Check if this window already has a placeholder slot at the right width
  local placeholder
  placeholder=$(find_placeholder)

  if [ -n "$clank_pane" ] && [ -n "$placeholder" ]; then
    # Swap clank into the placeholder slot — no geometry change, no resize flash
    tmux swap-pane -s "$clank_pane" -t "$placeholder" 2>/dev/null
  elif [ -n "$clank_pane" ]; then
    # No placeholder — fall back to join-pane (will cause resize)
    local target_pane
    target_pane=$(tmux list-panes -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
      | awk '$2 != "clank" {print $1; exit}')
    [ -z "$target_pane" ] && return
    tmux join-pane -hb -l "$WIDTH" -t "$target_pane" -s "$clank_pane" 2>/dev/null
  else
    # No clank running at all — start fresh
    tmux split-window -hb -l "$WIDTH" "$CLANK_BIN --width $WIDTH" 2>/dev/null
    tmux select-pane -R 2>/dev/null
  fi
}

# Pre-split a new window with a placeholder pane at the sidebar width
presplit() {
  # Only if clank is active (toggle is on)
  local clank_pane
  clank_pane=$(find_clank_pane)
  [ -z "$clank_pane" ] && return

  # Create a placeholder pane on the left — 'cat' keeps it alive with minimal resources
  tmux split-window -hb -l "$WIDTH" -d "exec cat" 2>/dev/null
}

toggle() {
  local clank_pane
  clank_pane=$(find_clank_pane)

  if [ -n "$clank_pane" ]; then
    # OFF: kill clank, kill all placeholder panes, remove hooks
    tmux kill-pane -t "$clank_pane" 2>/dev/null
    # Kill placeholder panes in all windows
    tmux list-panes -a -F '#{pane_id} #{pane_left} #{pane_current_command}' 2>/dev/null \
      | awk '$2 == "0" && ($3 == "cat" || $3 == "sleep") {print $1}' \
      | while read -r p; do tmux kill-pane -t "$p" 2>/dev/null; done
    tmux set-hook -gu after-select-window[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu client-session-changed[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu after-new-window[$HOOK_IDX] 2>/dev/null
  else
    # ON: pre-split current window, start clank, set hooks
    tmux split-window -hb -l "$WIDTH" "$CLANK_BIN --width $WIDTH" 2>/dev/null
    tmux select-pane -R 2>/dev/null
    tmux set-hook -g after-select-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g client-session-changed[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g after-new-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux presplit'"
  fi
}

case "${1:-toggle}" in
  toggle) toggle ;;
  ensure) ensure ;;
  presplit) presplit ;;
  focus)
    ensure
    clank_pane=$(find_clank_pane)
    [ -n "$clank_pane" ] && tmux select-pane -t "$clank_pane"
    ;;
esac
