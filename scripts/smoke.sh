set -euo pipefail

# smoke.sh
#
# Bucketed, cheap smoke test for chatgpt-cli.
# - Discovers models via: chatgpt --list-models
# - Buckets them by *what your CLI will do* (heuristics aligned with GetCapabilities)
# - Runs 1 minimal probe per bucket (cheap)
# - ALSO probes:
#     - the model marked "(current)"
#     - the latest dated releases for key families (gpt-5*, gpt-4o, o1)
#
# Optional env:
#   CHATGPT_BIN (default: chatgpt)
#   SMOKE_WEB (default: false)       # if true, run a web probe (costs more)
#   WEB_CONTEXT_SIZE (default: low)
#   MAX_TOKENS (default: 256)        # o1 can need >=256 to avoid "reasoning-only" empties
#   TIMEOUT_SECS (default: 30)

CHATGPT_BIN="${CHATGPT_BIN:-chatgpt}"
SMOKE_WEB="${SMOKE_WEB:-false}"
WEB_CONTEXT_SIZE="${WEB_CONTEXT_SIZE:-low}"
MAX_TOKENS="${MAX_TOKENS:-256}"
TIMEOUT_SECS="${TIMEOUT_SECS:-30}"

require() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: missing dependency: $1"; exit 2; }; }
require "$CHATGPT_BIN"
require awk
require sed
require grep
require sort
require tail
require head
require tr

PROMPT='Reply with exactly: pong'

run_with_timeout() {
  if command -v timeout >/dev/null 2>&1; then
    timeout "$TIMEOUT_SECS" "$@"
  elif command -v gtimeout >/dev/null 2>&1; then
    gtimeout "$TIMEOUT_SECS" "$@"
  else
    "$@"
  fi
}

# Raw list models output (keeps "(current)")
list_models_raw() {
  "$CHATGPT_BIN" --list-models 2>/dev/null \
    | sed -E 's/\x1B\[[0-9;]*[mK]//g'
}

# Parse model ids from raw output.
# Accept lines like:
#   - gpt-4o-mini
#   * gpt-5 (current)
list_models() {
  list_models_raw \
    | awk '
        /^- / { sub(/^- /,""); print; next }
        /^\* / {
          sub(/^\* /,"")
          sub(/ \(current\).*$/,"")
          print
          next
        }
      ' \
    | sed -e 's/[[:space:]]*$//' \
    | grep -v '^$'
}

# Extract the "(current)" model from raw output.
pick_current_model() {
  list_models_raw \
    | awk '
        /^\* / {
          sub(/^\* /,"")
          sub(/ \(current\).*$/,"")
          print
          exit
        }
      '
}

normalize() {
  tr -d '\r' | sed -E 's/[[:space:]]+/ /g; s/^ +| +$//g' | tr '[:upper:]' '[:lower:]'
}

assert_pong_line() {
  # Accept any output that contains a line that is exactly "pong" (case-insensitive),
  # ignoring whitespace and other chatter.
  printf "%s\n" "$1" \
    | tr -d '\r' \
    | sed -E 's/^[[:space:]]+|[[:space:]]+$//g' \
    | tr '[:upper:]' '[:lower:]' \
    | grep -xq 'pong'
}

pass() { echo "✅ $*"; }
fail() { echo "❌ $*"; }

effort_for_model() {
  local model="$1"
  if echo "$model" | grep -qi 'gpt-5-pro'; then
    echo "high"
  else
    echo ""
  fi
}

pick_first() {
  # pick first model matching regex from list
  local pattern="$1"; shift
  printf '%s\n' "$@" | grep -E "$pattern" | head -n 1 || true
}

is_empty() { [[ -z "${1:-}" ]]; }

# Pick the newest YYYY-MM-DD variant by lexicographic sort (works for ISO dates).
pick_latest_dated() {
  local pattern="$1"; shift
  printf '%s\n' "$@" \
    | grep -E "$pattern" \
    | grep -v 'search-api' \
    | sort \
    | tail -n 1 || true
}

# Pick "latest family" preferring dated variants, then chat-latest, then plain name.
# Examples:
#   pick_latest_family "gpt-5" "${models[@]}"
#   pick_latest_family "gpt-4o" "${models[@]}"
#   pick_latest_family "o1" "${models[@]}"
pick_latest_family() {
  local family="$1"; shift

  # e.g. gpt-5-mini-2025-08-07, gpt-5.2-pro-2025-12-11, gpt-4o-2024-11-20, o1-2024-12-17
  local dated
  dated="$(pick_latest_dated "^${family}(-[a-z0-9.]+)*-[0-9]{4}-[0-9]{2}-[0-9]{2}$" "$@")"
  if [[ -n "$dated" ]]; then
    echo "$dated"
    return 0
  fi

  # e.g. gpt-5-chat-latest, gpt-5.2-chat-latest
  local chat_latest
  chat_latest="$(printf '%s\n' "$@" \
  | grep -E "^${family}(-[a-z0-9.]+)*-chat-latest$" \
  | grep -v 'search-api' \
  | head -n 1 || true)"
  if [[ -n "$chat_latest" ]]; then
    echo "$chat_latest"
    return 0
  fi

  # fallback: plain family name if present
  printf '%s\n' "$@" | grep -E "^${family}$" | head -n 1 || true
}

run_query() {
  local name="$1"
  local model="$2"
  shift 2

  local out status
  set +e
  local effort
  effort="$(effort_for_model "$model")"

  if [[ -n "$effort" ]]; then
    out="$(run_with_timeout "$CHATGPT_BIN" \
        --new-thread \
        --temperature 0 \
        --effort "$effort" \
        --model "$model" \
        --max-tokens "$MAX_TOKENS" \
        "$@" \
        --query "$PROMPT" 2>&1)"
  else
    out="$(run_with_timeout "$CHATGPT_BIN" \
        --new-thread \
        --temperature 0 \
        --model "$model" \
        --max-tokens "$MAX_TOKENS" \
        "$@" \
        --query "$PROMPT" 2>&1)"
  fi
  status=$?
  set -e

  _dump_failure() {
    local why="$1"
    fail "$name (model=$model) $why (exit=$status)"
    echo "----- RAW OUTPUT -----" >&2
    printf "%s\n" "$out" >&2
    echo "----------------------" >&2

    echo "----- DEBUG RERUN -----" >&2
    local dbg dbg_status
    set +e
    dbg="$(run_with_timeout "$CHATGPT_BIN" \
        --new-thread \
        --model "$model" \
        --max-tokens "$MAX_TOKENS" \
        "$@" \
        --debug \
        --query "$PROMPT" 2>&1)"
    dbg_status=$?
    set -e
    echo "(debug exit=$dbg_status)" >&2
    printf "%s\n" "$dbg" >&2
    echo "-----------------------" >&2
  }

  if [[ $status -ne 0 ]]; then
    _dump_failure "failed to run"
    return 1
  fi

  if [[ -z "$(printf "%s" "$out" | tr -d '[:space:]')" ]]; then
    _dump_failure "returned empty output"
    return 1
  fi

  if ! assert_pong_line "$out"; then
    _dump_failure "did not produce 'pong' as a standalone line"
    return 1
  fi

  pass "$name (model=$model)"
  return 0
}

main() {
  echo "chatgpt-cli smoke test (bucketed + current/latest)"
  echo "bin: $CHATGPT_BIN"
  echo "max_tokens: $MAX_TOKENS"
  echo "web: $SMOKE_WEB (context_size=$WEB_CONTEXT_SIZE)"
  echo

  local models=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && models+=("$line")
  done < <(list_models)

  if [[ "${#models[@]}" -eq 0 ]]; then
    echo "ERROR: no models found from --list-models"
    exit 2
  fi

  # ---- Bucket counts (roughly aligned with your GetCapabilities) ----
  local count_realtime count_search count_gpt5 count_o1
  count_realtime="$(printf '%s\n' "${models[@]}" | grep -c 'realtime' || true)"
  count_search="$(printf '%s\n' "${models[@]}" | grep -c -- '-search' || true)"
  count_gpt5="$(printf '%s\n' "${models[@]}" | grep -c '^gpt-5' || true)"
  count_o1="$(printf '%s\n' "${models[@]}" | grep -c '^o1' || true)"

  echo "Discovered ${#models[@]} model(s)"
  echo "  realtime:      $count_realtime"
  echo "  search:        $count_search"
  echo "  gpt-5*:        $count_gpt5"
  echo "  o1*:           $count_o1"
  echo

  # ---- Existing bucket probes (keep these) ----

  local m_completions
  m_completions="$(pick_first '^gpt-4o-mini$' "${models[@]}")"
  if is_empty "$m_completions"; then m_completions="$(pick_first '^gpt-4\.1-mini$' "${models[@]}")"; fi
  if is_empty "$m_completions"; then m_completions="$(pick_first '^gpt-4o$' "${models[@]}")"; fi

  local m_responses
  m_responses="$(pick_first '^gpt-5-mini$' "${models[@]}")"
  if is_empty "$m_responses"; then m_responses="$(pick_first '^gpt-5$' "${models[@]}")"; fi
  if is_empty "$m_responses"; then m_responses="$(pick_first '^gpt-5(\.|-).*' "${models[@]}")"; fi
  if is_empty "$m_responses"; then m_responses="$(pick_first '^o1-pro$' "${models[@]}")"; fi

  local m_search
  m_search="$(pick_first '^gpt-4o-mini-search-preview$' "${models[@]}")"
  if is_empty "$m_search"; then m_search="$(pick_first '^gpt-4o-search-preview$' "${models[@]}")"; fi
  if is_empty "$m_search"; then m_search="$(pick_first 'search' "${models[@]}")"; fi

  local m_o1
  m_o1="$(pick_first '^o1-mini$' "${models[@]}")"
  if is_empty "$m_o1"; then m_o1="$(pick_first '^o1$' "${models[@]}")"; fi

  local m_web=""
  if [[ "$SMOKE_WEB" == "true" ]]; then
    m_web="$(pick_first '^gpt-5-mini$' "${models[@]}")"
    if is_empty "$m_web"; then m_web="$(pick_first '^gpt-5$' "${models[@]}")"; fi
  fi

  # ---- New probes: current + latest releases ----

  local m_current
  m_current="$(pick_current_model || true)"

  local m_latest_gpt5
  m_latest_gpt5="$(pick_latest_family 'gpt-5' "${models[@]}")"

  local m_latest_4o
  m_latest_4o="$(pick_latest_family 'gpt-4o' "${models[@]}")"

  local m_latest_o1
  m_latest_o1="$(pick_latest_family 'o1' "${models[@]}")"

  local failures=0
  local ran=0

  # Existing probes
  if is_empty "$m_completions"; then
    echo "WARN: no completions-ish model found (skipping)"
  else
    ran=$((ran+1))
    run_query "probe:completions" "$m_completions" || failures=$((failures+1))
  fi

  if is_empty "$m_responses"; then
    echo "WARN: no responses-ish model found (skipping)"
  else
    ran=$((ran+1))
    run_query "probe:responses" "$m_responses" || failures=$((failures+1))
  fi

  if is_empty "$m_search"; then
    echo "WARN: no search-preview model found (skipping)"
  else
    ran=$((ran+1))
    run_query "probe:search-preview" "$m_search" || failures=$((failures+1))
  fi

  if is_empty "$m_o1"; then
    echo "WARN: no o1 model found (skipping)"
  else
    ran=$((ran+1))
    run_query "probe:o1" "$m_o1" || failures=$((failures+1))
  fi

  if [[ "$SMOKE_WEB" == "true" ]]; then
    if is_empty "$m_web"; then
      echo "WARN: no gpt-5 model found for web probe (skipping)"
    else
      ran=$((ran+1))
      run_query "probe:web" "$m_web" --web true --web-context-size "$WEB_CONTEXT_SIZE" || failures=$((failures+1))
    fi
  fi

  # New probes
  if is_empty "$m_current"; then
    echo "WARN: could not detect '(current)' model (skipping probe:current)"
  else
    ran=$((ran+1))
    run_query "probe:current" "$m_current" || failures=$((failures+1))
  fi

  # Only run latest probes if they add coverage (avoid duplicates).
  if ! is_empty "$m_latest_gpt5" && [[ "$m_latest_gpt5" != "$m_responses" ]]; then
    ran=$((ran+1))
    run_query "probe:latest-gpt5" "$m_latest_gpt5" || failures=$((failures+1))
  fi

  if ! is_empty "$m_latest_4o" && [[ "$m_latest_4o" != "$m_completions" ]]; then
    ran=$((ran+1))
    run_query "probe:latest-4o" "$m_latest_4o" || failures=$((failures+1))
  fi

  if ! is_empty "$m_latest_o1" && [[ "$m_latest_o1" != "$m_o1" ]]; then
    ran=$((ran+1))
    run_query "probe:latest-o1" "$m_latest_o1" || failures=$((failures+1))
  fi

  echo
  echo "Ran $ran probe(s)."

  if [[ "$failures" -gt 0 ]]; then
    echo "Smoke test finished with $failures failure(s)."
    exit 1
  fi

  echo "Smoke test passed."
}

main "$@"
