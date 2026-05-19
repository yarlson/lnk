#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PLAN_PATH="${PLAN_PATH:-PLAN.md}"
REPO_AGENTS_PATH="${REPO_AGENTS_PATH:-AGENTS.md}"
DESIGN_PATH="${DESIGN_PATH:-DESIGN.md}"
START_PHASE="${START_PHASE:-1}"
END_PHASE="${END_PHASE:-}"
SLEEP_SECONDS="${SLEEP_SECONDS:-600}"
CODEX_BIN="${CODEX_BIN:-codex}"
STREAM_MODE="${STREAM_MODE:-pretty}"
LOG_DIR="${LOG_DIR:-$ROOT_DIR/codex-logs}"
RUN_STAMP="${RUN_STAMP:-$(date '+%Y%m%d-%H%M%S')}"
COLOR_RESET=""
COLOR_DIM=""
COLOR_HEADER=""
COLOR_THREAD=""
COLOR_TURN=""
COLOR_AGENT=""
COLOR_REASONING=""
COLOR_COMMAND=""
COLOR_FILE_CHANGE=""
COLOR_MCP=""
COLOR_COLLAB=""
COLOR_WEB_SEARCH=""
COLOR_TODO=""
COLOR_ERROR=""

CODEX_ARGS=(
    exec
    -C "$ROOT_DIR"
    --dangerously-bypass-approvals-and-sandbox
    --json
)

init_colors() {
    if [[ -t 1 && -z "${NO_COLOR:-}" ]]; then
        COLOR_RESET=$'\033[0m'
        COLOR_DIM=$'\033[2m'
        COLOR_HEADER=$'\033[1;34m'
        COLOR_THREAD=$'\033[35m'
        COLOR_TURN=$'\033[34m'
        COLOR_AGENT=$'\033[32m'
        COLOR_REASONING=$'\033[36m'
        COLOR_COMMAND=$'\033[33m'
        COLOR_FILE_CHANGE=$'\033[95m'
        COLOR_MCP=$'\033[94m'
        COLOR_COLLAB=$'\033[96m'
        COLOR_WEB_SEARCH=$'\033[92m'
        COLOR_TODO=$'\033[93m'
        COLOR_ERROR=$'\033[31m'
    fi
}

run_codex_prompt() {
    local prompt="$1"
    local log_name="$2"
    local jq_bin=""
    local log_path="$LOG_DIR/$log_name"

    jq_bin="$(command -v jq 2>/dev/null || true)"
    mkdir -p "$LOG_DIR"

    printf '[%s] Writing raw JSONL to %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$log_path"

    if [[ "$STREAM_MODE" == "raw" || -z "$jq_bin" ]]; then
        printf '%s\n' "$prompt" | "$CODEX_BIN" "${CODEX_ARGS[@]}" - | tee "$log_path"
        return
    fi

    # shellcheck disable=SC2016
    printf '%s\n' "$prompt" | "$CODEX_BIN" "${CODEX_ARGS[@]}" - |
        tee "$log_path" |
        "$jq_bin" --unbuffered -r \
            --arg c_reset "$COLOR_RESET" \
            --arg c_dim "$COLOR_DIM" \
            --arg c_thread "$COLOR_THREAD" \
            --arg c_turn "$COLOR_TURN" \
            --arg c_agent "$COLOR_AGENT" \
            --arg c_reasoning "$COLOR_REASONING" \
            --arg c_command "$COLOR_COMMAND" \
            --arg c_file_change "$COLOR_FILE_CHANGE" \
            --arg c_mcp "$COLOR_MCP" \
            --arg c_collab "$COLOR_COLLAB" \
            --arg c_web_search "$COLOR_WEB_SEARCH" \
            --arg c_todo "$COLOR_TODO" \
            --arg c_error "$COLOR_ERROR" '
    def prefix($kind; $color):
      $color + "[" + $kind + "]" + $c_reset;

    def emit($label; $color; $text):
      if ($text // "") == "" then empty else prefix($label; $color) + " " + $text end;

    def item_summary($event; $item):
      if $item.type == "agent_message" then
        emit("agent"; $c_agent; $item.text)
      elif $item.type == "reasoning" then
        emit("reasoning"; $c_reasoning; $item.text)
      elif $item.type == "command_execution" then
        emit("command"; $c_command; ($event + " " + ($item.status // "unknown") + ": " + ($item.command // "")))
      elif $item.type == "file_change" then
        emit("file_change"; $c_file_change; ($event + " " + ($item.status // "unknown") + " (" + (($item.changes // []) | length | tostring) + " files)"))
      elif $item.type == "mcp_tool_call" then
        emit("mcp"; $c_mcp; ($event + " " + ($item.status // "unknown") + ": " + ($item.server // "") + "/" + ($item.tool // "")))
      elif $item.type == "collab_tool_call" then
        emit("collab"; $c_collab; ($event + " " + ($item.status // "unknown") + ": " + ($item.tool // "")))
      elif $item.type == "web_search" then
        emit("web_search"; $c_web_search; ($event + ": " + ($item.query // "")))
      elif $item.type == "todo_list" then
        emit("todo"; $c_todo; (
          ($item.items // [])
          | map((if .completed then "[x] " else "[ ] " end) + (.text // ""))
          | join(" | ")
        ))
      elif $item.type == "error" then
        emit("item_error"; $c_error; ($item.message // ""))
      else
        emit("event"; $c_dim; ($event + " " + ($item.type // "unknown_item")))
      end;

    if .type == "thread.started" then
      emit("thread"; $c_thread; ("started " + (.thread_id // "")))
    elif .type == "turn.started" then
      emit("turn"; $c_turn; "started")
    elif .type == "turn.completed" then
      emit(
        "turn"; $c_turn;
        "completed input_tokens=\((.usage.input_tokens // 0)) cached_input_tokens=\((.usage.cached_input_tokens // 0)) output_tokens=\((.usage.output_tokens // 0)) reasoning_output_tokens=\((.usage.reasoning_output_tokens // 0))"
      )
    elif .type == "turn.failed" then
      emit("turn"; $c_error; ("failed: " + (.error.message // "unknown error")))
    elif .type == "error" then
      emit("error"; $c_error; (.message // "unknown error"))
    elif (.type == "item.started" or .type == "item.updated" or .type == "item.completed") then
      item_summary(.type; .item)
    else
      emit("event"; $c_dim; (.type // "unknown"))
    end
  '
}

detect_end_phase() {
    sed -n 's/^## Phase \([0-9][0-9]*\)\([[:space:]]*[-:].*\)$/\1/p' "$ROOT_DIR/$PLAN_PATH" |
        sort -n |
        tail -1
}

phase_title() {
    local phase="$1"

    sed -n "s/^## Phase ${phase}[[:space:]]*[-:][[:space:]]*//p" "$ROOT_DIR/$PLAN_PATH" | head -1
}

require_phase() {
    local phase="$1"
    local title=""

    title="$(phase_title "$phase")"
    if [[ -z "$title" ]]; then
        printf '%serror:%s Phase %s was not found in %s\n' \
            "$COLOR_ERROR" "$COLOR_RESET" "$phase" "$PLAN_PATH" >&2
        return 1
    fi
}

require_file() {
    local path="$1"

    if [[ ! -f "$ROOT_DIR/$path" ]]; then
        printf '%serror:%s required file %s was not found\n' \
            "$COLOR_ERROR" "$COLOR_RESET" "$path" >&2
        return 1
    fi
}

context_docs_block() {
    cat <<EOF
Read these context files before coding:
- ${REPO_AGENTS_PATH}
- ${DESIGN_PATH}
- ${PLAN_PATH}
Treat ${DESIGN_PATH} as the product contract and ${PLAN_PATH} as the implementation order.
Treat ${REPO_AGENTS_PATH} as the authoritative agent instructions for this run.
Use only repo-local files and official docs relevant to the selected phase.
When validating version-sensitive Rust, Cargo, Git, dependency, or test-runner behavior, search current official docs first and include 2026 in web searches.
EOF
}

implementation_prompt() {
    local phase="$1"
    local title="$2"

    cat <<EOF
$(context_docs_block)

Open ${PLAN_PATH} and locate exactly:

## Phase ${phase} - ${title}

Implement Phase ${phase} completely in the current lnk repo.
Use the phase's Goal, Steps, Validation, and Web validation sections as the task contract.
Before coding, inspect the current repo state and the files relevant to this phase.
Do not skip ahead into later phases unless the selected phase explicitly requires a small prerequisite.
Keep modules small and concern-based; do not recreate the deleted Go shape or large service files.
Do not add commands, flags, configuration, workflows, CI, release automation, or features outside ${DESIGN_PATH} and the selected phase.
Update ${DESIGN_PATH} only if implementation exposes a real product-design gap.
Update ${PLAN_PATH} only if the selected phase needs a corrected implementation order or validation gate.
Run the checks required by ${REPO_AGENTS_PATH} and the selected phase. Fix failures.
If a required tool such as cargo-nextest is missing, install it when practical or report the blocker explicitly; do not silently substitute weaker checks.
Do not commit or push unless the user explicitly requested it for this run.
EOF
}

validation_prompt() {
    local phase="$1"
    local title="$2"

    cat <<EOF
$(context_docs_block)

Open ${PLAN_PATH} and locate exactly:

## Phase ${phase} - ${title}

Validate that Phase ${phase} was implemented fully and correctly in the current lnk repo.
Compare the current implementation against the selected phase's Goal, Steps, Validation, and Web validation sections.
Reinspect the files touched by the phase and the surrounding modules.
Close whatever gaps are needed to make Phase ${phase} complete, correct, and well-verified.
Do not skip ahead into later phases except for a small prerequisite required to satisfy this phase.
Do not add commands, flags, configuration, workflows, CI, release automation, or features outside ${DESIGN_PATH} and the selected phase.
Fix issues, strengthen tests or verification where needed, and keep the work focused on satisfying Phase ${phase} end to end.
Run the checks required by ${REPO_AGENTS_PATH} and the selected phase. Fix failures.
If a required tool such as cargo-nextest is missing, install it when practical or report the blocker explicitly; do not silently substitute weaker checks.
Review git diff before finishing.
Do not commit or push unless the user explicitly requested it for this run.
EOF
}

main() {
    local phase
    local phase_title_text

    init_colors

    require_file "$PLAN_PATH"
    require_file "$REPO_AGENTS_PATH"
    require_file "$DESIGN_PATH"

    if [[ -z "$END_PHASE" ]]; then
        END_PHASE="$(detect_end_phase)"
    fi
    if [[ -z "$END_PHASE" ]]; then
        printf '%serror:%s could not detect end phase from %s\n' \
            "$COLOR_ERROR" "$COLOR_RESET" "$PLAN_PATH" >&2
        exit 1
    fi

    for ((phase = START_PHASE; phase <= END_PHASE; phase++)); do
        require_phase "$phase"
        phase_title_text="$(phase_title "$phase")"

        printf '\n%s[%s] Implementing Phase %d: %s%s\n' \
            "$COLOR_HEADER" "$(date '+%Y-%m-%d %H:%M:%S')" "$phase" "$phase_title_text" "$COLOR_RESET"
        run_codex_prompt \
            "$(implementation_prompt "$phase" "$phase_title_text")" \
            "${RUN_STAMP}-phase-$(printf '%02d' "$phase")-implement.jsonl"

        printf '\n%s[%s] Validating Phase %d: %s%s\n' \
            "$COLOR_HEADER" "$(date '+%Y-%m-%d %H:%M:%S')" "$phase" "$phase_title_text" "$COLOR_RESET"
        run_codex_prompt \
            "$(validation_prompt "$phase" "$phase_title_text")" \
            "${RUN_STAMP}-phase-$(printf '%02d' "$phase")-validate.jsonl"

        if ((phase < END_PHASE)); then
            printf '\n%s[%s] Sleeping for %d seconds before Phase %d%s\n' \
                "$COLOR_DIM" "$(date '+%Y-%m-%d %H:%M:%S')" "$SLEEP_SECONDS" "$((phase + 1))" "$COLOR_RESET"
            sleep "$SLEEP_SECONDS"
        fi
    done
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    main "$@"
fi
