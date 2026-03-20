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

codex_sessions_dir() {
  if [[ -n "${CODEX_FORK_SESSIONS_DIR:-}" ]]; then
    printf '%s\n' "$CODEX_FORK_SESSIONS_DIR"
    return 0
  fi

  if [[ -n "${CODEX_HOME:-}" ]]; then
    printf '%s/sessions\n' "$CODEX_HOME"
    return 0
  fi

  printf '%s/.codex/sessions\n' "$HOME"
}

list_session_files() {
  local sessions_dir="${1:-$(codex_sessions_dir)}"

  if [[ ! -d "$sessions_dir" ]]; then
    return 0
  fi

  find "$sessions_dir" -type f -name '*.jsonl' | LC_ALL=C sort -r
}

latest_session_file() {
  local sessions_dir="${1:-$(codex_sessions_dir)}"
  list_session_files "$sessions_dir" | head -n 1
}

pick_session_file() {
  local sessions_dir="${1:-$(codex_sessions_dir)}"
  local -a sessions=()
  local line
  local limit=5
  local count=0
  local choice
  local index

  while IFS= read -r line; do
    sessions+=("$line")
    count=$((count + 1))
    if [[ "$count" -ge "$limit" ]]; then
      break
    fi
  done < <(list_session_files "$sessions_dir")

  if [[ "${#sessions[@]}" -eq 0 ]]; then
    return 1
  fi

  echo "Pick a parent session:" >&2
  for index in "${!sessions[@]}"; do
    printf '  %d. %s\n' "$((index + 1))" "${sessions[$index]}" >&2
  done
  printf 'Choice [1-%d]: ' "${#sessions[@]}" >&2
  read -r choice

  if [[ ! "$choice" =~ ^[0-9]+$ ]]; then
    return 1
  fi
  if [[ "$choice" -lt 1 || "$choice" -gt "${#sessions[@]}" ]]; then
    return 1
  fi

  printf '%s\n' "${sessions[$((choice - 1))]}"
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

result_state_from_file() {
  local result_file="$1"

  if [[ ! -f "$result_file" ]]; then
    printf 'missing\n'
    return 0
  fi

  if grep -Fq '# Result Pending' "$result_file"; then
    printf 'pending\n'
    return 0
  fi

  printf 'written\n'
}

print_run_status() {
  local run_dir="$1"
  local status_file="$run_dir/status.json"
  local result_file="$run_dir/result.md"
  local launch_file="$run_dir/launch.sh"
  local status
  local session_id
  local task
  local workspace_dir
  local result_state
  local next_step

  if [[ ! -f "$status_file" ]]; then
    echo "missing: $status_file" >&2
    return 1
  fi

  status="$(jq -r '.status // "unknown"' "$status_file")"
  session_id="$(jq -r '.session_id // "unknown"' "$status_file")"
  task="$(jq -r '.task // ""' "$status_file")"
  workspace_dir="$(jq -r '.workspace_dir // ""' "$status_file")"
  result_state="$(result_state_from_file "$result_file")"

  if [[ "$status" == "completed" && "$result_state" == "written" ]]; then
    next_step="inspect result.md and workspace changes"
  elif [[ -f "$launch_file" ]]; then
    next_step="run the generated launch script"
  else
    next_step="inspect status.json and result.md"
  fi

  cat <<EOF
run status: $status
session id: $session_id
task: $task
workspace: $workspace_dir
result: $result_state
next: $next_step
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
  local -a args

  session_id="$(session_id_from_file "$session_file")"
  prompt="$(build_child_prompt "$packet_file" "$output_dir" "$workspace_dir")"
  args=("codex" "fork")

  if [[ "${CODEX_FORK_ENABLE_BYPASS:-1}" == "1" ]]; then
    args+=("--dangerously-bypass-approvals-and-sandbox")
  fi

  args+=("-C" "$workspace_dir" "$session_id" "$prompt")

  printf 'cd %q &&' "$workspace_dir"
  printf ' %q' "${args[@]}"
  printf '\n'
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
