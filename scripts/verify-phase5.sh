#!/usr/bin/env bash
#
# scripts/verify-phase5.sh -- Phase 5 canned-report harness (AI Player:
# coaching channel).
#
# Drives the HTTP-reachable half of Phase 5 -- pause and resume over the
# switch endpoint with their two waiting reasons (D-13, D-15), the coaching
# list and the conversation reading back GET-only (D-09, D-18), the AI
# command rate limit round-tripping as a profile setting with blank meaning
# the server default (D-25), and another user's connection being refused on
# every one of the new reads (T-4-09 re-run) -- against a running MUDPuppy
# server, and writes a report file that states, criterion by criterion
# (C1..C5, fixed in 05-10-PLAN.md's context table), whether Phase 5's
# reachable half holds, with the request path and response status printed
# under every step. This IS the canned report the owner's evidence rule
# requires; no database query is used anywhere in this script.
#
# UNLIKE scripts/verify-phase4.sh, A LIVE RUN OF THIS SCRIPT IS NOT
# DESTRUCTIVE. It changes exactly one thing on the profile -- the AI Command
# Rate Limit setting -- and puts it back at the end, and by an exit trap if
# the run is interrupted. It never turns the switch on, never sends a game
# command and never deletes anything.
#
# This is scripts/verify-phase4.sh's own shape, pointed at this phase's
# endpoints -- same argument parsing, same fixture-or-curl request helper,
# same PASS/FAIL/SKIP-per-criterion report, same restore-what-it-changed
# discipline, same two self-test modes. One deliberate departure from that
# skeleton: this machine's own standing instructions for this plan restrict
# this script to bash and curl only -- no JSON query tool, no alternate
# scripting interpreter of any kind -- so the field-extraction helpers below
# have no alternate-tool branch at all -- unlike
# verify-phase4.sh's optional-tool / grep-and-sed-otherwise pair, every
# extraction here always uses the grep/sed fallback. Documented as a
# deviation in 05-10-SUMMARY.md.
#
# A second deliberate departure, also documented as a deviation: this
# plan's own task text says to print "the full response body" under every
# step. This machine's own standing instructions (carried from the Phase 4
# security audit, T-4-05) forbid printing the raw bodies of conversation,
# coaching, decision or memory content into a committed report -- only
# status codes, counts and field presence. Where machine instruction and
# plan text conflict, the machine instruction governs: the two coaching/
# conversation reads (C2) print status, array-field presence and an item
# count only, never the coaching or conversation text itself. Every other
# step (session status, the autopilot switch, the AI settings) carries no
# game text, coaching text or conversation text at all, and prints its full
# body exactly as the plan's task text asks.
#
# Criterion labels (05-10-PLAN.md's context table, verbatim):
#   C1  Pause and resume over the switch endpoint: pausing an engaged switch
#       reports WAITING with paused_by_owner true, resuming reports ON with
#       both reasons false, and pausing an already-off switch is refused
#       harmlessly. Reachable.
#   C2  The coaching list and the conversation read back what the session
#       holds, and both are GET-only (a PUT, POST or DELETE is refused).
#       Reachable.
#   C3  The AI command rate limit round-trips as a profile setting: a set
#       value returns, a blank value returns blank, and an out-of-range
#       value is rejected with the bounds message. Reachable.
#   C4  Another user's connection id is refused on every one of the new
#       reads. Reachable.
#   C5  A coaching message reaching the next decision's reasoning, and the
#       reviewer, prompt and limiter behaviour. NOT reachable -- this script
#       prints exactly:
#         SKIP C5: ... proven instead by
#         `go test ./internal/driver/... -run "TestHandleChat_|TestCoaching" -count=1 -v`
#         and `go test ./internal/session/... -run "TestManager_Pause|TestManager_Resume" -count=1 -v`
#         and evidence/06-coaching-in-next-decision.png,
#         evidence/18-paused-waiting-reason.png
#
# Three ground rules a reader needs:
#   1. Proof in this project is a canned report or an end-user screenshot,
#      never a database query.
#   2. C5 is out of this script's reach: a coaching message reaching a real
#      model decision, and the reviewer/prompt/limiter behaviour, cannot be
#      driven by an HTTP client with no live model call. It is printed below
#      as a SKIP line naming the exact `go test` invocations and screenshot
#      filenames that prove it instead -- never as PASS, never as FAIL.
#   3. There are two invocations (run with no arguments to print full
#      usage):
#
#   1. Live run, against any running server (NOT destructive, see above):
#        BASE_URL=https://mudpuppy-staging.up.railway.app \
#        SESSION_COOKIE="session_token=abc123..." \
#        CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
#        scripts/verify-phase5.sh evidence/03-canned-report.txt
#
#   2. Self-test, no server, proves the assertion logic against
#      scripts/fixtures/phase5/ (a correct-server fixture set):
#        scripts/verify-phase5.sh --self-test /tmp/phase5-selftest.txt
#
#   3. Negative self-test, proves the harness can FAIL against
#      scripts/fixtures/phase5-negative/ (one deliberately wrong response)
#      so the harness is trusted before its report is (T-5-45):
#        scripts/verify-phase5.sh --self-test-negative /tmp/phase5-negative.txt
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
# the wire (T-4-09, re-run again on these two new reads), and both must be
# refused.
#
# This harness never issues a live Gemini call, never deploys anything, and
# never turns the autopilot switch on itself -- C1's pause/resume assertions
# only run when a prior owner action has already engaged the switch; if it
# finds the switch not engaged, C1 is SKIPPED-WITH-REASON rather than the
# harness engaging it (T-5-47). It never touches captured game text and
# never runs SQL.
#
set -uo pipefail   # pipefail only; errexit is intentionally omitted -- a
                    # failed check must not abort the report -- the report
                    # must list every result, not stop at the first FAIL.

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
SELF_TEST=0
SELF_TEST_NEGATIVE=0
REPORT=""

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/verify-phase5.sh [--self-test|--self-test-negative] <report-file>

Drives the Phase 5 pause/resume switch actions, the coaching and
conversation reads, and the AI command rate limit round trip over HTTP and
writes <report-file> with a PASS/FAIL/SKIP line per criterion (C1..C5) and
the request path and response status under every step.

UNLIKE scripts/verify-phase4.sh, a live run of this script is NOT
destructive: it changes only the AI Command Rate Limit setting and puts it
back at the end (and by an exit trap if this run is interrupted). It never
turns autopilot on, never sends a game command, and never deletes anything.

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
  NOT_OWNED_CONNECTION_ID=99999999-8888-7777-6666-555555555555 \
  scripts/verify-phase5.sh evidence/03-canned-report.txt

--self-test         run against scripts/fixtures/phase5/ instead of a
                      live server; no environment variables required.
--self-test-negative  run against scripts/fixtures/phase5-negative/ (one
                      deliberately wrong response) to prove the harness
                      FAILs and exits non-zero on a wrong server response.
USAGE
}

for arg in "$@"; do
  case "$arg" in
    --self-test) SELF_TEST=1 ;;
    --self-test-negative) SELF_TEST_NEGATIVE=1 ;;
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
  FIXTURE_DIR="scripts/fixtures/phase5"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase5-negative"
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
# indistinguishable on the wire (T-4-09); the default below is deliberate --
# no profile will ever be seeded with this id.
NOT_OWNED_CONNECTION_ID="${NOT_OWNED_CONNECTION_ID:-ffffffff-ffff-ffff-ffff-ffffffffffff}"

# Seconds any one HTTP call may take before it is given up as status 000.
HTTP_MAX_TIME="${HTTP_MAX_TIME:-30}"

# C3's in-bounds test value and out-of-bounds probe. 1-20 is the accepted
# range (internal/profiles/handler.go's validateAISettings); 25 is outside
# it on the high side.
RATE_LIMIT_IN_BOUNDS=5
RATE_LIMIT_OUT_OF_BOUNDS=25
RATE_LIMIT_BOUNDS_MESSAGE="AI command rate limit must be between 1 and 20 commands per second"

# ---------------------------------------------------------------------------
# Report file: everything printed from here on goes to both stdout and the
# report file, so the file is a verbatim transcript of this run.
# ---------------------------------------------------------------------------
exec > >(tee "$REPORT") 2>&1

RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================================================"
echo "Phase 5 canned-report harness -- scripts/verify-phase5.sh"
echo "Run started (UTC): $RUN_TS"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "Mode: --self-test (fixtures: scripts/fixtures/phase5/, no server)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "Mode: --self-test-negative (fixtures: scripts/fixtures/phase5-negative/, no server)"
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
echo "never a database query. (2) C5 is a Go-test-and-screenshot proof this"
echo "script cannot reach; it appears below as a SKIP line naming its real"
echo "evidence, never as PASS or FAIL. (3) invocations: live, --self-test,"
echo "--self-test-negative."
echo
if [ -z "$FIXTURE_DIR" ]; then
  echo "LIVE RUN -- NOT DESTRUCTIVE (unlike Phase 4's harness): the only thing"
  echo "this run changes is the AI Command Rate Limit setting, which is put"
  echo "back at the end, and by an exit trap if this run is interrupted. The"
  echo "switch is never engaged by this script; C1 is skipped with a reason"
  echo "if it finds the switch not already engaged by the owner."
  echo
fi

# ---------------------------------------------------------------------------
# Helpers -- grep/sed only. This machine's standing instructions for this
# plan restrict this script to bash and curl, with no other query tool or
# scripting interpreter, so unlike verify-phase4.sh's optional-tool branch,
# every extraction here always takes the fallback path.
# ---------------------------------------------------------------------------

# _get_field <json> <field>
# Extracts one flat field's value from a JSON object. Prints "null" for a
# JSON null, and the empty string when the key is altogether absent. Not
# escape-aware -- use _get_text_field for a string that may contain quotes,
# backslashes or newlines.
_get_field() {
  local json="$1" field="$2"
  printf '%s' "$json" \
    | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*(\"[^\"]*\"|null|true|false|-?[0-9]+(\.[0-9]+)?)" \
    | head -n 1 \
    | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*//" \
    | sed -E 's/^"(.*)"$/\1/'
}

# _get_text_field <json> <field>
# Extracts a flat string field and decodes its JSON string escapes (\n, \"
# and \\) back to real characters, so a PUT that echoes a GET's value never
# re-escapes what it read (the Phase 3.1 double-escaping defect, not
# repeated here).
_get_text_field() {
  local json="$1" field="$2"
  printf '%s' "$json" \
    | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\"([^\"\\\\]|\\\\.)*\"" \
    | head -n 1 \
    | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*\"(.*)\"\$/\\1/" \
    | sed -E 's/\\n/\n/g; s/\\"/"/g; s/\\\\/\\/g'
}

# _get_nested_object <json> <field>
# Returns the raw substring of a flat nested object field (no braces inside
# it), e.g. "ai_settings":{...}. Used only for ai_settings, whose four
# members (a string and three nullable ints) never themselves contain a
# brace.
_get_nested_object() {
  local json="$1" field="$2"
  printf '%s' "$json" \
    | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\{[^}]*\}" \
    | head -n 1 \
    | sed -E "s/^\"$field\"[[:space:]]*:[[:space:]]*//"
}

# _json_escape <text>
# Escapes backslashes, double quotes and newlines for embedding a raw
# string as a JSON string literal's contents.
_json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | sed -e ':a;N;$!ba;s/\n/\\n/g'
}

# _build_autopilot_body <action>
# Builds a {"action": "...", "connection_id": "..."} POST body for
# /api/v1/session/autopilot. <action> is always "pause" or "resume" in this
# script -- the switch-engaging action is never built here (T-5-47).
_build_autopilot_body() {
  local action="$1"
  printf '{"action":"%s","connection_id":"%s"}' "$action" "$CONNECTION_ID"
}

# _build_ai_settings_body <conduct> <approach> <never-issue> <model> <call-cap> <threshold> <rate-limit>
# Builds a full AISettingsResponse-shaped PUT body (internal/profiles/
# handler.go). <call-cap>, <threshold> and <rate-limit> are each either the
# literal bare word null or a bare integer -- never quoted, matching the
# *int fields they fill on the wire.
_build_ai_settings_body() {
  local conduct="$1" approach="$2" never_issue="$3" model="$4" call_cap="$5" threshold="$6" rate_limit="$7"
  printf '{"conduct_rules":"%s","approach_guidance":"%s","never_issue_list":"%s","ai_settings":{"model_name":"%s","call_cap":%s,"disengage_threshold":%s,"rate_limit_per_second":%s}}' \
    "$(_json_escape "$conduct")" "$(_json_escape "$approach")" "$(_json_escape "$never_issue")" \
    "$(_json_escape "$model")" "$call_cap" "$threshold" "$rate_limit"
}

# _count_string_array_items <json> <field>
# Counts the elements of a flat array-of-strings field (e.g. coaching).
# Assumes elements contain no unescaped double quotes.
_count_string_array_items() {
  local json="$1" field="$2"
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
}

# _count_object_array_items <json> <field> <inner-key>
# Counts the elements of an array-of-objects field (e.g. lines, each a
# ConversationLineResponse) by counting occurrences of one key every
# element carries exactly once (id). Never touches the text/speaker
# content, so a count is obtainable without reading conversation text.
_count_object_array_items() {
  local json="$1" field="$2" inner_key="$3"
  local arr
  arr=$(printf '%s' "$json" | grep -oE "\"$field\"[[:space:]]*:[[:space:]]*\\[.*\\]" | head -n 1)
  if [ -z "$arr" ]; then
    echo 0
    return
  fi
  printf '%s' "$arr" | grep -oE "\"$inner_key\"[[:space:]]*:" | wc -l | tr -d ' '
}

# Ordered list of fixture files, one per _http call in the sequence below.
# Used only when FIXTURE_DIR is set.
FIXTURE_FILES=(
  "01-session-status-get.json"
  "02-autopilot-pause.json"
  "03-autopilot-pause-again.json"
  "04-autopilot-resume.json"
  "05-ai-coaching-get.json"
  "06-ai-conversation-get.json"
  "07-ai-coaching-put-not-allowed.json"
  "08-ai-coaching-delete-not-allowed.json"
  "09-ai-conversation-put-not-allowed.json"
  "10-ai-conversation-delete-not-allowed.json"
  "11-ai-settings-get-original.json"
  "12-ai-settings-put-inbounds.json"
  "13-ai-settings-get-after-inbounds.json"
  "14-ai-settings-put-blank.json"
  "15-ai-settings-get-after-blank.json"
  "16-ai-settings-put-out-of-bounds.json"
  "17-ai-settings-put-restore.json"
  "18-ai-settings-get-after-restore.json"
  "19-ai-coaching-get-not-owned.json"
  "20-ai-conversation-get-not-owned.json"
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
# a copy of it. The session cookie is never echoed anywhere -- it is only
# ever passed to curl's -b flag, never printed.
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
  # --max-time bounds every call. Without it a request the server never
  # answers hangs this script for ever -- and with it the exit trap, because
  # bash runs a trap only once the command it is waiting on has returned. A
  # call that times out reads as status 000 and FAILS its check like any
  # other wrong answer.
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

# _check_refused <label> <desc> <data-field>
# The ownership check (T-4-09, re-run on this phase's two new reads). A
# refusal is a SPECIFIC answer from the application:
#   * status 400, 403 or 404;
#   * a JSON body with a non-empty "error";
#   * and NO <data-field> in it -- the field that would carry the other
#     owner's coaching list or conversation.
# Anything else FAILS, and says why.
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

# _skip <label> <reason-and-evidence>
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
# Prints the raw response body whole. Used only for steps whose body never
# carries conversation, coaching, decision or memory text (session status,
# the autopilot switch, the AI settings).
_print_response() {
  local method="$1" path="$2"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$HTTP_BODY"
}

# _print_response_redacted <method> <display-path> <note>
# Prints the request path and status only, with a stated reason -- never
# the response body. Used for the coaching and conversation reads (C2),
# per this machine's standing instruction (T-4-05 carried forward): this
# harness never prints conversation, coaching, decision or memory text into
# a committed report, only status codes, counts and field presence.
_print_response_redacted() {
  local method="$1" path="$2" note="$3"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "(response body redacted -- $note; see 05-10-SUMMARY.md Deviations)"
}

# ---------------------------------------------------------------------------
# Exit trap: put the owner's AI settings back if this run ends early. Once
# step 11 has read the original settings, the profile carries this
# harness's own rate-limit value until step 17 restores it (or a failure
# leaves one of the intermediate values standing); Ctrl-C, a closed
# terminal or a kill in that window must not leave it there.
#   ORIG_SETTINGS_READ=1  step 11 answered 200, so ORIG_* below are the
#                         owner's real settings -- never restore before this;
#   SETTINGS_DIRTY=1      a harness value has been written and step 17 has
#                         not yet put the original back.
# Does nothing in the two self-test modes (no server) and runs at most once.
# ---------------------------------------------------------------------------
ORIG_CONDUCT_RULES=""
ORIG_APPROACH_GUIDANCE=""
ORIG_NEVER_ISSUE_LIST=""
ORIG_MODEL_NAME=""
ORIG_CALL_CAP="null"
ORIG_DISENGAGE_THRESHOLD="null"
ORIG_RATE_LIMIT="null"
ORIG_SETTINGS_READ=0
SETTINGS_DIRTY=0
_RESTORE_TRAP_RAN=0

_restore_settings_on_exit() {
  if [ "$_RESTORE_TRAP_RAN" -eq 1 ]; then
    return 0
  fi
  _RESTORE_TRAP_RAN=1
  if [ -n "$FIXTURE_DIR" ] || [ "$ORIG_SETTINGS_READ" -ne 1 ] || [ "$SETTINGS_DIRTY" -ne 1 ]; then
    return 0
  fi
  echo
  echo "---- exit trap: this run ended before step 17 restored the AI settings -- restoring now ----"
  local restore_body trap_status
  restore_body=$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$ORIG_NEVER_ISSUE_LIST" \
    "$ORIG_MODEL_NAME" "$ORIG_CALL_CAP" "$ORIG_DISENGAGE_THRESHOLD" "$ORIG_RATE_LIMIT")
  trap_status=$(curl -sS --max-time "$HTTP_MAX_TIME" -o /dev/null -w '%{http_code}' -X PUT -b "$SESSION_COOKIE" \
    -H 'Content-Type: application/json' -d "$restore_body" \
    "${BASE_URL}/api/v1/profiles/${CONNECTION_ID}/ai-settings" 2>/dev/null)
  if [ "$trap_status" = "200" ]; then
    SETTINGS_DIRTY=0
    echo "RESTORED by the exit trap: the AI settings are back to what step 11 found."
  else
    echo "NOT RESTORED: the exit trap's PUT answered HTTP ${trap_status:-000}. The profile still"
    echo "carries a harness rate-limit value -- restore it by hand in AI Player settings"
    echo "before relying on the profile's own configured limit."
  fi
}
trap _restore_settings_on_exit EXIT
trap '_restore_settings_on_exit; exit 130' INT
trap '_restore_settings_on_exit; exit 143' TERM HUP

# ---------------------------------------------------------------------------
# C1 -- pause and resume over the switch endpoint
# ---------------------------------------------------------------------------
_step "Step 1: GET session/status -- read the switch's current state (C1)"
_http GET "session/status"
_print_response GET "session/status"
AUTOPILOT_STATE=""
if [ "$HTTP_STATUS" = "200" ]; then
  AUTOPILOT_STATE=$(_get_field "$HTTP_BODY" autopilot_state)
fi
echo "autopilot_state found before this run: '$AUTOPILOT_STATE'"

if [ "$AUTOPILOT_STATE" = "on" ]; then
  _step "Step 2: POST session/autopilot action=pause -- pause an engaged switch (C1, D-13, D-15)"
  _http POST "session/autopilot" "$(_build_autopilot_body pause)"
  _print_response POST "session/autopilot"
  _check_status C1 "pause answers HTTP 200" "200"
  PAUSE_STATE=$(_get_field "$HTTP_BODY" state)
  PAUSE_OUTCOME=$(_get_field "$HTTP_BODY" outcome)
  PAUSE_PAUSED_BY_OWNER=$(_get_field "$HTTP_BODY" paused_by_owner)
  PAUSE_CONNECTION_LOST=$(_get_field "$HTTP_BODY" connection_lost)
  _check_eq C1 "pausing an engaged switch reports state waiting" "$PAUSE_STATE" "waiting"
  _check_eq C1 "pausing an engaged switch reports outcome paused" "$PAUSE_OUTCOME" "paused"
  _check_eq C1 "pausing an engaged switch sets paused_by_owner true" "$PAUSE_PAUSED_BY_OWNER" "true"
  _check_eq C1 "pausing an engaged switch leaves connection_lost false" "$PAUSE_CONNECTION_LOST" "false"

  _step "Step 3: POST session/autopilot action=pause -- pause again is refused harmlessly (C1, D-15)"
  _http POST "session/autopilot" "$(_build_autopilot_body pause)"
  _print_response POST "session/autopilot"
  _check_status C1 "the repeated pause answers HTTP 200" "200"
  PAUSE2_STATE=$(_get_field "$HTTP_BODY" state)
  PAUSE2_OUTCOME=$(_get_field "$HTTP_BODY" outcome)
  PAUSE2_PAUSED_BY_OWNER=$(_get_field "$HTTP_BODY" paused_by_owner)
  PAUSE2_CONNECTION_LOST=$(_get_field "$HTTP_BODY" connection_lost)
  _check_eq C1 "a repeated pause reports outcome already-waiting, not a second paused" "$PAUSE2_OUTCOME" "already-waiting"
  _check_eq C1 "a repeated pause leaves the state at waiting" "$PAUSE2_STATE" "waiting"
  _check_eq C1 "a repeated pause changes nothing: paused_by_owner is unchanged" "$PAUSE2_PAUSED_BY_OWNER" "$PAUSE_PAUSED_BY_OWNER"
  _check_eq C1 "a repeated pause changes nothing: connection_lost is unchanged" "$PAUSE2_CONNECTION_LOST" "$PAUSE_CONNECTION_LOST"

  _step "Step 4: POST session/autopilot action=resume -- resume clears both reasons (C1, D-13, D-15)"
  _http POST "session/autopilot" "$(_build_autopilot_body resume)"
  _print_response POST "session/autopilot"
  _check_status C1 "resume answers HTTP 200" "200"
  RESUME_STATE=$(_get_field "$HTTP_BODY" state)
  RESUME_OUTCOME=$(_get_field "$HTTP_BODY" outcome)
  RESUME_PAUSED_BY_OWNER=$(_get_field "$HTTP_BODY" paused_by_owner)
  RESUME_CONNECTION_LOST=$(_get_field "$HTTP_BODY" connection_lost)
  _check_eq C1 "resuming reports state on" "$RESUME_STATE" "on"
  _check_eq C1 "resuming reports outcome resumed" "$RESUME_OUTCOME" "resumed"
  _check_eq C1 "resuming clears paused_by_owner" "$RESUME_PAUSED_BY_OWNER" "false"
  _check_eq C1 "resuming clears connection_lost" "$RESUME_CONNECTION_LOST" "false"
else
  _skip C1 "the switch was not engaged when this run started (autopilot_state='$AUTOPILOT_STATE') -- pausing an off switch would prove nothing, and this harness never engages the switch itself (T-5-47); SKIPPED-WITH-REASON. Engage autopilot by hand (#AUTO ON) before a live run if C1 must be exercised."
fi

# ---------------------------------------------------------------------------
# C2 -- the coaching list and the conversation, both GET-only
# ---------------------------------------------------------------------------
_step "Step 5: GET profiles/\$CONNECTION_ID/ai-coaching -- the coaching list reads back (C2, D-09)"
_http GET "profiles/${CONNECTION_ID}/ai-coaching"
_print_response_redacted GET "profiles/\$CONNECTION_ID/ai-coaching" "coaching suggestions are the owner's own working list, never printed raw"
_check_status C2 "ai-coaching GET answers HTTP 200 for the owning connection" "200"
if printf '%s' "$HTTP_BODY" | grep -qE '"coaching"[[:space:]]*:[[:space:]]*\['; then
  _check C2 "response carries a coaching array (never null)" 0
else
  _check C2 "response carries a coaching array (never null)" 1
fi
COACHING_COUNT=$(_count_string_array_items "$HTTP_BODY" coaching)
echo "coaching item count: $COACHING_COUNT"

_step "Step 6: GET profiles/\$CONNECTION_ID/ai-conversation -- the conversation reads back (C2, D-09)"
_http GET "profiles/${CONNECTION_ID}/ai-conversation"
_print_response_redacted GET "profiles/\$CONNECTION_ID/ai-conversation" "the conversation is owner and AI-chatter text, never printed raw"
_check_status C2 "ai-conversation GET answers HTTP 200 for the owning connection" "200"
if printf '%s' "$HTTP_BODY" | grep -qE '"lines"[[:space:]]*:[[:space:]]*\['; then
  _check C2 "response carries a lines array (never null)" 0
else
  _check C2 "response carries a lines array (never null)" 1
fi
CONVERSATION_COUNT=$(_count_object_array_items "$HTTP_BODY" lines id)
echo "conversation line count: $CONVERSATION_COUNT"

_step "Step 7: PUT profiles/\$CONNECTION_ID/ai-coaching -- refused, coaching is GET-only (C2, D-18)"
_http PUT "profiles/${CONNECTION_ID}/ai-coaching" '{"coaching":["hand-edited"]}'
_print_response_redacted PUT "profiles/\$CONNECTION_ID/ai-coaching" "a refusal body carries no coaching text"
_check_status C2 "ai-coaching PUT is refused with 405 Method Not Allowed" "405"

_step "Step 8: DELETE profiles/\$CONNECTION_ID/ai-coaching -- refused, coaching is GET-only (C2, D-18)"
_http DELETE "profiles/${CONNECTION_ID}/ai-coaching"
_print_response_redacted DELETE "profiles/\$CONNECTION_ID/ai-coaching" "a refusal body carries no coaching text"
_check_status C2 "ai-coaching DELETE is refused with 405 Method Not Allowed" "405"

_step "Step 9: PUT profiles/\$CONNECTION_ID/ai-conversation -- refused, the conversation is GET-only (C2, D-18)"
_http PUT "profiles/${CONNECTION_ID}/ai-conversation" '{"lines":[]}'
_print_response_redacted PUT "profiles/\$CONNECTION_ID/ai-conversation" "a refusal body carries no conversation text"
_check_status C2 "ai-conversation PUT is refused with 405 Method Not Allowed" "405"

_step "Step 10: DELETE profiles/\$CONNECTION_ID/ai-conversation -- refused, the conversation is GET-only (C2, D-18)"
_http DELETE "profiles/${CONNECTION_ID}/ai-conversation"
_print_response_redacted DELETE "profiles/\$CONNECTION_ID/ai-conversation" "a refusal body carries no conversation text"
_check_status C2 "ai-conversation DELETE is refused with 405 Method Not Allowed" "405"

# ---------------------------------------------------------------------------
# C3 -- the AI command rate limit round-trips as a profile setting
# ---------------------------------------------------------------------------
_step "Step 11: GET profiles/\$CONNECTION_ID/ai-settings -- record the current settings (C3, restore discipline)"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "ai-settings GET answers HTTP 200 for the owning connection" "200"
if [ "$HTTP_STATUS" != "200" ]; then
  echo
  echo "ABORTED: step 11 could not read the current AI settings (HTTP $HTTP_STATUS). Nothing"
  echo "has been changed on the server. Check BASE_URL, SESSION_COOKIE and CONNECTION_ID,"
  echo "then run again."
  exit 2
fi
ORIG_CONDUCT_RULES=$(_get_text_field "$HTTP_BODY" conduct_rules)
ORIG_APPROACH_GUIDANCE=$(_get_text_field "$HTTP_BODY" approach_guidance)
ORIG_NEVER_ISSUE_LIST=$(_get_text_field "$HTTP_BODY" never_issue_list)
ORIG_AI_SETTINGS_OBJ=$(_get_nested_object "$HTTP_BODY" ai_settings)
ORIG_MODEL_NAME=$(_get_text_field "$ORIG_AI_SETTINGS_OBJ" model_name)
ORIG_CALL_CAP=$(_get_field "$ORIG_AI_SETTINGS_OBJ" call_cap)
ORIG_DISENGAGE_THRESHOLD=$(_get_field "$ORIG_AI_SETTINGS_OBJ" disengage_threshold)
ORIG_RATE_LIMIT=$(_get_field "$ORIG_AI_SETTINGS_OBJ" rate_limit_per_second)
[ -z "$ORIG_CALL_CAP" ] && ORIG_CALL_CAP="null"
[ -z "$ORIG_DISENGAGE_THRESHOLD" ] && ORIG_DISENGAGE_THRESHOLD="null"
[ -z "$ORIG_RATE_LIMIT" ] && ORIG_RATE_LIMIT="null"
ORIG_SETTINGS_READ=1
echo "rate_limit_per_second found before this run (kept for the restore step): $ORIG_RATE_LIMIT"

_step "Step 12: PUT profiles/\$CONNECTION_ID/ai-settings -- set the rate limit to $RATE_LIMIT_IN_BOUNDS, inside 1-20 (C3, D-25)"
SETTINGS_DIRTY=1   # from here until step 17 succeeds, the exit trap restores the settings
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$ORIG_NEVER_ISSUE_LIST" "$ORIG_MODEL_NAME" "$ORIG_CALL_CAP" "$ORIG_DISENGAGE_THRESHOLD" "$RATE_LIMIT_IN_BOUNDS")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "PUT accepts an in-bounds rate limit" "200"

_step "Step 13: GET profiles/\$CONNECTION_ID/ai-settings -- the set value returns (C3, D-25)"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "ai-settings GET answers HTTP 200 after the PUT" "200"
GET_AI_SETTINGS_A=$(_get_nested_object "$HTTP_BODY" ai_settings)
GET_RATE_LIMIT_A=$(_get_field "$GET_AI_SETTINGS_A" rate_limit_per_second)
_check_eq C3 "the set rate limit round-trips through PUT and GET" "$GET_RATE_LIMIT_A" "$RATE_LIMIT_IN_BOUNDS"

_step "Step 14: PUT profiles/\$CONNECTION_ID/ai-settings -- set the rate limit to blank (C3, D-25)"
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$ORIG_NEVER_ISSUE_LIST" "$ORIG_MODEL_NAME" "$ORIG_CALL_CAP" "$ORIG_DISENGAGE_THRESHOLD" "null")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "PUT accepts a blank rate limit" "200"

_step "Step 15: GET profiles/\$CONNECTION_ID/ai-settings -- a blank value returns blank (C3, D-25)"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "ai-settings GET answers HTTP 200 after the blank PUT" "200"
GET_AI_SETTINGS_B=$(_get_nested_object "$HTTP_BODY" ai_settings)
GET_RATE_LIMIT_B=$(_get_field "$GET_AI_SETTINGS_B" rate_limit_per_second)
_check_eq C3 "a blank rate limit round-trips as null, resolved to the server default in Go, not the browser" "$GET_RATE_LIMIT_B" "null"

_step "Step 16: PUT profiles/\$CONNECTION_ID/ai-settings -- an out-of-range rate limit is rejected (C3, D-25)"
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$ORIG_NEVER_ISSUE_LIST" "$ORIG_MODEL_NAME" "$ORIG_CALL_CAP" "$ORIG_DISENGAGE_THRESHOLD" "$RATE_LIMIT_OUT_OF_BOUNDS")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "an out-of-range rate limit is rejected with 400" "400"
OUT_OF_BOUNDS_ERROR=$(_get_field "$HTTP_BODY" error)
_check_eq C3 "the refusal message matches the exact bounds sentence" "$OUT_OF_BOUNDS_ERROR" "$RATE_LIMIT_BOUNDS_MESSAGE"

_step "Step 17: PUT profiles/\$CONNECTION_ID/ai-settings -- restore the pre-run settings (restore discipline)"
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$ORIG_NEVER_ISSUE_LIST" "$ORIG_MODEL_NAME" "$ORIG_CALL_CAP" "$ORIG_DISENGAGE_THRESHOLD" "$ORIG_RATE_LIMIT")"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "the profile's AI settings are restored to their pre-run values" "200"
if [ "$HTTP_STATUS" = "200" ]; then
  SETTINGS_DIRTY=0
fi

_step "Step 18: GET profiles/\$CONNECTION_ID/ai-settings -- the restore matches what step 11 found (restore discipline)"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "ai-settings GET answers HTTP 200 after the restore" "200"
RESTORED_AI_SETTINGS=$(_get_nested_object "$HTTP_BODY" ai_settings)
RESTORED_RATE_LIMIT=$(_get_field "$RESTORED_AI_SETTINGS" rate_limit_per_second)
_check_eq C3 "the restored rate limit matches what step 11 found" "$RESTORED_RATE_LIMIT" "$ORIG_RATE_LIMIT"
echo "RESTORED: AI settings set back to what step 11 found. (The captured game text, decisions, coaching and conversation this run touched, if any, are untouched: this run wrote nothing to any of them.)"

# ---------------------------------------------------------------------------
# C4 -- ownership: a connection the caller does not own is refused
# ---------------------------------------------------------------------------
_step "Step 19: GET profiles/\$NOT_OWNED_CONNECTION_ID/ai-coaching -- refused, not answered with someone else's coaching (C4, T-4-09)"
_http GET "profiles/${NOT_OWNED_CONNECTION_ID}/ai-coaching"
_print_response_redacted GET "profiles/\$NOT_OWNED_CONNECTION_ID/ai-coaching" "a refusal body carries no coaching text"
_check_refused C4 "a not-owned/non-existent connection_id is refused on ai-coaching by the application itself" "coaching"

_step "Step 20: GET profiles/\$NOT_OWNED_CONNECTION_ID/ai-conversation -- refused, not answered with someone else's conversation (C4, T-4-09)"
_http GET "profiles/${NOT_OWNED_CONNECTION_ID}/ai-conversation"
_print_response_redacted GET "profiles/\$NOT_OWNED_CONNECTION_ID/ai-conversation" "a refusal body carries no conversation text"
_check_refused C4 "a not-owned/non-existent connection_id is refused on ai-conversation by the application itself" "lines"

# ---------------------------------------------------------------------------
# C5 -- not reachable: a coaching message reaching the next decision's
# reasoning, and the reviewer, prompt and limiter behaviour
# ---------------------------------------------------------------------------
_step "Step 21: C5 -- out of this script's reach"
_skip C5 "a coaching message reaching the next decision's reasoning, and the reviewer/prompt/limiter behaviour, require a live model call this script never makes -- proven instead by \`go test ./internal/driver/... -run \"TestHandleChat_|TestCoaching\" -count=1 -v\` and \`go test ./internal/session/... -run \"TestManager_Pause|TestManager_Resume\" -count=1 -v\` and evidence/06-coaching-in-next-decision.png, evidence/18-paused-waiting-reason.png"

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

if [ "$FAIL_COUNT" -eq 0 ]; then
  exit 0
else
  exit 1
fi
