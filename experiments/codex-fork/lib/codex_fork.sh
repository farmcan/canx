#!/usr/bin/env bash

set -euo pipefail

session_id_from_file() {
  local session_file="$1"
  jq -r '
    select(.type == "session_meta")
    | .payload.id // empty
  ' "$session_file" | head -n 1
}

session_cwd_from_file() {
  local session_file="$1"
  jq -r '
    select(.type == "session_meta")
    | .payload.cwd // empty
  ' "$session_file" | head -n 1
}

is_git_repo() {
  local dir="$1"
  git -C "$dir" rev-parse --is-inside-work-tree >/dev/null 2>&1
}

prepare_child_workspace() {
  local session_file="$1"
  local output_dir="$2"

  local parent_cwd
  local workspace_dir

  parent_cwd="$(session_cwd_from_file "$session_file")"
  workspace_dir="$output_dir/workspace"
  mkdir -p "$output_dir"

  if [[ -n "$parent_cwd" ]] && [[ -d "$parent_cwd" ]] && is_git_repo "$parent_cwd"; then
    git -C "$parent_cwd" worktree add --detach "$workspace_dir" >/dev/null 2>&1
    printf '%s|%s\n' "$workspace_dir" "git-worktree"
    return 0
  fi

  mkdir -p "$workspace_dir"
  printf '%s|%s\n' "$workspace_dir" "dir"
}

build_context_snapshot() {
  local session_file="$1"
  local limit="${2:-8}"

  jq -r '
    if .type == "response_item" and .payload.type == "message" then
      .payload.content[]?
      | select(.type == "output_text")
      | .text
    elif .type == "event_msg" and .payload.type == "agent_message" then
      .payload.message
    else
      empty
    end
  ' "$session_file" | tail -n "$limit"
}

write_status_file() {
  local output_file="$1"
  local status="$2"
  local session_file="$3"
  local task_text="$4"
  local workspace_dir="${5:-}"
  local workspace_mode="${6:-}"

  local session_id
  session_id="$(session_id_from_file "$session_file")"

  mkdir -p "$(dirname "$output_file")"
  jq -n \
    --arg status "$status" \
    --arg session_file "$session_file" \
    --arg session_id "$session_id" \
    --arg task "$task_text" \
    --arg workspace_dir "$workspace_dir" \
    --arg workspace_mode "$workspace_mode" \
    '{
      status: $status,
      session_file: $session_file,
      session_id: $session_id,
      task: $task,
      workspace_dir: $workspace_dir,
      workspace_mode: $workspace_mode
    }' >"$output_file"
}

write_result_template() {
  local output_file="$1"

  mkdir -p "$(dirname "$output_file")"
  cat >"$output_file" <<'EOF'
# Result Pending

Replace this file with the child Codex summary when the delegated task is done.
EOF
}

write_task_packet() {
  local session_file="$1"
  local task_text="$2"
  local output_file="$3"
  local limit="${4:-8}"

  local session_id
  local cwd
  local snapshot

  session_id="$(session_id_from_file "$session_file")"
  cwd="$(session_cwd_from_file "$session_file")"
  snapshot="$(build_context_snapshot "$session_file" "$limit")"

  mkdir -p "$(dirname "$output_file")"
  cat >"$output_file" <<EOF
# Codex Fork Task Packet

Parent Session File: $session_file
Parent Session ID: $session_id
Parent CWD: $cwd

Task: $task_text

## Inherited Context Snapshot
$snapshot
EOF
}

build_child_prompt() {
  local packet_file="$1"
  local output_dir="$2"
  local workspace_dir="${3:-$output_dir/workspace}"

  cat <<EOF
Read the task packet at: $packet_file

Work only on the delegated subtask described there.
Work in isolated workspace: $workspace_dir
Write your final result to: $output_dir/result.md
Update status file: $output_dir/status.json

If Codex asks whether to trust this workspace, choose yes.
If Codex asks whether to use the session directory or current directory, choose current directory.
If Codex asks whether to apply the handoff file edits, choose yes.

When finished, overwrite result.md with:
- what you changed
- what remains
- any blockers

Before you stop, update status.json so the status becomes "completed".
EOF
}

print_launch_command() {
  local session_file="$1"
  local packet_file="$2"
  local output_dir="$3"
  local workspace_dir="$4"
  local session_id
  local prompt

  session_id="$(session_id_from_file "$session_file")"
  prompt="$(build_child_prompt "$packet_file" "$output_dir" "$workspace_dir")"

  printf 'cd %q && codex fork %q %q\n' "$workspace_dir" "$session_id" "$prompt"
}

build_window_title() {
  local session_file="$1"
  local task_text="$2"
  local session_id

  session_id="$(session_id_from_file "$session_file")"
  printf 'Codex Fork [%s] %s\n' "$session_id" "$task_text"
}

shell_single_quote() {
  local value="$1"
  value="${value//\'/\'\\\'\'}"
  printf "'%s'" "$value"
}

build_ghostty_open_command() {
  local launch_file="$1"
  local title="${2:-Codex Fork}"

  printf 'open -na Ghostty.app --args --title=%s -e /bin/sh -lc %s\n' \
    "$(shell_single_quote "$title")" \
    "$(shell_single_quote "$launch_file")"
}
