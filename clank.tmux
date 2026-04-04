#!/usr/bin/env bash
# clank.tmux — persistent sidebar (TPM-compatible)
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLANK_BIN="$CURRENT_DIR/bin/clank"
_width=$(tmux show-option -gqv "@clank-width" 2>/dev/null)
WIDTH="${_width:-30}"
HOOK_IDX=100
LOG="/tmp/clank-tmux.log"
LOCK="/tmp/clank-ensure.lock"

log() { printf '%s  %s\n' "$(date +%H:%M:%S)" "$*" >> "$LOG"; }

find_clank_pane() {
  tmux list-panes -a -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
    | awk '$2 == "clank" {print $1; exit}'
}

in_current_window() {
  tmux list-panes -F '#{pane_id}' 2>/dev/null | grep -qF "$1"
}

ensure() {
  # Prevent re-entrant calls (split-window triggers after-select-window)
  if [ -f "$LOCK" ]; then
    local lock_age=$(( $(date +%s) - $(stat -f %m "$LOCK" 2>/dev/null || echo 0) ))
    [ "$lock_age" -lt 3 ] && return
  fi
  touch "$LOCK"

  local clank_pane
  clank_pane=$(find_clank_pane)

  if [ -z "$clank_pane" ]; then
    log "ensure: restarting clank"
    tmux split-window -fhb -l "$WIDTH" "$CLANK_BIN --width $WIDTH" 2>>"$LOG"
    tmux select-pane -R 2>/dev/null
    rm -f "$LOCK"
    return
  fi

  if in_current_window "$clank_pane"; then
    local cur_width cur_height win_height
    cur_width=$(tmux display-message -t "$clank_pane" -p '#{pane_width}' 2>/dev/null)
    cur_height=$(tmux display-message -t "$clank_pane" -p '#{pane_height}' 2>/dev/null)
    win_height=$(tmux display-message -p '#{window_height}' 2>/dev/null)

    # If clank isn't full height, it got trapped in a sub-split — break and rejoin
    if [ -n "$win_height" ] && [ -n "$cur_height" ] && [ "$cur_height" -lt "$((win_height - 2))" ]; then
      log "ensure: fixing layout (pane_height=$cur_height window_height=$win_height)"
      tmux break-pane -d -s "$clank_pane" 2>/dev/null
      clank_pane=$(find_clank_pane)
      local target
      target=$(tmux list-panes -F '#{pane_id}' 2>/dev/null | head -1)
      if [ -n "$target" ] && [ -n "$clank_pane" ]; then
        tmux join-pane -fhb -l "$WIDTH" -t "$target" -s "$clank_pane" 2>>"$LOG"
        clank_pane=$(find_clank_pane)
      fi
    elif [ -n "$cur_width" ] && [ "$cur_width" != "$WIDTH" ]; then
      tmux resize-pane -t "$clank_pane" -x "$WIDTH" 2>/dev/null
    fi
    rm -f "$LOCK"
    return
  fi

  # Move clank to current window (full-height left split)
  log "ensure: moving clank"
  local target_pane
  target_pane=$(tmux list-panes -F '#{pane_id}' 2>/dev/null | head -1)
  [ -z "$target_pane" ] && { rm -f "$LOCK"; return; }
  tmux join-pane -fhb -l "$WIDTH" -t "$target_pane" -s "$clank_pane" 2>>"$LOG"
  clank_pane=$(find_clank_pane)
  [ -n "$clank_pane" ] && tmux resize-pane -t "$clank_pane" -x "$WIDTH" 2>/dev/null
  rm -f "$LOCK"
}

toggle() {
  local clank_pane
  clank_pane=$(find_clank_pane)

  if [ -n "$clank_pane" ]; then
    log "toggle OFF"
    tmux kill-pane -t "$clank_pane" 2>/dev/null
    tmux set-hook -gu after-select-window[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu client-session-changed[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu after-new-window[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu client-resized[$HOOK_IDX] 2>/dev/null
    rm -f "$LOCK"
  else
    log "toggle ON"
    touch "$LOCK"
    tmux split-window -fhb -l "$WIDTH" "$CLANK_BIN --width $WIDTH" 2>>"$LOG"
    tmux select-pane -R 2>/dev/null
    tmux set-hook -g after-select-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g client-session-changed[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g after-new-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g client-resized[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    rm -f "$LOCK"
  fi
}

init() {
  # TPM plugin init — set up key bindings from tmux options
  _key=$(tmux show-option -gqv "@clank-key" 2>/dev/null)
  KEY="${_key:-o}"
  _focus_key=$(tmux show-option -gqv "@clank-focus-key" 2>/dev/null)
  FOCUS_KEY="${_focus_key:-O}"
  tmux bind-key "$KEY" run-shell "'$CURRENT_DIR/clank.tmux' toggle"
  tmux bind-key "$FOCUS_KEY" run-shell "'$CURRENT_DIR/clank.tmux' focus"
  log "init: bound $KEY=toggle $FOCUS_KEY=focus"
}

case "${1:-init}" in
  init) init ;;
  toggle) toggle ;;
  ensure) ensure ;;
  focus)
    ensure
    clank_pane=$(find_clank_pane)
    [ -n "$clank_pane" ] && tmux select-pane -t "$clank_pane"
    ;;
esac
