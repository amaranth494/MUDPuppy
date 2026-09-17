#!/usr/bin/env bash
#
# scripts/verify-phase4.sh -- Phase 4 canned-report harness (AI Player:
# continuous play).
#
# Drives the HTTP-reachable half of Phase 4 -- the session goal round trip
# and its Quest reactivate-vs-create (D-01, D-03, D-04), Session Memory
# read-back on attach (D-10), and the captured-text retention delete with
# its counts and the decision audit trail's survival (D-21) -- against a
# running MUDPuppy server, and writes a report file that states, criterion
# by criterion (C1..C5, fixed in 04-CONTEXT.md/04-10-PLAN.md), whether
# Phase 4's reachable half holds, with the raw request path and response
# body printed under every step. This IS the canned report the owner's
# evidence rule requires; no database query is used anywhere in this
# script.
#
# !! A LIVE RUN IS DESTRUCTIVE !!  Step 11 issues the real
# DELETE profiles/$CONNECTION_ID/captured-text. It PERMANENTLY DELETES the
# captured game text of that connection -- every session transcript line on
# the Logs page and the game-text snapshot on every AI decision row -- and
# nothing can bring it back. There is no confirmation prompt and no opt-out:
# criterion C4's proof IS that delete, run end to end (accepted as threat
# T-4-15; the report records the counts it removed). Decision rows
# themselves, their reasoning and outcomes, Session Memory and Quest Memory
# are not touched. Point CONNECTION_ID at a connection whose captured text
# you are content to lose. The two self-test modes touch nothing.
#
# What a live run changes, in full:
#   * captured text for CONNECTION_ID: deleted, permanently (above);
#   * the session goal: changed during the run and put back at the end, and
#     put back by an exit trap if the run is interrupted (T-4-33);
#   * Quests: two new active Quest rows, one per harness goal, which stay
#     (nothing in Phase 4 closes a Quest, D-04).
# So a live run does NOT leave the profile exactly as it found it; it leaves
# the GOAL exactly as it found it.
#
# This is scripts/verify-phase3-1.sh's own shape, pointed at this phase's
# three endpoints -- same argument parsing, same fixture-or-curl request
# helper, same flat-JSON field extractor, same PASS/FAIL/SKIP-per-criterion
# report, same restore-what-it-changed discipline.
#
# Criterion labels (04-10-PLAN.md's context table):
#   C1  The session goal round trip, the goal edit, and the Quest being
#       reactivated rather than duplicated (D-01, D-03, D-04). Reachable.
#   C2  Session Memory read back on attach (D-10). Reachable.
#   C3  The mechanical limits: cap halt, threshold disengage,
#       blocked-repeatedly, blank settings (D-14 to D-17). NOT reachable --
#       this script prints exactly:
#         SKIP C3: ... proven instead by
#         `go test ./internal/driver/... -run "TestLoop_CallCap|TestLoop_ErrorThreshold|TestLoop_ConsecutiveBlocks|TestLoop_BlankSettings" -count=1 -v`
#         and evidence/08-cap-halt-badge-off.png
#   C4  The retention standard: the delete action's counts, and the
#       decision record still returning its reasoning and outcome
#       afterwards (D-21). Reachable.
#   C5  Wheel-grab and the re-engage reassessment (D-08). NOT reachable --
#       this script prints exactly:
#         SKIP C5: ... proven instead by
#         `go test ./internal/driver/... -run "TestLoop_|TestEngageLoop_" -count=1 -v`
#         and evidence/09-wheel-grab-reengage-reasoning.png
#
# Three ground rules a reader needs:
#   1. Proof in this project is a canned report or an end-user screenshot,
#      never a database query.
#   2. C3 and C5 are out of this script's reach: the loop's own pacing, the
#      cap and threshold halts and the wheel-grab re-engage reassessment
#      cannot be driven by an HTTP client at all. They are printed below as
#      SKIP lines naming the exact `go test` invocation and the screenshot
#      filename that prove each instead -- never as PASS, never as FAIL.
#   3. There are four invocations (run with no arguments to print full
#      usage):
#
#   1. Live run, against any running server (DESTRUCTIVE, see above):
#        BASE_URL=https://mudpuppy-staging.up.railway.app \
#        SESSION_COOKIE="session_token=abc123..." \
#        CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
#        scripts/verify-phase4.sh evidence/03-canned-report.txt
#
#   2. Self-test, no server, proves the assertion logic against
#      scripts/fixtures/phase4/ (a correct-server fixture set):
#        scripts/verify-phase4.sh --self-test /tmp/phase4-selftest.txt
#
#   3. Negative self-test, proves the harness can FAIL against
#      scripts/fixtures/phase4-negative/ (one deliberately wrong response)
#      so the harness is trusted before its report is (T-4-11):
#        scripts/verify-phase4.sh --self-test-negative /tmp/phase4-negative.txt
#
#   4. Skip the appended `go test` runs (live mode only):
#        BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... \
#        scripts/verify-phase4.sh --no-tests report.txt
#
# Environment variables (live mode):
#   BASE_URL                 server root, e.g. https://mudpuppy-staging.up.railway.app
#   SESSION_COOKIE            full Cookie header value of an authenticated session
#   CONNECTION_ID             connection_id of a profile the caller owns
#   NOT_OWNED_CONNECTION_ID   optional; connection_id the caller does not own.
#                             Defaults to a fixed literal UUID that no profile
#                             will ever have.
#   HTTP_MAX_TIME             optional; seconds any one HTTP call may take
#                             before it is given up as status 000. Default 30.
#
# CONNECTION_ID must name a profile the caller owns. A connection id the
# caller does not own and one that does not exist are indistinguishable on
# the wire (T-4-09, the Phase 3/3.1 IDOR check re-run again on these three
# new endpoints), and both must be refused.
#
# This harness never issues a live Gemini call and never deploys anything.
# It never reads the session cookie, an API key or any game text back into
# the report it writes (T-4-05). Step 1 below reads the profile's current
# goal and keeps it decoded (not re-escaped) so the restore step at the end
# can PUT it back exactly as found (T-4-33) -- this is the Phase 3.1
# double-escaping defect (03.1-07-SUMMARY.md) deliberately not repeated
# here. Two rules protect the owner's real goal (code review WR-11):
#   * if step 1 cannot read the goal (anything but HTTP 200), the run ABORTS
#     there with exit code 2, before it has changed anything -- it never
#     carries on and "restores" an empty goal over the real one;
#   * once step 1 HAS read the goal, an exit trap puts it back on any exit
#     or interrupt (Ctrl-C, a closed terminal, a kill) that happens before
#     step 16 has restored it, so a harness goal is never left on a profile
#     a running autopilot would then play toward.
#
# Self-test modes need no server, no SESSION_COOKIE and no CONNECTION_ID:
# the appended go test runs are skipped in --self-test and
# --self-test-negative, since the fixtures, not this repository's Go
# suite, are the subject of those two runs.
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
Usage: scripts/verify-phase4.sh [--self-test|--self-test-negative] [--no-tests] <report-file>

Drives the Phase 4 session-goal round trip, the Session Memory read-back,
and the captured-text retention delete over HTTP and writes <report-file>
with a PASS/FAIL/SKIP line per criterion (C1..C5) and every
request/response body underneath.

WARNING: a live run is DESTRUCTIVE. It PERMANENTLY DELETES the captured game
text of CONNECTION_ID -- every session transcript line on the Logs page and
the game-text snapshot on every AI decision row. There is no prompt, no
opt-out and no undo: that delete is what criterion C4 proves. Decision rows,
their reasoning and outcomes, Session Memory and Quest Memory are kept. The
session goal is changed during the run and put back at the end (and by an
exit trap if the run is interrupted); two new Quest rows are left behind.
The two self-test modes touch nothing.

Live run reads its inputs from the environment:

  BASE_URL                 server root, e.g. https://mudpuppy-staging.up.railway.app
  SESSION_COOKIE            full Cookie header value of an authenticated session
  CONNECTION_ID             connection_id of a profile the caller owns
  NOT_OWNED_CONNECTION_ID   optional; connection_id the caller does not own.
                            Defaults to a fixed literal UUID that no profile
                            will ever have.

  BASE_URL=https://mudpuppy-staging.up.railway.app \
  SESSION_COOKIE="session_token=abc123..." \
  CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
  scripts/verify-phase4.sh evidence/03-canned-report.txt

--self-test         run against scripts/fixtures/phase4/ instead of a
                      live server; no environment variables required.
--self-test-negative  run against scripts/fixtures/phase4-negative/ (one
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
  FIXTURE_DIR="scripts/fixtures/phase4"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase4-negative"
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
# indistinguishable on the wire (T-4-09); the default below is deliberate
# -- no profile will ever be seeded with this id.
NOT_OWNED_CONNECTION_ID="${NOT_OWNED_CONNECTION_ID:-ffffffff-ffff-ffff-ffff-ffffffffffff}"

# Seconds any one HTTP call may take before it is given up as status 000.
HTTP_MAX_TIME="${HTTP_MAX_TIME:-30}"

# Fixed goal text used by both this script's self-test fixtures and its
# live run. Self-test text stays literal so the static fixture bodies can
# match it exactly; live mode appends a run tag so two live runs in the
# same minute never collide with a goal already sitting on the profile.
if [ -n "$FIXTURE_DIR" ]; then
  GOAL_A="AI-PLAYER-HARNESS-GOAL-A"
  GOAL_B="AI-PLAYER-HARNESS-GOAL-B"
else
  RUN_TAG=$(date -u +%s)
  GOAL_A="AI-Player-Harness-Goal-A-${RUN_TAG}"
  GOAL_B="AI-Player-Harness-Goal-B-${RUN_TAG}"
fi
# The same goal, differently cased with surrounding whitespace -- D-04's
# matching is trim-and-case-fold, so this must reactivate GOAL_A's Quest,
# not create a second one.
GOAL_A_VARIANT="  $(printf '%s' "$GOAL_A" | tr '[:upper:]' '[:lower:]')  "

# ---------------------------------------------------------------------------
# Report file: everything printed from here on goes to both stdout and the
# report file, so the file is a verbatim transcript of this run.
# ---------------------------------------------------------------------------
exec > >(tee "$REPORT") 2>&1

RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================================================"
echo "Phase 4 canned-report harness -- scripts/verify-phase4.sh"
echo "Run started (UTC): $RUN_TS"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "Mode: --self-test (fixtures: scripts/fixtures/phase4/, no server)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "Mode: --self-test-negative (fixtures: scripts/fixtures/phase4-negative/, no server)"
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
echo "never a database query. (2) C3 and C5 are Go-test-and-screenshot proofs"
echo "this script cannot reach; they appear below as SKIP lines naming their"
echo "real evidence, never as PASS or FAIL. (3) invocations: live,"
echo "--self-test, --self-test-negative, --no-tests."
echo
if [ -z "$FIXTURE_DIR" ]; then
  echo "LIVE RUN -- DESTRUCTIVE: step 11 permanently deletes the captured game"
  echo "text of this connection (transcript lines and decision snapshots). That"
  echo "delete is criterion C4's proof (threat T-4-15, accepted); the counts it"
  echo "removed are printed under step 11. The session goal is put back at the"
  echo "end, and by an exit trap if this run is interrupted."
  echo
fi

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# _get_field <json> <field>
# Extracts one flat field's value from a JSON object. Prints "null" for a
# JSON null. Prints the empty string via the non-jq fallback when the key
# is altogether absent from the text (e.g. an `omitempty` field that was
# omitted). Not escape-aware -- use _get_text_field for a string that may
# contain quotes, backslashes or newlines.
_get_field() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r --arg f "$field" '
      (.[$f]) as $v
      | if $v == null then "null"
        elif ($v | type) == "string" then $v
        else ($v | tostring)
        end' 2>/dev/null
  else
    printf '%s' "$json" \
      | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*(\"[^\"]*\"|null|true|false|-?[0-9]+(\.[0-9]+)?)" \
      | head -n 1 \
      | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*//" \
      | sed -E 's/^"(.*)"$/\1/'
  fi
}

# _get_text_field <json> <field>
# Extracts a flat string field and decodes its JSON string escapes (\n, \"
# and \\) back to real characters, so a PUT that echoes a GET's value never
# re-escapes what it read (the Phase 3.1 double-escaping defect,
# 03.1-07-SUMMARY.md -- deliberately not repeated here). Used for "goal",
# never for "quest" (an unescaped bare word needs no decoding).
_get_text_field() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r --arg f "$field" '.[$f] // ""' 2>/dev/null
  else
    printf '%s' "$json" \
      | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\"([^\"\\\\]|\\\\.)*\"" \
      | head -n 1 \
      | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*\"(.*)\"$/\\1/" \
      | sed -E 's/\\n/\n/g; s/\\"/"/g; s/\\\\/\\/g'
  fi
}

# _json_escape <text>
# Escapes backslashes, double quotes and newlines for embedding a raw
# string as a JSON string literal's contents. Used only by the non-jq
# fallback of _build_goal_body below.
_json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | sed -e ':a;N;$!ba;s/\n/\\n/g'
}

# _build_goal_body <goal-text>
# Builds a {"goal": "..."} PUT body from a raw (unescaped) goal string.
_build_goal_body() {
  local goal="$1"
  if command -v jq >/dev/null 2>&1; then
    jq -n --arg g "$goal" '{goal: $g}' 2>/dev/null
  else
    printf '{"goal":"%s"}' "$(_json_escape "$goal")"
  fi
}

# _count_array_items <json> <field>
# Counts the elements of a flat array-of-strings field (e.g.
# session_memory). Assumes elements contain no unescaped double quotes --
# true of every fixture and expected server response this script drives.
_count_array_items() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r --arg f "$field" '(.[$f] // []) | length' 2>/dev/null
  else
    local arr inner
    arr=$(printf '%s' "$json" | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\\[[^]]*\\]" | head -n 1)
    if [ -z "$arr" ]; then
      echo 0
      return
    fi
    inner=$(printf '%s' "$arr" | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*\\[//; s/\\]\$//")
    if [ -z "$(printf '%s' "$inner" | tr -d '[:space:]')" ]; then
      echo 0
    else
      printf '%s' "$inner" | grep -oE '"[^"]*"' | wc -l | tr -d ' '
    fi
  fi
}

# _last_decision_field <json> <field>
# Returns the named field of the LAST object in the "decisions" array
# (document order == oldest first, per GetDecisions' own doc comment, so
# the last object is the most recent decision). Assumes flat decision
# objects (no nested braces), matching DecisionResponseItem exactly.
_last_decision_field() {
  local json="$1" field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r --arg f "$field" '(.decisions // []) | last | (.[$f] // "")' 2>/dev/null
  else
    local obj
    obj=$(printf '%s' "$json" | grep -oE '\{[^{}]*\}' | tail -n 1)
    printf '%s' "$obj" \
      | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\"([^\"\\\\]|\\\\.)*\"" \
      | tail -n 1 \
      | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*\"(.*)\"\$/\\1/" \
      | sed -E 's/\\"/"/g; s/\\\\/\\/g'
  fi
}

# Ordered list of fixture files, one per _http call in the sequence below.
# Used only when FIXTURE_DIR is set.
FIXTURE_FILES=(
  "01-ai-goal-get-original.json"
  "02-ai-goal-put-a.json"
  "03-ai-goal-get-after-a.json"
  "04-ai-goal-put-b.json"
  "05-ai-goal-get-after-b.json"
  "06-ai-goal-put-reactivate.json"
  "07-ai-goal-put-too-long.json"
  "08-ai-memory-get.json"
  "09-ai-memory-put-not-allowed.json"
  "10-decisions-get-before.json"
  "11-captured-text-delete.json"
  "12-decisions-get-after.json"
  "13-ai-goal-get-not-owned.json"
  "14-ai-memory-get-not-owned.json"
  "15-captured-text-delete-not-owned.json"
  "16-ai-goal-put-restore.json"
)
_HTTP_CALL_NO=0

# _http <method> <path> [body]
# Performs a live HTTP call against ${BASE_URL}/api/v1/<path> using the
# supplied session cookie, OR -- when FIXTURE_DIR is set (--self-test /
# --self-test-negative) -- reads the next fixture file in FIXTURE_FILES
# instead of calling curl at all. Either way, sets HTTP_STATUS and
# HTTP_BODY as globals; every line below this function (the assertions,
# the printing, the counters, the summary and the exit code) is identical
# in both modes, so the self-test exercises the real assertion logic, not
# a copy of it.
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
  # --max-time bounds every call (code review WR-11/IN-10). Without it a
  # request the server never answers hangs this script for ever -- and with
  # it the exit trap, because bash runs a trap only once the command it is
  # waiting on has returned. A call that times out reads as status 000 and
  # FAILS its check like any other wrong answer.
  local resp
  if [ -n "$body" ]; then
    resp=$(curl -sS --max-time "$HTTP_MAX_TIME" -X "$method" -b "$SESSION_COOKIE" -H 'Content-Type: application/json' \
      -d "$body" -w '\n%{http_code}' "${BASE_URL}/api/v1/${path}")
  else
    resp=$(curl -sS --max-time "$HTTP_MAX_TIME" -X "$method" -b "$SESSION_COOKIE" -H 'Content-Type: application/json' \
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
# Prints a line of the form "PASS C1: <description>" or
# "FAIL C1: <description>" for every assertion.
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

# _check_ne <label> <desc> <actual> <not-expected>
_check_ne() {
  local label="$1" desc="$2" actual="$3" not_expected="$4"
  if [ "$actual" != "$not_expected" ]; then
    _check "$label" "$desc (got: $actual)" 0
  else
    _check "$label" "$desc (got the same value: $actual)" 1
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

# _check_refused <label> <desc> <data-field>
# The ownership check (code review WR-12, T-4-09). The old assertion was
# "anything other than 200", which a server that was DOWN passed with flying
# colours: curl's own failure (000), a proxy's 502, a 500 and an expired
# cookie's 401 all printed "PASS ... is refused", so the filed IDOR evidence
# could not tell "refused because not owned" from "never got there". A
# refusal is now a SPECIFIC answer from the application:
#   * status 400, 403 or 404 (the handlers answer a not-owned or non-existent
#     connection with 400 and {"error":"Profile not found"});
#   * a JSON body with a non-empty "error";
#   * and NO <data-field> in it -- the field that would carry the other
#     owner's goal, memory, or the delete's counts.
# Anything else FAILS, and says why: 000 (transport failure or timeout), 401
# (the session is not authenticated, so ownership was never tested), 5xx
# (server or proxy error), and of course 200.
_check_refused() {
  local label="$1" desc="$2" data_field="$3"
  local why=""
  case "$HTTP_STATUS" in
    400|403|404) ;;
    000) why="no HTTP answer at all (transport failure or timeout) -- this is not a refusal" ;;
    401) why="HTTP 401: the session is not authenticated, so ownership was never tested" ;;
    5??) why="HTTP $HTTP_STATUS: a server or proxy error is not a refusal" ;;
    200) why="HTTP 200: the request was ANSWERED, not refused" ;;
    *)   why="HTTP $HTTP_STATUS is not one of the refusals 400/403/404" ;;
  esac
  if [ -n "$why" ]; then
    _check "$label" "$desc ($why)" 1
    return
  fi
  local err
  err=$(_get_field "$HTTP_BODY" error)
  if [ -z "$err" ] || [ "$err" = "null" ]; then
    _check "$label" "$desc (status $HTTP_STATUS, but the body carries no \"error\" -- not the application's own refusal)" 1
    return
  fi
  if printf '%s' "$HTTP_BODY" | grep -qE "\"$data_field\"[[:space:]]*:"; then
    _check "$label" "$desc (status $HTTP_STATUS, but the body carries \"$data_field\" -- data was returned alongside the refusal)" 1
    return
  fi
  _check "$label" "$desc (status $HTTP_STATUS, error: $err, no \"$data_field\" in the body)" 0
}

# _check_nonempty <label> <desc> <value>
_check_nonempty() {
  local label="$1" desc="$2" value="$3"
  if [ -n "$value" ] && [ "$value" != "null" ]; then
    _check "$label" "$desc (got: $value)" 0
  else
    _check "$label" "$desc (got: '$value')" 1
  fi
}

# _skip <label> <reason-and-evidence>
# Prints "SKIP C3: <text>" and increments the skip counter. A skip is
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
# Prints the raw response body whole. Never prints the session cookie, an
# API key, or any game text (T-4-05) -- only request paths and response
# bodies of the endpoints this script drives, which by construction never
# carry game text (the goal is owner-typed, memory is model-curated
# bullets, decisions omit the window snapshot by design -- T-3-15).
_print_response() {
  local method="$1" path="$2"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$HTTP_BODY"
}

# ---------------------------------------------------------------------------
# Exit trap: put the owner's goal back if this run ends early (code review
# WR-11, T-4-33). Between step 2 and step 16 the profile carries a harness
# goal; Ctrl-C, a closed terminal or a kill in that window used to leave it
# there, where a running autopilot plays toward it. The trap restores only
# when there is something real to restore:
#   ORIG_GOAL_READ=1  step 1 answered 200, so ORIG_GOAL is the owner's goal
#                     (which may legitimately be empty) -- never before that;
#   GOAL_DIRTY=1      a harness goal has been written and step 16 has not yet
#                     put the original back.
# It does nothing in the two self-test modes (no server) and runs at most
# once. It prints a status and never the cookie.
# ---------------------------------------------------------------------------
ORIG_GOAL=""
ORIG_GOAL_READ=0
GOAL_DIRTY=0
_RESTORE_TRAP_RAN=0

_restore_goal_on_exit() {
  if [ "$_RESTORE_TRAP_RAN" -eq 1 ]; then
    return 0
  fi
  _RESTORE_TRAP_RAN=1
  if [ -n "$FIXTURE_DIR" ] || [ "$ORIG_GOAL_READ" -ne 1 ] || [ "$GOAL_DIRTY" -ne 1 ]; then
    return 0
  fi
  echo
  echo "---- exit trap: this run ended before step 16 restored the goal -- restoring it now (T-4-33) ----"
  local trap_status
  trap_status=$(curl -sS --max-time "$HTTP_MAX_TIME" -o /dev/null -w '%{http_code}' -X PUT -b "$SESSION_COOKIE" \
    -H 'Content-Type: application/json' -d "$(_build_goal_body "$ORIG_GOAL")" \
    "${BASE_URL}/api/v1/profiles/${CONNECTION_ID}/ai-goal" 2>/dev/null)
  if [ "$trap_status" = "200" ]; then
    GOAL_DIRTY=0
    echo "RESTORED by the exit trap: the goal is back to what step 1 found."
  else
    echo "NOT RESTORED: the exit trap's PUT answered HTTP ${trap_status:-000}. The profile still"
    echo "carries a harness goal -- put the goal printed under step 1 back by hand in"
    echo "the AI Assist panel before engaging autopilot."
  fi
}
trap _restore_goal_on_exit EXIT
trap '_restore_goal_on_exit; exit 130' INT
trap '_restore_goal_on_exit; exit 143' TERM HUP

# ---------------------------------------------------------------------------
# Step 1 -- GET profiles/$CONNECTION_ID/ai-goal -- record the current goal
# verbatim (decoded, not re-escaped) so the restore step at the end can put
# it back exactly as found (T-4-33)
# ---------------------------------------------------------------------------
_step "Step 1: GET profiles/\$CONNECTION_ID/ai-goal -- record the current goal (T-4-33 restore)"
_http GET "profiles/${CONNECTION_ID}/ai-goal"
_print_response GET "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "ai-goal GET answers HTTP 200 for the owning connection" "200"
if [ "$HTTP_STATUS" != "200" ]; then
  # Code review WR-11: without the original goal there is nothing to put
  # back. Carrying on used to end in step 16 "restoring" an EMPTY goal over
  # the owner's real one and printing RESTORED. Nothing has been changed
  # yet, so stop here.
  echo
  echo "ABORTED: step 1 could not read the current goal (HTTP $HTTP_STATUS). Nothing has"
  echo "been changed on the server. Check BASE_URL, SESSION_COOKIE (an expired"
  echo "session answers 401) and CONNECTION_ID, then run again."
  exit 2
fi
ORIG_GOAL=$(_get_text_field "$HTTP_BODY" goal)
ORIG_GOAL_READ=1
echo "goal found before this run (kept verbatim for the restore step): $ORIG_GOAL"

# ---------------------------------------------------------------------------
# Step 2 -- PUT a distinctive goal (GOAL_A), GET it back, compare byte for
# byte (C1, D-01)
# ---------------------------------------------------------------------------
_step "Step 2: PUT profiles/\$CONNECTION_ID/ai-goal -- write goal A (C1, D-01)"
GOAL_DIRTY=1   # from here until step 16 succeeds, the exit trap restores the goal
_http PUT "profiles/${CONNECTION_ID}/ai-goal" "$(_build_goal_body "$GOAL_A")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "PUT accepts goal A" "200"
PUT_A_GOAL=$(_get_text_field "$HTTP_BODY" goal)
_check_eq C1 "PUT response echoes goal A" "$PUT_A_GOAL" "$GOAL_A"
PUT_A_QUEST=$(_get_field "$HTTP_BODY" quest)
_check_eq C1 "PUT response names the Quest as newly created" "$PUT_A_QUEST" "created"

_step "Step 3: GET profiles/\$CONNECTION_ID/ai-goal -- goal A survives a real PUT and GET (C1, D-01)"
_http GET "profiles/${CONNECTION_ID}/ai-goal"
_print_response GET "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "ai-goal GET answers HTTP 200 after the PUT" "200"
GET_A_GOAL=$(_get_text_field "$HTTP_BODY" goal)
_check_eq C1 "goal A round-trips character for character through PUT and GET" "$GET_A_GOAL" "$GOAL_A"

# ---------------------------------------------------------------------------
# Step 4 -- PUT a second, different goal (GOAL_B) and GET it back, proving
# an edit takes (C1, D-03)
# ---------------------------------------------------------------------------
_step "Step 4: PUT profiles/\$CONNECTION_ID/ai-goal -- write goal B, a different goal (C1, D-03)"
_http PUT "profiles/${CONNECTION_ID}/ai-goal" "$(_build_goal_body "$GOAL_B")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "PUT accepts goal B" "200"
PUT_B_GOAL=$(_get_text_field "$HTTP_BODY" goal)
_check_eq C1 "PUT response echoes goal B" "$PUT_B_GOAL" "$GOAL_B"

_step "Step 5: GET profiles/\$CONNECTION_ID/ai-goal -- goal B survives, and differs from goal A (C1, D-03)"
_http GET "profiles/${CONNECTION_ID}/ai-goal"
_print_response GET "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "ai-goal GET answers HTTP 200 after the edit" "200"
GET_B_GOAL=$(_get_text_field "$HTTP_BODY" goal)
_check_eq C1 "goal B round-trips character for character through PUT and GET" "$GET_B_GOAL" "$GOAL_B"
_check_ne C1 "the edited goal differs from the original goal A, proving an edit takes" "$GET_B_GOAL" "$GOAL_A"

# ---------------------------------------------------------------------------
# Step 6 -- PUT goal A again, differently cased with surrounding spaces:
# the same Quest is reactivated, not duplicated (C1, D-04)
# ---------------------------------------------------------------------------
_step "Step 6: PUT profiles/\$CONNECTION_ID/ai-goal -- goal A again, re-cased and padded: reactivate, not duplicate (C1, D-04)"
_http PUT "profiles/${CONNECTION_ID}/ai-goal" "$(_build_goal_body "$GOAL_A_VARIANT")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "PUT accepts the re-cased, padded repeat of goal A" "200"
REACTIVATE_QUEST=$(_get_field "$HTTP_BODY" quest)
_check_eq C1 "the same Quest is reactivated rather than a second one created" "$REACTIVATE_QUEST" "reactivated"

# ---------------------------------------------------------------------------
# Step 7 -- PUT a 1001-character goal, refused with 400 (C1, T-4-10)
# ---------------------------------------------------------------------------
_step "Step 7: PUT profiles/\$CONNECTION_ID/ai-goal -- a 1001-character goal is refused (C1, T-4-10)"
TOO_LONG_GOAL=$(printf '%1001s' '' | tr ' ' 'a')
_http PUT "profiles/${CONNECTION_ID}/ai-goal" "$(_build_goal_body "$TOO_LONG_GOAL")"
echo "PUT /api/v1/profiles/\$CONNECTION_ID/ai-goal -> HTTP $HTTP_STATUS (request body's goal omitted from this report: 1001 repeated characters, not game text, but long)"
echo "$HTTP_BODY"
_check_status C1 "a 1001-character goal is refused with 400" "400"
TOO_LONG_ERROR=$(_get_field "$HTTP_BODY" error)
_check_eq C1 "the refusal message matches the exact 1000-character cap sentence" "$TOO_LONG_ERROR" "Session goal must be 1000 characters or less"

# ---------------------------------------------------------------------------
# Step 8 -- GET profiles/$CONNECTION_ID/ai-memory -- a well-formed array
# response (C2, D-10)
# ---------------------------------------------------------------------------
_step "Step 8: GET profiles/\$CONNECTION_ID/ai-memory -- Session Memory reads back as an array (C2, D-10)"
_http GET "profiles/${CONNECTION_ID}/ai-memory"
_print_response GET "profiles/\$CONNECTION_ID/ai-memory"
_check_status C2 "ai-memory GET answers HTTP 200 for the owning connection" "200"
MEMORY_ARRAY_SHAPE=$(printf '%s' "$HTTP_BODY" | grep -oE '"session_memory"[[:space:]]*:[[:space:]]*\[' || true)
if [ -n "$MEMORY_ARRAY_SHAPE" ]; then
  _check C2 "response carries a session_memory array" 0
else
  _check C2 "response carries a session_memory array" 1
fi
MEMORY_LEN=$(_count_array_items "$HTTP_BODY" session_memory)
echo "session_memory length: $MEMORY_LEN"

_step "Step 9: PUT profiles/\$CONNECTION_ID/ai-memory -- read-only this phase, refused with 405 (C2, D-10)"
_http PUT "profiles/${CONNECTION_ID}/ai-memory" '{"session_memory":["hand-edited"]}'
_print_response PUT "profiles/\$CONNECTION_ID/ai-memory"
_check_status C2 "ai-memory PUT is refused with 405 Method Not Allowed (editing memory by hand is Phase 5)" "405"

# ---------------------------------------------------------------------------
# Step 10 -- GET decisions before the delete: record the most recent
# decision's id, outcome and whether its reasoning is non-empty (C4, D-21)
# ---------------------------------------------------------------------------
_step "Step 10: GET profiles/\$CONNECTION_ID/decisions -- record the most recent decision before the delete (C4, D-21)"
_http GET "profiles/${CONNECTION_ID}/decisions"
_print_response GET "profiles/\$CONNECTION_ID/decisions"
_check_status C4 "decisions GET answers HTTP 200 for the owning connection" "200"
DECISIONS_BEFORE_ID=$(_last_decision_field "$HTTP_BODY" id)
DECISIONS_BEFORE_OUTCOME=$(_last_decision_field "$HTTP_BODY" outcome)
DECISIONS_BEFORE_REASONING=$(_last_decision_field "$HTTP_BODY" reasoning)
DECISIONS_BEFORE_COMMAND=$(_last_decision_field "$HTTP_BODY" command)
DECISIONS_BEFORE_FAILURE_KIND=$(_last_decision_field "$HTTP_BODY" failure_kind)
if [ -n "$DECISIONS_BEFORE_ID" ]; then
  echo "most recent decision before delete: id=$DECISIONS_BEFORE_ID outcome=$DECISIONS_BEFORE_OUTCOME reasoning_length=${#DECISIONS_BEFORE_REASONING}"
  _check_nonempty C4 "the most recent decision's id is present" "$DECISIONS_BEFORE_ID"
  _check_nonempty C4 "the most recent decision's reasoning is non-empty" "$DECISIONS_BEFORE_REASONING"
else
  echo "no decisions exist yet for this connection -- the survival check (step 12) is skipped, not faked"
fi

# ---------------------------------------------------------------------------
# Step 11 -- DELETE captured-text: the retention delete reports both counts
# (C4, D-21)
# ---------------------------------------------------------------------------
_step "Step 11: DELETE profiles/\$CONNECTION_ID/captured-text -- the retention delete reports its counts (C4, D-21)"
_http DELETE "profiles/${CONNECTION_ID}/captured-text"
_print_response DELETE "profiles/\$CONNECTION_ID/captured-text"
_check_status C4 "captured-text delete answers HTTP 200 for the owning connection" "200"
SNAPSHOTS_CLEARED=$(_get_field "$HTTP_BODY" snapshots_cleared)
LINES_DELETED=$(_get_field "$HTTP_BODY" transcript_lines_deleted)
case "$SNAPSHOTS_CLEARED" in
  ''|*[!0-9]*) _check C4 "the delete response carries a numeric snapshots_cleared count (got: $SNAPSHOTS_CLEARED)" 1 ;;
  *) _check C4 "the delete response carries a numeric snapshots_cleared count (got: $SNAPSHOTS_CLEARED)" 0 ;;
esac
case "$LINES_DELETED" in
  ''|*[!0-9]*) _check C4 "the delete response carries a numeric transcript_lines_deleted count (got: $LINES_DELETED)" 1 ;;
  *) _check C4 "the delete response carries a numeric transcript_lines_deleted count (got: $LINES_DELETED)" 0 ;;
esac

# ---------------------------------------------------------------------------
# Step 12 -- GET decisions after the delete: the same decision still
# returns with its reasoning, command, outcome and failure_kind intact
# (C4, D-21 -- the decision audit record is not captured text)
# ---------------------------------------------------------------------------
_step "Step 12: GET profiles/\$CONNECTION_ID/decisions -- the decision record survives the delete (C4, D-21)"
_http GET "profiles/${CONNECTION_ID}/decisions"
_print_response GET "profiles/\$CONNECTION_ID/decisions"
_check_status C4 "decisions GET answers HTTP 200 after the delete" "200"
if [ -n "$DECISIONS_BEFORE_ID" ]; then
  DECISIONS_AFTER_ID=$(_last_decision_field "$HTTP_BODY" id)
  DECISIONS_AFTER_OUTCOME=$(_last_decision_field "$HTTP_BODY" outcome)
  DECISIONS_AFTER_REASONING=$(_last_decision_field "$HTTP_BODY" reasoning)
  DECISIONS_AFTER_COMMAND=$(_last_decision_field "$HTTP_BODY" command)
  DECISIONS_AFTER_FAILURE_KIND=$(_last_decision_field "$HTTP_BODY" failure_kind)
  _check_eq C4 "the same decision id is still returned after the delete" "$DECISIONS_AFTER_ID" "$DECISIONS_BEFORE_ID"
  _check_eq C4 "the decision's outcome is unchanged" "$DECISIONS_AFTER_OUTCOME" "$DECISIONS_BEFORE_OUTCOME"
  _check_eq C4 "the decision's command is unchanged" "$DECISIONS_AFTER_COMMAND" "$DECISIONS_BEFORE_COMMAND"
  _check_eq C4 "the decision's failure_kind is unchanged" "$DECISIONS_AFTER_FAILURE_KIND" "$DECISIONS_BEFORE_FAILURE_KIND"
  _check_nonempty C4 "the decision's reasoning is still non-empty after the delete -- the audit record survives the prune" "$DECISIONS_AFTER_REASONING"
else
  _skip C4 "no decision existed before the delete to check survival of -- run the walkthrough first so at least one decision exists (a live run before it has nothing to find and must not lie in either direction)"
fi

# ---------------------------------------------------------------------------
# Step 13 -- Ownership: a connection the caller does not own is refused on
# every new endpoint (C1, C2, C4, T-4-09)
# ---------------------------------------------------------------------------
_step "Step 13: GET profiles/\$NOT_OWNED_CONNECTION_ID/ai-goal -- refused, not answered with someone else's goal (C1, T-4-09)"
_http GET "profiles/${NOT_OWNED_CONNECTION_ID}/ai-goal"
_print_response GET "profiles/\$NOT_OWNED_CONNECTION_ID/ai-goal"
_check_refused C1 "a not-owned/non-existent connection_id is refused on ai-goal by the application itself" "goal"

_step "Step 14: GET profiles/\$NOT_OWNED_CONNECTION_ID/ai-memory -- refused, not answered with someone else's memory (C2, T-4-09)"
_http GET "profiles/${NOT_OWNED_CONNECTION_ID}/ai-memory"
_print_response GET "profiles/\$NOT_OWNED_CONNECTION_ID/ai-memory"
_check_refused C2 "a not-owned/non-existent connection_id is refused on ai-memory by the application itself" "session_memory"

_step "Step 15: DELETE profiles/\$NOT_OWNED_CONNECTION_ID/captured-text -- refused, never deletes someone else's captured text (C4, T-4-09)"
_http DELETE "profiles/${NOT_OWNED_CONNECTION_ID}/captured-text"
_print_response DELETE "profiles/\$NOT_OWNED_CONNECTION_ID/captured-text"
_check_refused C4 "a not-owned/non-existent connection_id is refused on the captured-text delete by the application itself, and nothing was deleted" "snapshots_cleared"

# ---------------------------------------------------------------------------
# Step 16 -- Restore: PUT the goal captured in step 1 back, so a live run
# leaves the profile exactly as it found it (T-4-33)
# ---------------------------------------------------------------------------
_step "Step 16: PUT profiles/\$CONNECTION_ID/ai-goal -- restore the pre-run goal (T-4-33)"
_http PUT "profiles/${CONNECTION_ID}/ai-goal" "$(_build_goal_body "$ORIG_GOAL")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-goal"
_check_status C1 "the profile's goal is restored to its pre-run value" "200"
RESTORED_GOAL=$(_get_text_field "$HTTP_BODY" goal)
_check_eq C1 "the restored goal matches what step 1 found, decoded and not re-escaped" "$RESTORED_GOAL" "$ORIG_GOAL"
if [ "$HTTP_STATUS" = "200" ] && [ "$RESTORED_GOAL" = "$ORIG_GOAL" ]; then
  GOAL_DIRTY=0
  echo "RESTORED: goal set back to what step 1 found. (The captured text deleted in step 11 is NOT restored and cannot be; two harness Quest rows remain.)"
else
  echo "NOT RESTORED by step 16 -- the exit trap will try once more when this script ends."
fi

# ---------------------------------------------------------------------------
# Step 17 -- C3, out of this script's reach: the mechanical limits
# ---------------------------------------------------------------------------
_step "Step 17: C3 -- out of this script's reach"
_skip C3 "the call cap, threshold disengage, blocked-repeatedly disengage and blank-settings behavior (D-14 to D-17) cannot be driven by an HTTP client at all -- proven instead by \`go test ./internal/driver/... -run \"TestLoop_CallCap|TestLoop_ErrorThreshold|TestLoop_ConsecutiveBlocks|TestLoop_BlankSettings\" -count=1 -v\` and evidence/08-cap-halt-badge-off.png"

# ---------------------------------------------------------------------------
# Step 18 -- C5, out of this script's reach: wheel-grab and the re-engage
# reassessment
# ---------------------------------------------------------------------------
_step "Step 18: C5 -- out of this script's reach"
_skip C5 "wheel-grab and the re-engage reassessment (D-08) require a live game session, a hand-typed command, and a human reading the first post-re-engage decision's reasoning -- proven instead by \`go test ./internal/driver/... -run \"TestLoop_|TestEngageLoop_\" -count=1 -v\` and evidence/09-wheel-grab-reengage-reasoning.png"

# ---------------------------------------------------------------------------
# Step 19 -- go test: the diagnostic proof that the driver and session
# packages this phase touched are green. Skipped in self-test modes and
# when --no-tests is given. Never runs the live corpus test -- that stays
# a hand-run phase gate spending real Gemini quota (D-24, plan 04-11).
# ---------------------------------------------------------------------------
_step "Step 19: go test -- driver and session packages"
GO_TEST_EXIT=0
if [ "$SELF_TEST" -eq 1 ] || [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "SKIPPED (self-test mode: fixtures are the subject of this run, not the Go suite)"
elif [ "$NO_TESTS" -eq 1 ]; then
  echo "SKIPPED (--no-tests)"
else
  REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo .)
  echo "\$ go test ./internal/driver/... ./internal/session/... -count=1"
  ( cd "$REPO_ROOT" && go test ./internal/driver/... ./internal/session/... -count=1 )
  GO_TEST_EXIT=$?
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "================================================================"
echo "SUMMARY"
echo "================================================================"
for label in C1 C2 C3 C4 C5; do
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
