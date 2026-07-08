#!/usr/bin/env bash
#
# diff-review — reference jin plugin (action-only; see jin-plugin.yaml).
# Launch with: jin plugin run diff-review --session <selector>
#
# Two-stage design, both stages living in this one file behind --inner:
#
#   1. Outer stage: runs as the plugin process. jin injects the full JIN_*
#      environment here (JIN_SESSION_ID, JIN_WORKDIR, ...). It opens a tmux
#      popup over the session's pane via `jin pane popup`.
#   2. Inner stage: runs *inside* that popup. A popup is a fresh process tmux
#      spawns on its own — it does NOT inherit this script's environment, so
#      JIN_SESSION_ID and JIN_WORKDIR are passed as positional args on the
#      popup's command line instead. This hand-off is the main thing this
#      example demonstrates; see build_inner_cmd below for how it's done
#      safely.
#
# Behavior inside the popup:
#   - Shows `git diff` for the session's working directory (via delta if
#     installed, else less).
#   - If fzf is installed: lets you multi-select changed lines and attach a
#     comment to each, then sends the assembled feedback back into the
#     session as a prompt.
#   - If fzf is not installed: degrades to a single overall comment.
set -uo pipefail

# collect_feedback_fzf reads a diff on stdin, lets the reviewer multi-select
# changed lines with fzf, prompts a comment for each selected line, and
# prints the assembled feedback text (empty output means "nothing to send").
collect_feedback_fzf() {
  local diff=$1
  local selected
  selected=$(printf '%s\n' "$diff" \
    | grep -E '^[+-]' \
    | grep -Ev '^(\+\+\+|---)' \
    | fzf --multi --height=90% --reverse \
          --prompt="select changed lines to comment on (tab=select, enter=confirm) > ")

  if [[ -z "$selected" ]]; then
    return 0
  fi

  local out="" line comment
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    # This loop's own stdin is the here-string below, so the prompt reads
    # from /dev/tty explicitly — otherwise it would read from the same
    # here-string as `line` above and hit EOF instead of the user's input.
    read -r -p "Comment on: ${line}: " comment </dev/tty
    if [[ -n "$comment" ]]; then
      out+="Line: ${line}"$'\n'"Comment: ${comment}"$'\n\n'
    fi
  done <<<"$selected"

  [[ -n "$out" ]] && printf 'Diff review feedback:\n\n%s' "$out"
}

# collect_feedback_plain is the no-fzf fallback: one overall comment instead
# of per-line comments.
collect_feedback_plain() {
  local comment
  read -r -p "Overall feedback (leave empty to skip sending): " comment
  [[ -n "$comment" ]] && printf 'Diff review feedback:\n\n%s\n' "$comment"
}

# run_inner is everything that happens inside the popup: show the diff,
# collect feedback, send it back to the session.
run_inner() {
  local session_id=$1 workdir=$2

  local diff
  if ! diff=$(git -C "$workdir" diff 2>&1); then
    echo "diff-review: git diff failed:" >&2
    echo "$diff" >&2
    read -r -p "Press Enter to close... " _ || true
    return 1
  fi

  if [[ -z "$diff" ]]; then
    echo "No unstaged changes in ${workdir}."
    read -r -p "Press Enter to close... " _ || true
    return 0
  fi

  if command -v delta >/dev/null 2>&1; then
    printf '%s\n' "$diff" | delta
  else
    printf '%s\n' "$diff" | less -R
  fi

  local feedback=""
  if command -v fzf >/dev/null 2>&1; then
    feedback=$(collect_feedback_fzf "$diff")
  else
    feedback=$(collect_feedback_plain)
  fi

  if [[ -z "$feedback" ]]; then
    echo "No feedback entered; nothing sent."
    return 0
  fi

  if ! printf '%s' "$feedback" | jin session send "$session_id" -; then
    echo "diff-review: failed to send feedback (see error above)" >&2
    read -r -p "Press Enter to close... " _ || true
    return 1
  fi

  echo
  echo "Sent feedback to session ${session_id}."
  sleep 1
}

# build_inner_cmd assembles the popup's command line as a SINGLE pre-quoted
# token. This matters because `jin pane popup` rejoins everything after `--`
# with plain spaces rather than preserving individual argv boundaries — so
# passing this script's path and the two positional args as three separate
# tokens would silently mis-split JIN_WORKDIR if it ever contained a space.
# Quoting the whole invocation with `printf %q` and handing it over as one
# token sidesteps that.
build_inner_cmd() {
  local self=$1 session_id=$2 workdir=$3
  printf '%q ' "$self" --inner "$session_id" "$workdir"
}

# run_outer is the plugin entry point: validate the environment jin injected,
# then open the popup.
run_outer() {
  : "${JIN_SESSION_ID:?JIN_SESSION_ID is not set; run via: jin plugin run diff-review --session <selector>}"
  : "${JIN_WORKDIR:?JIN_WORKDIR is not set}"

  if ! command -v git >/dev/null 2>&1; then
    echo "diff-review: git is required" >&2
    exit 1
  fi

  local self
  self="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/$(basename "${BASH_SOURCE[0]}")"

  local inner_cmd
  inner_cmd=$(build_inner_cmd "$self" "$JIN_SESSION_ID" "$JIN_WORKDIR")

  jin pane popup "$JIN_SESSION_ID" --title "diff-review" -- "$inner_cmd"
}

if [[ "${1:-}" == "--inner" ]]; then
  shift
  run_inner "${1:?missing session id}" "${2:?missing workdir}"
else
  run_outer
fi
