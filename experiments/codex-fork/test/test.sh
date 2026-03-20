#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LIB_FILE="$ROOT_DIR/lib/codex_fork.sh"
TMP_DIR="$ROOT_DIR/.tmp-test"

source "$LIB_FILE"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

assert_eq() {
  local actual="$1"
  local expected="$2"
  local message="$3"
  if [[ "$actual" != "$expected" ]]; then
    fail "$message: expected [$expected], got [$actual]"
  fi
}

assert_file_contains() {
  local file="$1"
  local pattern="$2"
  if ! grep -Fq "$pattern" "$file"; then
    fail "expected [$file] to contain [$pattern]"
  fi
}

assert_file_exists() {
  local file="$1"
  [[ -f "$file" ]] || fail "expected file to exist: $file"
}

assert_dir_exists() {
  local dir="$1"
  [[ -d "$dir" ]] || fail "expected directory to exist: $dir"
}

assert_path_exists() {
  local path="$1"
  [[ -e "$path" ]] || fail "expected path to exist: $path"
}

setup() {
  rm -rf "$TMP_DIR"
  mkdir -p "$TMP_DIR"
}

write_sample_session() {
  cat >"$TMP_DIR/session.jsonl" <<'EOF'
{"timestamp":"2026-03-20T10:00:00Z","type":"session_meta","payload":{"id":"session-123","cwd":"/tmp/demo"}}
{"timestamp":"2026-03-20T10:01:00Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"先检查 README 和 engine。"}]}}
{"timestamp":"2026-03-20T10:02:00Z","type":"event_msg","payload":{"type":"agent_message","message":"当前问题在任务拆分和状态耦合。","phase":"final_answer"}}
EOF
}

write_repo_session() {
  cat >"$TMP_DIR/repo-session.jsonl" <<EOF
{"timestamp":"2026-03-20T10:00:00Z","type":"session_meta","payload":{"id":"session-repo","cwd":"$TMP_DIR/repo"}}
EOF
}

init_git_repo() {
  mkdir -p "$TMP_DIR/repo"
  (
    cd "$TMP_DIR/repo"
    git init >/dev/null
    git config user.name test >/dev/null
    git config user.email test@example.com >/dev/null
    echo "demo" > README.md
    git add README.md
    git commit -m "init" >/dev/null
  )
}

test_extract_session_id() {
  setup
  write_sample_session

  local session_id
  session_id="$(session_id_from_file "$TMP_DIR/session.jsonl")"
  assert_eq "$session_id" "session-123" "session id should be parsed from session file"
}

test_extract_cwd() {
  setup
  write_sample_session

  local cwd
  cwd="$(session_cwd_from_file "$TMP_DIR/session.jsonl")"
  assert_eq "$cwd" "/tmp/demo" "cwd should be parsed from session file"
}

test_build_context_snapshot() {
  setup
  write_sample_session

  local snapshot
  snapshot="$(build_context_snapshot "$TMP_DIR/session.jsonl" 4)"

  [[ "$snapshot" == *"先检查 README 和 engine。"* ]] || fail "snapshot should include assistant output_text"
  [[ "$snapshot" == *"当前问题在任务拆分和状态耦合。"* ]] || fail "snapshot should include agent message"
}

test_write_task_packet() {
  setup
  write_sample_session

  local packet="$TMP_DIR/task-packet.md"
  write_task_packet \
    "$TMP_DIR/session.jsonl" \
    "拆出 review 子任务" \
    "$packet" \
    6

  assert_file_contains "$packet" "# Codex Fork Task Packet"
  assert_file_contains "$packet" "Parent Session ID: session-123"
  assert_file_contains "$packet" "Parent CWD: /tmp/demo"
  assert_file_contains "$packet" "Task: 拆出 review 子任务"
  assert_file_contains "$packet" "当前问题在任务拆分和状态耦合。"
}

test_write_status_file() {
  setup
  write_sample_session

  local status_file="$TMP_DIR/status.json"
  write_status_file \
    "$status_file" \
    "prepared" \
    "$TMP_DIR/session.jsonl" \
    "拆出 review 子任务"

  assert_file_contains "$status_file" '"status": "prepared"'
  assert_file_contains "$status_file" '"session_id": "session-123"'
  assert_file_contains "$status_file" '"task": "拆出 review 子任务"'
}

test_build_child_prompt() {
  setup
  write_sample_session

  local prompt
  prompt="$(build_child_prompt "$TMP_DIR/task-packet.md" "$TMP_DIR")"

  [[ "$prompt" == *"Read the task packet at: $TMP_DIR/task-packet.md"* ]] || fail "prompt should point to task packet"
  [[ "$prompt" == *"Write your final result to: $TMP_DIR/result.md"* ]] || fail "prompt should point to result file"
  [[ "$prompt" == *"Update status file: $TMP_DIR/status.json"* ]] || fail "prompt should point to status file"
  [[ "$prompt" == *"If Codex asks whether to trust this workspace, choose yes."* ]] || fail "prompt should explain trust confirmation"
  [[ "$prompt" == *"If Codex asks whether to use the session directory or current directory, choose current directory."* ]] || fail "prompt should explain cwd confirmation"
}

test_cli_creates_result_handoff_files() {
  setup
  write_sample_session

  bash "$ROOT_DIR/bin/codex-fork" \
    "$TMP_DIR/session.jsonl" \
    "拆出 review 子任务" \
    "$TMP_DIR/run" >/dev/null

  assert_file_exists "$TMP_DIR/run/task-packet.md"
  assert_file_exists "$TMP_DIR/run/status.json"
  assert_file_exists "$TMP_DIR/run/result.md"
  assert_file_exists "$TMP_DIR/run/launch.sh"
  assert_file_contains "$TMP_DIR/run/status.json" '"status": "prepared"'
  assert_file_contains "$TMP_DIR/run/result.md" "# Result Pending"
  assert_file_contains "$TMP_DIR/run/launch.sh" "Write your final result to:"
}

test_prepare_child_workspace_falls_back_to_subdir() {
  setup
  write_sample_session

  local workspace_info
  workspace_info="$(prepare_child_workspace "$TMP_DIR/session.jsonl" "$TMP_DIR/run")"

  assert_eq "$workspace_info" "$TMP_DIR/run/workspace|dir" "non-git cwd should use plain directory workspace"
  assert_dir_exists "$TMP_DIR/run/workspace"
}

test_prepare_child_workspace_uses_git_worktree() {
  setup
  init_git_repo
  write_repo_session

  local workspace_info
  workspace_info="$(prepare_child_workspace "$TMP_DIR/repo-session.jsonl" "$TMP_DIR/run")"

  [[ "$workspace_info" == "$TMP_DIR/run/workspace|git-worktree" ]] || fail "git repo should use git worktree"
  assert_path_exists "$TMP_DIR/run/workspace/.git"
}

test_cli_uses_isolated_workspace_in_launch_script() {
  setup
  init_git_repo
  write_repo_session

  bash "$ROOT_DIR/bin/codex-fork" \
    "$TMP_DIR/repo-session.jsonl" \
    "拆出 review 子任务" \
    "$TMP_DIR/run" >/dev/null

  assert_file_contains "$TMP_DIR/run/status.json" '"workspace_mode": "git-worktree"'
  assert_file_contains "$TMP_DIR/run/status.json" "\"workspace_dir\": \"$TMP_DIR/run/workspace\""
  assert_file_contains "$TMP_DIR/run/launch.sh" "cd $TMP_DIR/run/workspace"
  assert_file_contains "$TMP_DIR/run/launch.sh" "Work in isolated workspace:"
}

test_cli_uses_absolute_handoff_paths_in_launch_script() {
  setup
  init_git_repo
  write_repo_session

  local rel_run="experiments/codex-fork/.tmp-test/run-rel"
  bash "$ROOT_DIR/bin/codex-fork" \
    "$TMP_DIR/repo-session.jsonl" \
    "拆出 review 子任务" \
    "$rel_run" >/dev/null

  assert_file_contains "$ROOT_DIR/.tmp-test/run-rel/launch.sh" "Read the task packet at: $ROOT_DIR/.tmp-test/run-rel/task-packet.md"
  assert_file_contains "$ROOT_DIR/.tmp-test/run-rel/launch.sh" "Write your final result to: $ROOT_DIR/.tmp-test/run-rel/result.md"
  assert_file_contains "$ROOT_DIR/.tmp-test/run-rel/launch.sh" "Update status file: $ROOT_DIR/.tmp-test/run-rel/status.json"
}

test_build_ghostty_open_command() {
  setup

  local command
  command="$(build_ghostty_open_command "$TMP_DIR/run/launch.sh" "Codex Fork: task-1")"

  [[ "$command" == open\ -na\ Ghostty.app\ --args* ]] || fail "ghostty command should use open -na Ghostty.app"
  [[ "$command" == *"--title='Codex Fork: task-1'"* ]] || fail "ghostty command should set window title"
  [[ "$command" == *"-e /bin/sh -lc"* ]] || fail "ghostty command should execute launch.sh via shell"
  [[ "$command" == *"$TMP_DIR/run/launch.sh"* ]] || fail "ghostty command should target launch.sh"
}

test_build_window_title() {
  setup
  write_sample_session

  local title
  title="$(build_window_title "$TMP_DIR/session.jsonl" "拆出 review 子任务")"

  assert_eq "$title" "Codex Fork [session-123] 拆出 review 子任务" "window title should include session id and task"
}

test_ghostty_wrapper_prepares_run_and_prints_command() {
  setup
  write_sample_session

  local output
  output="$(
    CODEX_FORK_GHOSTTY_DRY_RUN=1 \
      bash "$ROOT_DIR/bin/codex-fork-ghostty" \
      "$TMP_DIR/session.jsonl" \
      "拆出 review 子任务" \
      "$TMP_DIR/run"
  )"

  assert_file_exists "$TMP_DIR/run/launch.sh"
  [[ "$output" == *"session id: session-123"* ]] || fail "wrapper should print extracted session id"
  [[ "$output" == *"window title: Codex Fork [session-123] 拆出 review 子任务"* ]] || fail "wrapper should print window title"
  [[ "$output" == *"ghostty command:"* ]] || fail "wrapper should print ghostty command"
  [[ "$output" == *"open -na Ghostty.app --args"* ]] || fail "wrapper should print open command"
  [[ "$output" == *"--title='Codex Fork [session-123] 拆出 review 子任务'"* ]] || fail "wrapper should print title argument"
}

run_tests() {
  test_extract_session_id
  test_extract_cwd
  test_build_context_snapshot
  test_write_task_packet
  test_write_status_file
  test_build_child_prompt
  test_cli_creates_result_handoff_files
  test_prepare_child_workspace_falls_back_to_subdir
  test_prepare_child_workspace_uses_git_worktree
  test_cli_uses_isolated_workspace_in_launch_script
  test_cli_uses_absolute_handoff_paths_in_launch_script
  test_build_window_title
  test_build_ghostty_open_command
  test_ghostty_wrapper_prepares_run_and_prints_command
}

run_tests
echo "PASS"
