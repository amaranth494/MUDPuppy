#!/usr/bin/env bash
#
# scripts/verify-phase3.sh -- Phase 3 canned-report harness (AI Player: one
# AI decision).
#
# Drives the HTTP-reachable half of Phase 3 -- POST /api/v1/session/autopilot
# (the D-20 configuration gate), GET /api/v1/session/status,
# GET /api/v1/profiles/{connection_id}/decisions,
# GET /api/v1/profiles/{connection_id}/sessions[/{session_id}], and
# POST /api/v1/icm/validate -- against a running MUDPuppy server, plus a
# source grep proving no model identifier is a literal in Go source, and
# writes a report file that states, criterion by criterion (C1..C4, matching
# ROADMAP.md's Phase 3 Success Criteria 1-4), whether Phase 3's reachable
# half holds, with the raw request path and response body printed under
# every step. This IS the canned report the owner's evidence rule requires;
# no database query is used anywhere in this script.
#
# Three ground rules a reader needs:
#   1. Proof in this project is a canned report or an end-user screenshot,
#      never a database query.
#   2. C1 (the live decision itself) and the live half of C2 (the panel and
#      terminal actually showing a decision as it happens) are out of this
#      script's scope: engaging autopilot with a real decision requires a
#      connected game session and a configured model, which this script does
#      not and must not fabricate. They are printed as SKIP lines naming the
#      screenshot(s) and log excerpt that prove them instead -- never as
#      PASS.
#   3. There are four invocations (run with no arguments to print full
#      usage):
#
#   1. Live run, against any running server:
#        BASE_URL=https://mudpuppy-staging.up.railway.app \
#        SESSION_COOKIE="session=abc123..." \
#        CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
#        scripts/verify-phase3.sh evidence/03-canned-report.txt
#
#   2. Self-test, no server, proves the assertion logic against
#      scripts/fixtures/phase3/ (a correct-server fixture set):
#        scripts/verify-phase3.sh --self-test /tmp/phase3-selftest.txt
#
#   3. Negative self-test, proves the harness can FAIL against
#      scripts/fixtures/phase3-negative/ (one deliberately wrong response):
#        scripts/verify-phase3.sh --self-test-negative /tmp/phase3-negative.txt
#
#   4. Skip the appended `go test` runs (live mode only):
#        BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... \
#        scripts/verify-phase3.sh --no-tests report.txt
#
# CONNECTION_ID must name a profile that HAS accepted the Safety and Abuse
# policy, so the Phase 1 gate passes and the Phase 3 configuration gate
# (D-20) is the one under test in the C3 steps below. NOT_OWNED_CONNECTION_ID
# is optional and defaults to a fixed literal UUID that no profile will ever
# have -- a connection id the caller does not own and one that does not
# exist are indistinguishable on the wire, and both must be refused.
#
# Self-test modes need no server and no repository state: the source-grep
# step and the appended go test runs are both skipped in --self-test and
# --self-test-negative, since fixtures are the subject of those runs, not
# this repository's Go source or test suite.
#
set -uo pipefail   # pipefail only; errexit is intentionally omitted -- a
                    # failed check must not abort the report -- the report
                    # must list every result, not stop at the first FAIL.

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
SELF_TEST=0
SELF_TEST_NEGATIVE=0
NO_TESTS=0
REPORT=""

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/verify-phase3.sh [--self-test|--self-test-negative] [--no-tests] <report-file>

Drives the Phase 3 AI Player one-decision HTTP sequence and writes
<report-file> with a PASS/FAIL/SKIP line per ROADMAP criterion (C1..C4) and
every request/response body underneath.

Live run reads its inputs from the environment:

  BASE_URL                 server root, e.g. https://mudpuppy-staging.up.railway.app
  SESSION_COOKIE            full Cookie header value of an authenticated session
  CONNECTION_ID             connection_id of a profile that HAS accepted the
                            Safety and Abuse policy, so the Phase 1 gate
                            passes and the Phase 3 configuration gate (D-20)
                            is the one under test.
  NOT_OWNED_CONNECTION_ID   optional; connection_id the caller does not own.
                            Defaults to a fixed literal UUID that no profile
                            will ever have.

  BASE_URL=https://mudpuppy-staging.up.railway.app \
  SESSION_COOKIE="session=abc123..." \
  CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
  scripts/verify-phase3.sh evidence/03-canned-report.txt

--self-test          run against scripts/fixtures/phase3/ instead of a live
                      server; no environment variables required.
--self-test-negative  run against scripts/fixtures/phase3-negative/ (one
                      deliberately wrong response) to prove the harness
                      FAILs and exits non-zero on a wrong server response.
--no-tests            skip the appended `go test` runs (live mode only;
                      self-test modes always skip them).
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --self-test) SELF_TEST=1 ;;
    --self-test-negative) SELF_TEST_NEGATIVE=1 ;;
    --no-tests) NO_TESTS=1 ;;
    --*)
      echo "Unknown flag: $arg" >&2
      usage
      exit 2
      ;;
    *) REPORT="$arg" ;;
  esac
done

if [ -z "$REPORT" ]; then
  usage
  exit 2
fi

if [ "$SELF_TEST" -eq 1 ] && [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "--self-test and --self-test-negative are mutually exclusive" >&2
  exit 2
fi

FIXTURE_DIR=""
if [ "$SELF_TEST" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase3"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase3-negative"
fi

if [ -z "$FIXTURE_DIR" ]; then
  if [ -z "${BASE_URL:-}" ] || [ -z "${SESSION_COOKIE:-}" ] || [ -z "${CONNECTION_ID:-}" ]; then
    usage
    exit 2
  fi
else
  # Self-test modes need no live inputs; harmless placeholders keep the
  # header block and _http's live-path arguments well-defined even though
  # that path is never reached in these modes.
  BASE_URL="${BASE_URL:-http://fixture.invalid}"
  SESSION_COOKIE="${SESSION_COOKIE:-}"
  CONNECTION_ID="${CONNECTION_ID:-00000000-0000-0000-0000-000000000000}"
fi

# A connection id the caller does not own and one that does not exist are
# indistinguishable on the wire (T-3-03/T-3-04); the default below is
# deliberate -- no profile will ever be seeded with this id.
NOT_OWNED_CONNECTION_ID="${NOT_OWNED_CONNECTION_ID:-ffffffff-ffff-ffff-ffff-ffffffffffff}"

# ---------------------------------------------------------------------------
# Report file: everything printed from here on goes to both stdout and the
# report file, so the file is a verbatim transcript of this run.
# ---------------------------------------------------------------------------
exec > >(tee "$REPORT") 2>&1

RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================================================"
echo "Phase 3 canned-report harness -- scripts/verify-phase3.sh"
echo "Run started (UTC): $RUN_TS"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "Mode: --self-test (fixtures: scripts/fixtures/phase3/, no server)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "Mode: --self-test-negative (fixtures: scripts/fixtures/phase3-negative/, no server)"
else
  echo "Mode: live"
fi
echo "BASE_URL: $BASE_URL"
echo "CONNECTION_ID: $CONNECTION_ID"
echo "NOT_OWNED_CONNECTION_ID: $NOT_OWNED_CONNECTION_ID"
echo "git rev-parse --short HEAD: $GIT_SHA"
echo "================================================================"
echo
echo "Ground rules: (1) proof here is this report or an end-user screenshot,"
echo "never a database query. (2) C1 and the live half of C2 are"
echo "live-Gemini/live-MUD behaviours this script cannot reach without"
echo "fabricating a connected game and a configured model; they appear below"
echo "as SKIP lines naming their screenshot/log evidence, never as PASS."
echo "(3) invocations: live, --self-test, --self-test-negative, --no-tests."
echo

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# _get_field <json> <field>
# Extracts one field's value from a flat JSON object. Prints "null" for a
# JSON null. Prints the empty string via the non-jq fallback when the key is
# altogether absent from the text (e.g. an `omitempty` field that was
# omitted).
_get_field() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r --arg f "$field" '
      (getpath($f | split("."))) as $v
      | if $v == null then "null"
        elif ($v | type) == "string" then $v
        else ($v | tostring)
        end' 2>/dev/null
  else
    local leaf="${field##*.}"
    printf '%s' "$json" \
      | grep -oE "\"$leaf\"[[:space:]]*:[[:space:]]*(\"[^\"]*\"|null|true|false|-?[0-9]+(\.[0-9]+)?)" \
      | head -n 1 \
      | sed -E "s/^\"$leaf\"[[:space:]]*:[[:space:]]*//" \
      | sed -E 's/^"(.*)"$/\1/'
  fi
}

# Ordered list of fixture files, one per _http call in the sequence below.
# Used only when FIXTURE_DIR is set. The fixtures always represent a
# non-empty decisions/sessions history (D-12, D-17) so the sequence of
# _http calls is identical in both self-test modes and in live mode against
# a walked-through profile -- only live mode against a brand-new profile may
# take fewer calls (an empty sessions list skips the transcript-read call),
# which is harmless because fixture indexing is never consulted outside
# FIXTURE_DIR being set.
FIXTURE_FILES=(
  "01-autopilot-on-refused-not-configured.json"
  "02-status-agrees.json"
  "03-decisions-own.json"
  "04-decisions-not-owned.json"
  "05-sessions-own.json"
  "06-sessions-transcript.json"
  "07-icm-validate.json"
)
_HTTP_CALL_NO=0

# _http <method> <path> [body]
# Performs a live HTTP call against ${BASE_URL}/api/v1/<path> using the
# supplied session cookie, OR -- when FIXTURE_DIR is set (--self-test /
# --self-test-negative) -- reads the next fixture file in FIXTURE_FILES
# instead of calling curl at all. Either way, sets HTTP_STATUS and HTTP_BODY
# as globals; every line below this function (the assertions, the printing,
# the counters, the summary and the exit code) is identical in both modes,
# so the self-test exercises the real assertion logic, not a copy of it.
_http() {
  local method="$1" path="$2" body="${3:-}"
  _HTTP_CALL_NO=$((_HTTP_CALL_NO + 1))
  if [ -n "$FIXTURE_DIR" ]; then
    local fname="${FIXTURE_FILES[$((_HTTP_CALL_NO - 1))]:-}"
    local fixture="$FIXTURE_DIR/$fname"
    if [ -z "$fname" ] || [ ! -f "$fixture" ]; then
      echo "FIXTURE MISSING for call #$_HTTP_CALL_NO ($method $path): ${fixture:-<none>}" >&2
      HTTP_STATUS="000"
      HTTP_BODY='{"error":"fixture missing"}'
      return 0
    fi
    HTTP_STATUS=$(head -n 1 "$fixture")
    HTTP_BODY=$(tail -n +2 "$fixture")
    return 0
  fi
  local resp
  if [ -n "$body" ]; then
    resp=$(curl -sS -X "$method" -b "$SESSION_COOKIE" -H 'Content-Type: application/json' \
      -d "$body" -w '\n%{http_code}' "${BASE_URL}/api/v1/${path}")
  else
    resp=$(curl -sS -X "$method" -b "$SESSION_COOKIE" -H 'Content-Type: application/json' \
      -w '\n%{http_code}' "${BASE_URL}/api/v1/${path}")
  fi
  HTTP_STATUS=$(printf '%s' "$resp" | tail -n 1)
  HTTP_BODY=$(printf '%s' "$resp" | sed '$d')
}

FAIL_COUNT=0
TOTAL_COUNT=0
SKIP_COUNT=0
declare -A CRIT_FAILED
declare -A CRIT_SEEN
declare -A CRIT_SKIPPED

# _check <label> <description> <0-for-pass|nonzero-for-fail>
# Prints a line of the form "PASS C3: <description> (got: ...)" or
# "FAIL C3: <description> (expected: ..., got: ...)" for every assertion.
_check() {
  local label="$1" desc="$2" result="$3"
  TOTAL_COUNT=$((TOTAL_COUNT + 1))
  CRIT_SEEN[$label]=1
  if [ "$result" -eq 0 ]; then
    echo "PASS $label: $desc"
  else
    echo "FAIL $label: $desc"
    FAIL_COUNT=$((FAIL_COUNT + 1))
    CRIT_FAILED[$label]=1
  fi
}

# _check_eq <label> <desc> <actual> <expected>
_check_eq() {
  local label="$1" desc="$2" actual="$3" expected="$4"
  if [ "$actual" = "$expected" ]; then
    _check "$label" "$desc (got: $actual)" 0
  else
    _check "$label" "$desc (expected: $expected, got: $actual)" 1
  fi
}

# _check_status <label> <desc> <expected-http-status>
_check_status() {
  local label="$1" desc="$2" expected_status="$3"
  if [ "$HTTP_STATUS" = "$expected_status" ]; then
    _check "$label" "$desc (status $HTTP_STATUS)" 0
  else
    _check "$label" "$desc (expected status $expected_status, got $HTTP_STATUS)" 1
  fi
}

# _check_not_status <label> <desc> <forbidden-http-status>
_check_not_status() {
  local label="$1" desc="$2" forbidden_status="$3"
  if [ "$HTTP_STATUS" != "$forbidden_status" ]; then
    _check "$label" "$desc (status $HTTP_STATUS)" 0
  else
    _check "$label" "$desc (expected anything other than $forbidden_status, got $HTTP_STATUS)" 1
  fi
}

# _skip <label> <reason-and-evidence>
# Prints "SKIP C1: <text>" and increments the skip counter. A skip is
# neither a pass nor a failure and must never be counted as either.
_skip() {
  local label="$1" desc="$2"
  SKIP_COUNT=$((SKIP_COUNT + 1))
  CRIT_SKIPPED[$label]=1
  echo "SKIP $label: $desc"
}

_step() {
  echo
  echo "---- $* ----"
}

# _print_response <method> <display-path>
# Prints the raw response body whole.
_print_response() {
  local method="$1" path="$2"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$HTTP_BODY"
}

D20_MESSAGE="Autopilot refused: AI is not configured on this server"

# ---------------------------------------------------------------------------
# Step 1 -- POST session/autopilot action=on -- the D-20 configuration gate,
# the heart of this report (C3)
# ---------------------------------------------------------------------------
_step "Step 1: POST session/autopilot action=on -- the D-20 configuration gate (C3)"
_http POST "session/autopilot" "{\"action\":\"on\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
OUTCOME_1=$(_get_field "$HTTP_BODY" outcome)
STATE_1=$(_get_field "$HTTP_BODY" state)
GATE_MESSAGE_1=$(_get_field "$HTTP_BODY" gate_message)
if [ -n "$FIXTURE_DIR" ]; then
  # Self-test modes fix the world: the fixtures represent a server with the
  # Gemini variables unset, so the correct answer is always
  # refused-not-configured. The negative fixture set deliberately swaps
  # this one response for outcome=engaged/state=on to prove the harness
  # catches it (T-3-10).
  _check_eq C3 "outcome is refused-not-configured (Gemini variables unset on the target server)" "$OUTCOME_1" "refused-not-configured"
  _check_eq C3 "gate_message matches the exact D-20 refusal sentence" "$GATE_MESSAGE_1" "$D20_MESSAGE"
  _check_eq C3 "state stays off on refusal" "$STATE_1" "off"
elif [ "$OUTCOME_1" = "refused-not-configured" ]; then
  _check_eq C3 "outcome is refused-not-configured (Gemini variables unset on the target server)" "$OUTCOME_1" "refused-not-configured"
  _check_eq C3 "gate_message matches the exact D-20 refusal sentence" "$GATE_MESSAGE_1" "$D20_MESSAGE"
  _check_eq C3 "state stays off on refusal" "$STATE_1" "off"
  echo "NOTE: this server ran with the Gemini variables UNSET -- the D-20 refusal is the Phase 3 behaviour this run demonstrates."
elif [ "$OUTCOME_1" = "engaged" ] || [ "$OUTCOME_1" = "refused-no-session" ] || [ "$OUTCOME_1" = "refused-wrong-connection" ]; then
  # refused-wrong-connection (Phase 2 code review C2): the owner's browser is
  # connected to a different profile right now. The configuration gate was
  # passed before that check, so it is the "variables set" world too.
  _check C3 "the configuration gate passed (outcome: $OUTCOME_1) -- this server has the Gemini variables set" 0
  echo "NOTE: this server ran with the Gemini variables SET -- the D-20 refusal cannot be demonstrated on this run; see evidence/08-refused-not-configured.png for that world instead."
else
  _check C3 "outcome is refused-not-configured, engaged, refused-no-session or refused-wrong-connection (got: $OUTCOME_1)" 1
fi

# ---------------------------------------------------------------------------
# Step 2 -- GET session/status -- agrees with step 1's switch position (C3)
# ---------------------------------------------------------------------------
_step "Step 2: GET session/status -- autopilot_state agrees with step 1 (C3)"
_http GET "session/status"
_print_response GET "session/status"
AUTOPILOT_2=$(_get_field "$HTTP_BODY" autopilot_state)
_check_eq C3 "autopilot_state agrees with step 1's state" "$AUTOPILOT_2" "$STATE_1"

# ---------------------------------------------------------------------------
# Step 3 -- source grep: no model identifier is a literal in Go source
# (REQ-env-config). Skipped in self-test modes -- fixtures are the subject
# of those runs, not this repository's Go source.
# ---------------------------------------------------------------------------
_step "Step 3: source grep -- no model identifier is a literal in Go source (C3, REQ-env-config)"
if [ "$SELF_TEST" -eq 1 ] || [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "SKIPPED (self-test mode: fixtures are the subject of this run, not this repository's Go source)"
else
  GREP_CMD='grep -rniE "gemini-[0-9]|flash-latest|generativelanguage" internal/ cmd/ --include=*.go | grep -v "_test.go"'
  echo "\$ $GREP_CMD"
  GREP_OUTPUT=$(grep -rniE "gemini-[0-9]|flash-latest|generativelanguage" internal/ cmd/ --include=*.go 2>/dev/null | grep -v "_test.go")
  echo "${GREP_OUTPUT:-<no output>}"
  if [ -z "$GREP_OUTPUT" ]; then
    _check C3 "no model identifier literal found outside test files" 0
  else
    _check C3 "no model identifier literal found outside test files" 1
  fi
fi

# ---------------------------------------------------------------------------
# Step 4 -- GET profiles/$CONNECTION_ID/decisions -- the stored half of
# reasoning visibility (C2, D-12)
# ---------------------------------------------------------------------------
_step "Step 4: GET profiles/\$CONNECTION_ID/decisions -- survives a refresh, no query needed (C2)"
_http GET "profiles/${CONNECTION_ID}/decisions"
_print_response GET "profiles/\$CONNECTION_ID/decisions"
_check_status C2 "decisions endpoint answers HTTP 200 for the owning connection" "200"
DECISIONS_RAW=$(printf '%s' "$HTTP_BODY" | grep -oE '"decisions"[[:space:]]*:[[:space:]]*\[' || true)
if [ -n "$DECISIONS_RAW" ]; then
  _check C2 "response carries a decisions array" 0
else
  _check C2 "response carries a decisions array" 1
fi
# The endpoint returns a connection's decisions OLDEST first (plan 03-09),
# and a failed decision carries an empty reasoning and command by design
# (D-13). The entry that must carry reasoning, command and outcome is the
# newest decision whose outcome is "sent".
if command -v jq >/dev/null 2>&1; then
  NEWEST_REASONING=$(printf '%s' "$HTTP_BODY" | jq -r '[.decisions[]? | select(.outcome == "sent")] | last | .reasoning // ""' 2>/dev/null)
  NEWEST_COMMAND=$(printf '%s' "$HTTP_BODY" | jq -r '[.decisions[]? | select(.outcome == "sent")] | last | .command // ""' 2>/dev/null)
  NEWEST_OUTCOME=$(printf '%s' "$HTTP_BODY" | jq -r '[.decisions[]? | select(.outcome == "sent")] | last | .outcome // ""' 2>/dev/null)
else
  # No jq: take the last non-empty reasoning and command in document order
  # (oldest first, so the last one is the newest) and the last "sent" outcome.
  NEWEST_REASONING=$(printf '%s' "$HTTP_BODY" | grep -oE '"reasoning"[[:space:]]*:[[:space:]]*"[^"]+"' | tail -n 1 | sed -E 's/^"reasoning"[[:space:]]*:[[:space:]]*"(.*)"$/\1/')
  NEWEST_COMMAND=$(printf '%s' "$HTTP_BODY" | grep -oE '"command"[[:space:]]*:[[:space:]]*"[^"]+"' | tail -n 1 | sed -E 's/^"command"[[:space:]]*:[[:space:]]*"(.*)"$/\1/')
  NEWEST_OUTCOME=$(printf '%s' "$HTTP_BODY" | grep -oE '"outcome"[[:space:]]*:[[:space:]]*"sent"' | tail -n 1 | sed -E 's/.*"(sent)"$/\1/')
fi
if [ -n "$NEWEST_REASONING" ] && [ "$NEWEST_REASONING" != "null" ] && [ -n "$NEWEST_COMMAND" ] && [ "$NEWEST_OUTCOME" = "sent" ]; then
  _check C2 "the newest sent decision carries reasoning, command and outcome (reasoning: $NEWEST_REASONING | command: $NEWEST_COMMAND | outcome: $NEWEST_OUTCOME)" 0
else
  echo "NOTE: no sent decision for this connection yet (no walkthrough run, or only failed attempts) -- presence and well-formedness were still checked above; the newest-sent-entry shape check is a no-op here."
fi

# ---------------------------------------------------------------------------
# Step 5 -- GET profiles/$NOT_OWNED_CONNECTION_ID/decisions -- refused, not
# answered with someone else's rows (C2, T-3-03)
# ---------------------------------------------------------------------------
_step "Step 5: GET profiles/\$NOT_OWNED_CONNECTION_ID/decisions -- refused, not someone else's rows (C2, T-3-03)"
_http GET "profiles/${NOT_OWNED_CONNECTION_ID}/decisions"
_print_response GET "profiles/\$NOT_OWNED_CONNECTION_ID/decisions"
_check_not_status C2 "a not-owned/non-existent connection_id is refused, never answered with HTTP 200 rows" "200"

# ---------------------------------------------------------------------------
# Step 6 -- GET profiles/$CONNECTION_ID/sessions -- the transcript half of
# the session log (C2, D-14, D-17)
# ---------------------------------------------------------------------------
_step "Step 6: GET profiles/\$CONNECTION_ID/sessions -- the transcript half of the session log (C2)"
_http GET "profiles/${CONNECTION_ID}/sessions"
_print_response GET "profiles/\$CONNECTION_ID/sessions"
_check_status C2 "sessions endpoint answers HTTP 200 for the owning connection" "200"
SESSIONS_RAW=$(printf '%s' "$HTTP_BODY" | grep -oE '"sessions"[[:space:]]*:[[:space:]]*\[' || true)
if [ -n "$SESSIONS_RAW" ]; then
  _check C2 "response carries a sessions array" 0
else
  _check C2 "response carries a sessions array" 1
fi
NEWEST_SESSION_ID=$(_get_field "$HTTP_BODY" sessions.0.id)

if [ -n "$NEWEST_SESSION_ID" ] && [ "$NEWEST_SESSION_ID" != "null" ]; then
  _step "Step 6b: GET profiles/\$CONNECTION_ID/sessions/\$NEWEST_SESSION_ID -- transcript lines are tagged human/ai/game/marker (C2)"
  _http GET "profiles/${CONNECTION_ID}/sessions/${NEWEST_SESSION_ID}"
  _print_response GET "profiles/\$CONNECTION_ID/sessions/\$NEWEST_SESSION_ID"
  _check_status C2 "session transcript read answers HTTP 200" "200"
  LINES_RAW=$(printf '%s' "$HTTP_BODY" | grep -oE '"lines"[[:space:]]*:[[:space:]]*\[' || true)
  if [ -n "$LINES_RAW" ]; then
    _check C2 "response carries a lines array" 0
  else
    _check C2 "response carries a lines array" 1
  fi
  LINE0_SOURCE=$(_get_field "$HTTP_BODY" lines.0.source)
  case "$LINE0_SOURCE" in
    human|ai|game|marker)
      _check C2 "transcript lines carry a source in human|ai|game|marker (got: $LINE0_SOURCE)" 0
      ;;
    "")
      echo "NOTE: transcript has no lines yet -- the array-presence check above already ran; the per-line source check is a no-op here."
      ;;
    *)
      _check C2 "transcript lines carry a source in human|ai|game|marker (got: $LINE0_SOURCE)" 1
      ;;
  esac
else
  echo "NOTE: sessions array is empty for this connection -- no game connection has ever been made on it, so the per-session transcript read is skipped. This is not a fixture path in self-test modes, which always seed one session."
fi

# ---------------------------------------------------------------------------
# Step 7 -- the live half of C2 (panel/terminal showing a decision as it
# happens) is out of this script's reach
# ---------------------------------------------------------------------------
_step "Step 7: C2 (live half) -- out of this script's reach"
_skip C2 "the live half of reasoning visibility requires a connected game session and a configured model; proven instead by evidence/05-decision-in-panel.png and evidence/07-panel-after-refresh.png"

# ---------------------------------------------------------------------------
# Step 8 -- POST /api/v1/icm/validate -- the ICM routes that have never
# existed now answer (C4)
# ---------------------------------------------------------------------------
_step "Step 8: POST /api/v1/icm/validate -- the ICM routes now answer (C4)"
_http POST "icm/validate" '{"raw":"north","sessionId":"verify-phase3"}'
_print_response POST "icm/validate"
_check_not_status C4 "the ICM validate route answers with anything other than 404 -- before this phase it did not exist at all" "404"

# ---------------------------------------------------------------------------
# Step 9 -- go test: the diagnostic proof that a plain automation-context
# command is genuinely dispatched through the ICM engine's dispatcher and
# safety checker (C4). Skipped in self-test modes and when --no-tests is
# given.
# ---------------------------------------------------------------------------
_step "Step 9: go test -- ICM dispatch and driver-engage diagnostics (C4)"
GO_TEST_EXIT=0
if [ "$SELF_TEST" -eq 1 ] || [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "SKIPPED (self-test mode: fixtures are the subject of this run, not the Go suite)"
elif [ "$NO_TESTS" -eq 1 ]; then
  echo "SKIPPED (--no-tests)"
else
  REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo .)
  echo "\$ go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v"
  ( cd "$REPO_ROOT" && go test ./internal/icm/... -run TestDispatch_AutomationPassThrough -v )
  ICM_TEST_EXIT=$?
  echo "\$ go test ./internal/driver/... -run TestHandleEngage -v"
  ( cd "$REPO_ROOT" && go test ./internal/driver/... -run TestHandleEngage -v )
  DRIVER_TEST_EXIT=$?
  if [ "$ICM_TEST_EXIT" -eq 0 ]; then
    _check C4 "TestDispatch_AutomationPassThrough exits 0" 0
  else
    _check C4 "TestDispatch_AutomationPassThrough exits 0" 1
  fi
  if [ "$DRIVER_TEST_EXIT" -eq 0 ]; then
    _check C4 "TestHandleEngage exits 0" 0
  else
    _check C4 "TestHandleEngage exits 0" 1
  fi
  if [ "$ICM_TEST_EXIT" -ne 0 ] || [ "$DRIVER_TEST_EXIT" -ne 0 ]; then
    GO_TEST_EXIT=1
  fi
fi

# ---------------------------------------------------------------------------
# Step 10 -- C1, the live decision itself, is out of this script's reach
# ---------------------------------------------------------------------------
_step "Step 10: C1 -- out of this script's reach"
_skip C1 "engaging autopilot with a live decision requires a connected game session and a configured model; this script must not fabricate one. Proven instead by evidence/06-ai-assist-terminal-line.png and the stage=request, stage=dispatch and stage=sent lines of evidence/04-staging-ai-player.log"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "================================================================"
echo "SUMMARY"
echo "================================================================"
for label in C1 C2 C3 C4; do
  if [ "${CRIT_FAILED[$label]:-}" = "1" ]; then
    echo "$label: FAIL"
  elif [ "${CRIT_SEEN[$label]:-}" = "1" ]; then
    echo "$label: PASS"
  elif [ "${CRIT_SKIPPED[$label]:-}" = "1" ]; then
    echo "$label: SKIP"
  else
    echo "$label: NOT EXERCISED"
  fi
done
echo "TOTAL CHECKS: $TOTAL_COUNT  FAILURES: $FAIL_COUNT  SKIPPED: $SKIP_COUNT"

echo
echo "================================================================"
echo "GO TEST EXIT: $GO_TEST_EXIT"
echo "================================================================"

if [ "$FAIL_COUNT" -eq 0 ] && [ "$GO_TEST_EXIT" -eq 0 ]; then
  exit 0
else
  exit 1
fi
