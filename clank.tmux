#!/usr/bin/env bash
# clank.tmux — persistent sidebar using swap-pane to avoid resize flash
CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLANK_BIN="$CURRENT_DIR/bin/clank"
_width=$(tmux show-option -gqv "@clank-width" 2>/dev/null)
WIDTH="${_width:-30}"
HOOK_IDX=100
LOG="/tmp/clank-tmux.log"

log() { printf '%s  %s\n' "$(date +%H:%M:%S)" "$*" >> "$LOG"; }

find_clank_pane() {
  tmux list-panes -a -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
    | awk '$2 == "clank" {print $1; exit}'
}

in_current_window() {
  tmux list-panes -F '#{pane_id}' 2>/dev/null | grep -qF "$1"
}

# Find placeholder (cat) in current window at left edge
find_placeholder() {
  tmux list-panes -F '#{pane_id} #{pane_left} #{pane_current_command}' 2>/dev/null \
    | awk '$2 == "0" && $3 == "cat" {print $1; exit}'
}

ensure() {
  local clank_pane
  clank_pane=$(find_clank_pane)
  log "ensure: clank_pane=$clank_pane"

  if [ -z "$clank_pane" ]; then
    log "ensure: no clank pane found, skipping"
    return
  fi

  if in_current_window "$clank_pane"; then
    # Ensure correct width (terminal size may differ across clients)
    local cur_width
    cur_width=$(tmux display-message -t "$clank_pane" -p '#{pane_width}' 2>/dev/null)
    if [ "$cur_width" != "$WIDTH" ]; then
      log "ensure: resizing clank $cur_width -> $WIDTH"
      tmux resize-pane -t "$clank_pane" -x "$WIDTH" -y 100% 2>/dev/null
    fi
    return
  fi

  local placeholder
  placeholder=$(find_placeholder)
  log "ensure: placeholder=$placeholder"

  if [ -n "$placeholder" ]; then
    log "ensure: swap-pane clank=$clank_pane placeholder=$placeholder"
    tmux swap-pane -s "$clank_pane" -t "$placeholder" 2>>"$LOG"
  else
    log "ensure: join-pane clank=$clank_pane"
    local target_pane
    target_pane=$(tmux list-panes -F '#{pane_id} #{pane_current_command}' 2>/dev/null \
      | awk '$2 != "clank" {print $1; exit}')
    [ -z "$target_pane" ] && return
    tmux join-pane -hb -l "$WIDTH" -t "$target_pane" -s "$clank_pane" 2>>"$LOG"
  fi

  # Ensure correct size regardless of client terminal or pane layout
  tmux resize-pane -t "$clank_pane" -x "$WIDTH" -y 100% 2>/dev/null
}

presplit() {
  local clank_pane
  clank_pane=$(find_clank_pane)
  [ -z "$clank_pane" ] && return

  log "presplit: creating placeholder in new window"
  tmux split-window -hb -l "$WIDTH" -d "exec cat" 2>>"$LOG"
}

toggle() {
  local clank_pane
  clank_pane=$(find_clank_pane)

  if [ -n "$clank_pane" ]; then
    log "toggle OFF"
    tmux kill-pane -t "$clank_pane" 2>/dev/null
    # Kill all placeholder panes
    tmux list-panes -a -F '#{pane_id} #{pane_left} #{pane_current_command}' 2>/dev/null \
      | awk '$2 == "0" && $3 == "cat" {print $1}' \
      | while read -r p; do tmux kill-pane -t "$p" 2>/dev/null; done
    tmux set-hook -gu after-select-window[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu client-session-changed[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu after-new-window[$HOOK_IDX] 2>/dev/null
    tmux set-hook -gu client-resized[$HOOK_IDX] 2>/dev/null
  else
    log "toggle ON"
    # Pre-split ALL existing windows with placeholders first
    tmux list-windows -a -F '#{session_name}:#{window_index}' 2>/dev/null | while read -r win; do
      local sess_name="${win%%:*}"
      [ "$sess_name" = "_clank" ] && continue
      # Check if window already has a left-side pane at width 30
      local existing_left
      existing_left=$(tmux list-panes -t "$win" -F '#{pane_left} #{pane_width}' 2>/dev/null | awk '$1 == "0" && $2 == "'"$WIDTH"'" {print "yes"; exit}')
      if [ -z "$existing_left" ]; then
        log "toggle ON: presplit $win"
        tmux split-window -hb -l "$WIDTH" -d -t "$win" "exec cat" 2>>"$LOG"
      fi
    done

    # Start clank in current window's placeholder
    local placeholder
    placeholder=$(find_placeholder)
    if [ -n "$placeholder" ]; then
      log "toggle ON: replacing placeholder $placeholder with clank"
      tmux respawn-pane -k -t "$placeholder" "$CLANK_BIN --width $WIDTH" 2>>"$LOG"
    else
      log "toggle ON: no placeholder, split-window"
      tmux split-window -hb -l "$WIDTH" "$CLANK_BIN --width $WIDTH" 2>>"$LOG"
    fi
    tmux select-pane -R 2>/dev/null

    tmux set-hook -g after-select-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g client-session-changed[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
    tmux set-hook -g after-new-window[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux presplit'"
    tmux set-hook -g client-resized[$HOOK_IDX] "run-shell '$CURRENT_DIR/clank.tmux ensure'"
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
