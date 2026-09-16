#!/usr/bin/env bash
#
# scripts/verify-phase3-1.sh -- Phase 3.1 canned-report harness (AI Player:
# prompt-injection review).
#
# Drives the HTTP-reachable half of Phase 3.1 -- GET/PUT
# /api/v1/profiles/{connection_id}/ai-settings (the Never-issue list, C3)
# and GET /api/v1/profiles/{connection_id}/decisions (a blocked decision's
# read-back, C5) -- against a running MUDPuppy server, and writes a report
# file that states, criterion by criterion (C1..C6, matching ROADMAP.md's
# Phase 3.1 Success Criteria 1-6), whether Phase 3.1's reachable half holds,
# with the raw request path and response body printed under every step.
# This IS the canned report the owner's evidence rule requires; no database
# query is used anywhere in this script.
#
# This is scripts/verify-phase3.sh's own shape, pointed at this phase's two
# endpoints -- same argument parsing, same fixture-or-curl request helper,
# same flat-JSON field extractor, same PASS/FAIL/SKIP-per-criterion report.
#
# Three ground rules a reader needs:
#   1. Proof in this project is a canned report or an end-user screenshot,
#      never a database query.
#   2. C1, C2, C4 and C6 are out of this script's reach: the live red-team
#      run, the hostile-content Go test report, and the reviewer-pass Go
#      tests and staging screenshots are not things this script can drive
#      or fabricate. They are printed as SKIP lines naming the evidence
#      file or test name that proves each instead -- never as PASS.
#   3. There are four invocations (run with no arguments to print full
#      usage):
#
#   1. Live run, against any running server:
#        BASE_URL=https://mudpuppy-staging.up.railway.app \
#        SESSION_COOKIE="session=abc123..." \
#        CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
#        scripts/verify-phase3-1.sh evidence/04-canned-report.txt
#
#   2. Self-test, no server, proves the assertion logic against
#      scripts/fixtures/phase3-1/ (a correct-server fixture set):
#        scripts/verify-phase3-1.sh --self-test /tmp/phase3-1-selftest.txt
#
#   3. Negative self-test, proves the harness can FAIL against
#      scripts/fixtures/phase3-1-negative/ (one deliberately wrong
#      response) so the harness is trusted before its report is (T-3.1-12):
#        scripts/verify-phase3-1.sh --self-test-negative /tmp/phase3-1-negative.txt
#
#   4. Skip the appended `go test` runs (live mode only):
#        BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... \
#        scripts/verify-phase3-1.sh --no-tests report.txt
#
# CONNECTION_ID must name a profile the caller owns. NOT_OWNED_CONNECTION_ID
# is optional and defaults to a fixed literal UUID that no profile will ever
# have -- a connection id the caller does not own and one that does not
# exist are indistinguishable on the wire, and both must be refused
# (T-3.1-09, the Phase 3 IDOR check re-run unchanged).
#
# This harness never issues a live Gemini call and never runs the live
# corpus test -- that stays a hand-run phase gate (D-11, T-3.1-18). It never
# reads the session cookie, an API key or any game text back into the
# report it writes (T-3.1-13). The step-5 restore below PUTs back the
# Never-issue list value this run found at step 1, so a live run leaves the
# profile exactly as it found it (T-3.1-26).
#
# Self-test modes need no server and no repository state: the appended go
# test runs are skipped in --self-test and --self-test-negative, since
# fixtures are the subject of those runs, not this repository's Go suite.
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
Usage: scripts/verify-phase3-1.sh [--self-test|--self-test-negative] [--no-tests] <report-file>

Drives the Phase 3.1 Never-issue list round trip and the blocked-decision
read-back over HTTP and writes <report-file> with a PASS/FAIL/SKIP line per
ROADMAP criterion (C1..C6) and every request/response body underneath.

Live run reads its inputs from the environment:

  BASE_URL                 server root, e.g. https://mudpuppy-staging.up.railway.app
  SESSION_COOKIE            full Cookie header value of an authenticated session
  CONNECTION_ID             connection_id of a profile the caller owns
  NOT_OWNED_CONNECTION_ID   optional; connection_id the caller does not own.
                            Defaults to a fixed literal UUID that no profile
                            will ever have.

  BASE_URL=https://mudpuppy-staging.up.railway.app \
  SESSION_COOKIE="session=abc123..." \
  CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
  scripts/verify-phase3-1.sh evidence/04-canned-report.txt

--self-test          run against scripts/fixtures/phase3-1/ instead of a
                      live server; no environment variables required.
--self-test-negative  run against scripts/fixtures/phase3-1-negative/ (one
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
  FIXTURE_DIR="scripts/fixtures/phase3-1"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase3-1-negative"
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
# indistinguishable on the wire (T-3.1-09, carried from Phase 3's T-3-03);
# the default below is deliberate -- no profile will ever be seeded with
# this id.
NOT_OWNED_CONNECTION_ID="${NOT_OWNED_CONNECTION_ID:-ffffffff-ffff-ffff-ffff-ffffffffffff}"

# ---------------------------------------------------------------------------
# Report file: everything printed from here on goes to both stdout and the
# report file, so the file is a verbatim transcript of this run.
# ---------------------------------------------------------------------------
exec > >(tee "$REPORT") 2>&1

RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================================================"
echo "Phase 3.1 canned-report harness -- scripts/verify-phase3-1.sh"
echo "Run started (UTC): $RUN_TS"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "Mode: --self-test (fixtures: scripts/fixtures/phase3-1/, no server)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "Mode: --self-test-negative (fixtures: scripts/fixtures/phase3-1-negative/, no server)"
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
echo "never a database query. (2) C1, C2, C4 and C6 are red-team-run,"
echo "hostile-content-test, reviewer-test and staging-screenshot proofs this"
echo "script cannot reach; they appear below as SKIP lines naming their real"
echo "evidence, never as PASS. (3) invocations: live, --self-test,"
echo "--self-test-negative, --no-tests."
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

# _get_never_issue_list <json>
# Extracts the never_issue_list field and decodes its JSON string escapes
# (\n, \" and \\) back to real characters, so a multi-line value round-trips
# through comparison correctly whether or not jq is installed -- unlike
# _get_field above, which is deliberately left unmodified per this phase's
# analog and is single-line-safe only.
_get_never_issue_list() {
  local json="$1"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -r '.never_issue_list // ""' 2>/dev/null
  else
    printf '%s' "$json" \
      | grep -oE '"never_issue_list"[[:space:]]*:[[:space:]]*"([^"\\]|\\.)*"' \
      | head -n 1 \
      | sed -E 's/^"never_issue_list"[[:space:]]*:[[:space:]]*"(.*)"$/\1/' \
      | sed -E 's/\\n/\n/g; s/\\"/"/g; s/\\\\/\\/g'
  fi
}

# _get_ai_settings_json <json>
# Extracts the raw ai_settings object substring, unchanged, so it can be
# echoed back verbatim on the PUT that restores/round-trips the profile.
# Safe with a plain grep because AISettings (model_name, call_cap,
# disengage_threshold) is flat -- no nested braces.
_get_ai_settings_json() {
  local json="$1"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$json" | jq -c '.ai_settings // {}' 2>/dev/null
  else
    printf '%s' "$json" \
      | grep -oE '"ai_settings"[[:space:]]*:[[:space:]]*\{[^}]*\}' \
      | head -n 1 \
      | sed -E 's/^"ai_settings"[[:space:]]*:[[:space:]]*//'
  fi
}

# _json_escape <text>
# Escapes backslashes, double quotes and newlines for embedding a raw string
# as a JSON string literal's contents. Used only by the non-jq fallback of
# _build_ai_settings_body below.
_json_escape() {
  printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g' | sed -e ':a;N;$!ba;s/\n/\\n/g'
}

# _build_ai_settings_body <conduct_rules> <approach_guidance> <never_issue_list> <ai_settings_json>
# Builds the AISettingsResponse-shaped PUT body from previously-extracted
# field values, so every PUT in this script sends every AI field back
# unchanged except the one field the step under test is exercising (the
# handler's PutAISettings decodes the whole struct, so a partial body would
# blank the fields it omits).
_build_ai_settings_body() {
  local conduct="$1" approach="$2" never_issue="$3" ai_settings_json="$4"
  if command -v jq >/dev/null 2>&1; then
    jq -n --arg cr "$conduct" --arg ag "$approach" --arg nl "$never_issue" --argjson ais "$ai_settings_json" \
      '{conduct_rules: $cr, approach_guidance: $ag, never_issue_list: $nl, ai_settings: $ais}' 2>/dev/null
  else
    printf '{"conduct_rules":"%s","approach_guidance":"%s","never_issue_list":"%s","ai_settings":%s}' \
      "$(_json_escape "$conduct")" "$(_json_escape "$approach")" "$(_json_escape "$never_issue")" "$ai_settings_json"
  fi
}

# Ordered list of fixture files, one per _http call in the sequence below.
# Used only when FIXTURE_DIR is set. Call 5 (the restore PUT) deliberately
# reuses fixture 01: the restore sends back the value step 1 found (empty on
# this fixture profile), so the correct response to it is the same body
# fixture 01 already represents -- no seventh fixture file is needed, and
# the plan's six-fixture file list stays exactly six.
FIXTURE_FILES=(
  "01-ai-settings-get-empty.json"
  "02-ai-settings-put.json"
  "03-ai-settings-get-after.json"
  "04-ai-settings-too-long.json"
  "01-ai-settings-get-empty.json"
  "05-decisions-blocked.json"
  "06-decisions-not-owned.json"
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
# Prints the raw response body whole. Never prints the session cookie, an
# API key, or any game text (T-3.1-13) -- only request paths and response
# bodies of the two endpoints this script drives.
_print_response() {
  local method="$1" path="$2"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$HTTP_BODY"
}

TOO_LONG_MESSAGE="Never-issue list must be 20000 characters or less"

# ---------------------------------------------------------------------------
# Step 1 -- GET profiles/$CONNECTION_ID/ai-settings -- record the profile's
# existing fields so every PUT below sends them back unchanged (C3)
# ---------------------------------------------------------------------------
_step "Step 1: GET profiles/\$CONNECTION_ID/ai-settings -- record existing fields (C3)"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "ai-settings GET answers HTTP 200 for the owning connection" "200"
ORIG_CONDUCT_RULES=$(_get_field "$HTTP_BODY" conduct_rules)
ORIG_APPROACH_GUIDANCE=$(_get_field "$HTTP_BODY" approach_guidance)
ORIG_NEVER_ISSUE_LIST=$(_get_never_issue_list "$HTTP_BODY")
ORIG_AI_SETTINGS_JSON=$(_get_ai_settings_json "$HTTP_BODY")
echo "never_issue_list found before this run: $(printf '%s' "$ORIG_NEVER_ISSUE_LIST" | tr '\n' '|')"

# ---------------------------------------------------------------------------
# Step 2 -- PUT profiles/$CONNECTION_ID/ai-settings -- write a two-entry
# never_issue_list, conduct_rules/approach_guidance/ai_settings sent back
# unchanged (C3's first assertion)
# ---------------------------------------------------------------------------
_step "Step 2: PUT profiles/\$CONNECTION_ID/ai-settings -- write a two-entry never_issue_list (C3)"
NEW_LIST=$'give\nopen vault'
PUT_BODY=$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$NEW_LIST" "$ORIG_AI_SETTINGS_JSON")
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$PUT_BODY"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "PUT accepts a two-entry never_issue_list (\"give\" and \"open vault\")" "200"
PUT_NEVER_ISSUE=$(_get_never_issue_list "$HTTP_BODY")
_check_eq C3 "PUT response echoes the never_issue_list it was sent" "$(printf '%s' "$PUT_NEVER_ISSUE" | tr '\n' '|')" "$(printf '%s' "$NEW_LIST" | tr '\n' '|')"

# ---------------------------------------------------------------------------
# Step 3 -- GET profiles/$CONNECTION_ID/ai-settings -- the value survives a
# real PUT and GET character for character (C3's main assertion)
# ---------------------------------------------------------------------------
_step "Step 3: GET profiles/\$CONNECTION_ID/ai-settings -- the two lines survive a real PUT and GET (C3)"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "ai-settings GET answers HTTP 200 after the PUT" "200"
GET_AFTER_NEVER_ISSUE=$(_get_never_issue_list "$HTTP_BODY")
_check_eq C3 "never_issue_list round-trips character for character through PUT and GET" "$(printf '%s' "$GET_AFTER_NEVER_ISSUE" | tr '\n' '|')" "$(printf '%s' "$NEW_LIST" | tr '\n' '|')"

# ---------------------------------------------------------------------------
# Step 4 -- PUT profiles/$CONNECTION_ID/ai-settings -- a 20,001-character
# list is refused with 400 (C3's second assertion)
# ---------------------------------------------------------------------------
_step "Step 4: PUT profiles/\$CONNECTION_ID/ai-settings -- a 20001-character never_issue_list is refused (C3)"
TOO_LONG_LIST=$(printf '%20001s' '' | tr ' ' 'a')
PUT_TOO_LONG_BODY=$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$TOO_LONG_LIST" "$ORIG_AI_SETTINGS_JSON")
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$PUT_TOO_LONG_BODY"
echo "PUT /api/v1/profiles/\$CONNECTION_ID/ai-settings -> HTTP $HTTP_STATUS (request body's never_issue_list omitted from this report: 20001 repeated characters, not game text, but long)"
echo "$HTTP_BODY"
_check_status C3 "a 20001-character never_issue_list is refused with 400" "400"
TOO_LONG_ERROR=$(_get_field "$HTTP_BODY" error)
_check_eq C3 "the refusal message matches the exact 20000-character cap sentence" "$TOO_LONG_ERROR" "$TOO_LONG_MESSAGE"

# ---------------------------------------------------------------------------
# Step 5 -- PUT profiles/$CONNECTION_ID/ai-settings -- restore the value
# step 1 found, so a live run leaves the profile exactly as it found it
# (T-3.1-26)
# ---------------------------------------------------------------------------
_step "Step 5: PUT profiles/\$CONNECTION_ID/ai-settings -- restore the pre-run never_issue_list (T-3.1-26)"
RESTORE_BODY=$(_build_ai_settings_body "$ORIG_CONDUCT_RULES" "$ORIG_APPROACH_GUIDANCE" "$ORIG_NEVER_ISSUE_LIST" "$ORIG_AI_SETTINGS_JSON")
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "$RESTORE_BODY"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C3 "the profile's never_issue_list is restored to its pre-run value" "200"
echo "RESTORED: never_issue_list set back to what step 1 found -- this run leaves the profile exactly as it found it."

# ---------------------------------------------------------------------------
# Step 6 -- GET profiles/$CONNECTION_ID/decisions -- a blocked decision
# reads back with its outcome, failure_kind and notice (C5's main
# assertion, D-08, D-14)
# ---------------------------------------------------------------------------
_step "Step 6: GET profiles/\$CONNECTION_ID/decisions -- a blocked decision reads back (C5)"
_http GET "profiles/${CONNECTION_ID}/decisions"
_print_response GET "profiles/\$CONNECTION_ID/decisions"
_check_status C5 "decisions endpoint answers HTTP 200 for the owning connection" "200"
DECISIONS_RAW=$(printf '%s' "$HTTP_BODY" | grep -oE '"decisions"[[:space:]]*:[[:space:]]*\[' || true)
if [ -n "$DECISIONS_RAW" ]; then
  _check C5 "response carries a decisions array" 0
else
  _check C5 "response carries a decisions array" 1
fi
HAS_BLOCKED=$(printf '%s' "$HTTP_BODY" | grep -oE '"outcome"[[:space:]]*:[[:space:]]*"blocked"' || true)
if [ -n "$HAS_BLOCKED" ]; then
  if command -v jq >/dev/null 2>&1; then
    BLOCKED_FAILURE_KIND=$(printf '%s' "$HTTP_BODY" | jq -r '[.decisions[]? | select(.outcome == "blocked")] | last | .failure_kind // ""' 2>/dev/null)
    BLOCKED_NOTICE=$(printf '%s' "$HTTP_BODY" | jq -r '[.decisions[]? | select(.outcome == "blocked")] | last | .notice // ""' 2>/dev/null)
  else
    # No jq: take the last occurrence of failure_kind/notice in document
    # order (oldest first, so the last one is the newest blocked entry).
    # The notice field's own pattern allows escaped quotes inside the
    # value (the locked notice text quotes the owner's Never-issue entry,
    # e.g. \"give\"), then decodes \" and \\ back to real characters.
    # Isolate the LAST decision object whose outcome is blocked (objects are flat, no nested braces), then read
    # its fields -- never the last field in the whole body, which may belong to a later sent or failed row.
    BLOCKED_OBJ=$(printf '%s' "$HTTP_BODY" | grep -oE '\{[^{}]*\}' | grep -E '"outcome"[[:space:]]*:[[:space:]]*"blocked"' | tail -n 1)
    BLOCKED_FAILURE_KIND=$(printf '%s' "$BLOCKED_OBJ" | grep -oE '"failure_kind"[[:space:]]*:[[:space:]]*"[^"]*"' | tail -n 1 | sed -E 's/^"failure_kind"[[:space:]]*:[[:space:]]*"(.*)"$/\1/')
    BLOCKED_NOTICE=$(printf '%s' "$BLOCKED_OBJ" | grep -oE '"notice"[[:space:]]*:[[:space:]]*"([^"\\]|\\.)*"' | tail -n 1 | sed -E 's/^"notice"[[:space:]]*:[[:space:]]*"(.*)"$/\1/' | sed -E 's/\\"/"/g; s/\\\\/\\/g')
  fi
  case "$BLOCKED_FAILURE_KIND" in
    never-issue|reviewer)
      _check C5 "the blocked decision carries a failure_kind of never-issue or reviewer (got: $BLOCKED_FAILURE_KIND)" 0
      ;;
    *)
      _check C5 "the blocked decision carries a failure_kind of never-issue or reviewer (got: $BLOCKED_FAILURE_KIND)" 1
      ;;
  esac
  if [ -n "$BLOCKED_NOTICE" ] && [ "$BLOCKED_NOTICE" != "null" ]; then
    _check C5 "the blocked decision carries a non-empty notice (notice: $BLOCKED_NOTICE)" 0
  else
    _check C5 "the blocked decision carries a non-empty notice (notice: $BLOCKED_NOTICE)" 1
  fi
else
  _skip C5 "no blocked decision exists yet for this connection -- run the Never-issue-listed-command walkthrough first (a live run before it has nothing to find and must not lie in either direction)"
fi

# ---------------------------------------------------------------------------
# Step 7 -- GET profiles/$NOT_OWNED_CONNECTION_ID/decisions -- refused, not
# answered with someone else's rows (C5, T-3.1-09, the Phase 3 IDOR
# re-check)
# ---------------------------------------------------------------------------
_step "Step 7: GET profiles/\$NOT_OWNED_CONNECTION_ID/decisions -- refused, not someone else's rows (C5, T-3.1-09)"
_http GET "profiles/${NOT_OWNED_CONNECTION_ID}/decisions"
_print_response GET "profiles/\$NOT_OWNED_CONNECTION_ID/decisions"
_check_not_status C5 "a not-owned/non-existent connection_id is refused, never answered with HTTP 200 rows" "200"

# ---------------------------------------------------------------------------
# Step 8 -- go test: the diagnostic proof that the driver, gemini, store and
# profiles packages this phase touched are green. Skipped in self-test
# modes and when --no-tests is given. Never runs the live corpus test --
# that stays a hand-run phase gate spending real Gemini quota (D-11,
# T-3.1-18).
# ---------------------------------------------------------------------------
_step "Step 8: go test -- driver, gemini, store and profiles packages"
GO_TEST_EXIT=0
if [ "$SELF_TEST" -eq 1 ] || [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "SKIPPED (self-test mode: fixtures are the subject of this run, not the Go suite)"
elif [ "$NO_TESTS" -eq 1 ]; then
  echo "SKIPPED (--no-tests)"
else
  REPO_ROOT=$(git rev-parse --show-toplevel 2>/dev/null || echo .)
  echo "\$ go test ./internal/driver/... ./internal/gemini/... ./internal/store/... ./internal/profiles/... -v"
  ( cd "$REPO_ROOT" && go test ./internal/driver/... ./internal/gemini/... ./internal/store/... ./internal/profiles/... -v )
  GO_TEST_EXIT=$?
fi

# ---------------------------------------------------------------------------
# Step 9 -- C1, out of this script's reach: the live red-team run
# ---------------------------------------------------------------------------
_step "Step 9: C1 -- out of this script's reach"
_skip C1 "the red-team run against a live model requires real Gemini quota and a connected game session; this script must not fabricate one. Proven instead by evidence/01-redteam-before.txt and evidence/05-redteam-after.txt"

# ---------------------------------------------------------------------------
# Step 10 -- C2, out of this script's reach: the untrusted-data framing test
# ---------------------------------------------------------------------------
_step "Step 10: C2 -- out of this script's reach"
_skip C2 "the untrusted-data system-instruction framing is proven by a Go test, not an HTTP call. Proven by the TestBuildSystemInstruction run in evidence/02-test-report.txt"

# ---------------------------------------------------------------------------
# Step 11 -- C4, out of this script's reach: the reviewer pass
# ---------------------------------------------------------------------------
_step "Step 11: C4 -- out of this script's reach"
_skip C4 "the reviewer pass (a second Gemini call judging the model's chosen command) is proven by Go tests, not an HTTP call this script can drive. Proven by the TestHandleEngageReviewer and TestReviewCommand runs in evidence/02-test-report.txt"

# ---------------------------------------------------------------------------
# Step 12 -- C6, out of this script's reach: the staging demonstration
# ---------------------------------------------------------------------------
_step "Step 12: C6 -- out of this script's reach"
_skip C6 "the end-to-end staging demonstration (the owner seeing a blocked attempt on screen) requires a live game session and a human's eyes. Proven instead by the staging screenshots and the AFTER red-team report"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "================================================================"
echo "SUMMARY"
echo "================================================================"
for label in C1 C2 C3 C4 C5 C6; do
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
