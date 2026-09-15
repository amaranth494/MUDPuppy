#!/usr/bin/env bash
#
# scripts/verify-phase2.sh -- Phase 2 canned-report harness (AI Player:
# autopilot switch).
#
# Drives the HTTP-reachable half of Phase 2 -- POST /api/v1/session/autopilot
# (on/off/status), GET /api/v1/session/status, and the Phase 1
# profiles/{connection_id}/policy/accept endpoint used only to move a fixture
# profile from unaccepted to accepted -- against a running MUDPuppy server
# and writes a report file that states, criterion by criterion (C1..C4,
# matching ROADMAP.md's Phase 2 Success Criteria 1-4), whether Phase 2's
# reachable half holds, with the raw request path and response body printed
# under every step. This IS the canned report the owner's evidence rule
# requires; no database query is used anywhere in this script.
#
# Three ground rules a reader needs:
#   1. Proof in this project is a canned report or an end-user screenshot,
#      never a database query.
#   2. C2 (wheel-grab) and C3 (disconnect/waiting/resume) are out of this
#      script's scope: engaging autopilot at all requires a connected game
#      session (D-03), which this script does not and must not fabricate.
#      They are printed as SKIP lines naming the screenshot(s) and log
#      excerpt that prove them instead -- never as PASS.
#   3. There are four invocations (run with no arguments to print full
#      usage):
#
#   1. Live run, against any running server:
#        BASE_URL=https://mudpuppy-staging.up.railway.app \
#        SESSION_COOKIE="session=abc123..." \
#        CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
#        scripts/verify-phase2.sh evidence/03-canned-report.txt
#
#   2. Self-test, no server, proves the assertion logic against
#      scripts/fixtures/phase2/ (a correct-server fixture set):
#        scripts/verify-phase2.sh --self-test /tmp/phase2-selftest.txt
#
#   3. Negative self-test, proves the harness can FAIL against
#      scripts/fixtures/phase2-negative/ (one deliberately wrong response):
#        scripts/verify-phase2.sh --self-test-negative /tmp/phase2-negative.txt
#
#   4. Skip the appended `go test ./... -v` run (live mode only):
#        BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... \
#        scripts/verify-phase2.sh --no-tests report.txt
#
# CONNECTION_ID must name a profile that has NEVER accepted the Safety and
# Abuse policy -- steps 1-4 assert the unaccepted state before step 5
# accepts it. NOT_OWNED_CONNECTION_ID is optional and defaults to a fixed
# literal UUID that no profile will ever have -- a connection id the caller
# does not own and one that does not exist are indistinguishable on the
# wire, and both must be refused.
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
Usage: scripts/verify-phase2.sh [--self-test|--self-test-negative] [--no-tests] <report-file>

Drives the Phase 2 AI Player autopilot HTTP sequence and writes
<report-file> with a PASS/FAIL/SKIP line per ROADMAP criterion (C1..C4) and
every request/response body underneath.

Live run reads its inputs from the environment:

  BASE_URL                 server root, e.g. https://mudpuppy-staging.up.railway.app
  SESSION_COOKIE            full Cookie header value of an authenticated session
  CONNECTION_ID             connection_id of a profile that has NEVER accepted
                            the Safety and Abuse policy -- steps 1-4 assert the
                            unaccepted state before step 5 accepts it, so a
                            profile that has already accepted will show steps
                            1-4 as FAIL.
  NOT_OWNED_CONNECTION_ID   optional; connection_id the caller does not own.
                            Defaults to a fixed literal UUID that no profile
                            will ever have.

  BASE_URL=https://mudpuppy-staging.up.railway.app \
  SESSION_COOKIE="session=abc123..." \
  CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
  scripts/verify-phase2.sh evidence/03-canned-report.txt

--self-test          run against scripts/fixtures/phase2/ instead of a live
                      server; no environment variables required.
--self-test-negative  run against scripts/fixtures/phase2-negative/ (one
                      deliberately wrong response) to prove the harness
                      FAILs and exits non-zero on a wrong server response.
--no-tests            skip the appended `go test ./... -v` run (live mode
                      only; self-test modes always skip it).
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
  FIXTURE_DIR="scripts/fixtures/phase2"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase2-negative"
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
# indistinguishable on the wire (T-2-02); the default below is deliberate --
# no profile will ever be seeded with this id.
NOT_OWNED_CONNECTION_ID="${NOT_OWNED_CONNECTION_ID:-ffffffff-ffff-ffff-ffff-ffffffffffff}"

# ---------------------------------------------------------------------------
# Report file: everything printed from here on goes to both stdout and the
# report file, so the file is a verbatim transcript of this run.
# ---------------------------------------------------------------------------
exec > >(tee "$REPORT") 2>&1

RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================================================"
echo "Phase 2 canned-report harness -- scripts/verify-phase2.sh"
echo "Run started (UTC): $RUN_TS"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "Mode: --self-test (fixtures: scripts/fixtures/phase2/, no server)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "Mode: --self-test-negative (fixtures: scripts/fixtures/phase2-negative/, no server)"
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
echo "never a database query. (2) C2 and C3 are websocket/live-MUD behaviours"
echo "this script cannot reach without fabricating a connected game (D-03);"
echo "they appear below as SKIP lines naming their screenshot/log evidence,"
echo "never as PASS. (3) invocations: live, --self-test, --self-test-negative,"
echo "--no-tests."
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

# Ordered list of fixture files, one per _http call in the eleven-step
# sequence below. Used only when FIXTURE_DIR is set.
FIXTURE_FILES=(
  "01-status-fresh.json"
  "02-status-pre-accept.json"
  "03-on-refused-gate.json"
  "04-status-unchanged.json"
  "05-policy-accept.json"
  "06-on-refused-no-session.json"
  "07-status-post-accept.json"
  "08-off-already-off.json"
  "09-on-not-owned.json"
  "10-on-invalid-action.json"
  "11-status-final.json"
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
# Prints a line of the form "PASS C1: <description> (got: ...)" or
# "FAIL C1: <description> (expected: ..., got: ...)" for every assertion.
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

# _skip <label> <reason-and-evidence>
# Prints "SKIP C2: <text>" and increments the skip counter. A skip is
# neither a pass nor a failure and must never be counted as either (T-2-15).
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
# Prints the raw response body whole. Never call this for the policy/accept
# response -- use _print_policy_response instead so the full policy prose
# (screenshot evidence, not report content, per T-2-16) never lands in this
# file whole.
_print_response() {
  local method="$1" path="$2"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$HTTP_BODY"
}

# _print_policy_response <method> <display-path>
# Same as _print_response but truncates the policy "text" field to its first
# 80 characters before printing (T-2-16) -- the full policy body is
# screenshot evidence, not report content.
_print_policy_response() {
  local method="$1" path="$2"
  local truncated
  truncated=$(printf '%s' "$HTTP_BODY" | sed -E 's/("text"[[:space:]]*:[[:space:]]*")([^"]{0,80})([^"]*)(")/\1\2...[truncated, full text not printed to report - T-2-16]\4/')
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$truncated"
}

REFUSAL_MESSAGE="AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot."
POLICY_VERSION="1.0"

# ---------------------------------------------------------------------------
# Step 1 -- GET session/status on a fresh session
# ---------------------------------------------------------------------------
_step "Step 1: GET session/status on a fresh session"
_http GET "session/status"
_print_response GET "session/status"
AUTOPILOT_1=$(_get_field "$HTTP_BODY" autopilot_state)
_check_eq C1 "autopilot_state is off on a fresh session" "$AUTOPILOT_1" "off"

# ---------------------------------------------------------------------------
# Step 2 -- POST session/autopilot action=status before acceptance
# ---------------------------------------------------------------------------
_step "Step 2: POST session/autopilot action=status before acceptance"
_http POST "session/autopilot" "{\"action\":\"status\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
STATE_2=$(_get_field "$HTTP_BODY" state)
GATE_ALLOWED_2=$(_get_field "$HTTP_BODY" gate_allowed)
GATE_MESSAGE_2=$(_get_field "$HTTP_BODY" gate_message)
_check_eq C1 "state is off before acceptance" "$STATE_2" "off"
_check_eq C4 "gate_allowed is false before acceptance" "$GATE_ALLOWED_2" "false"
_check_eq C4 "gate_message matches the exact refusal sentence" "$GATE_MESSAGE_2" "$REFUSAL_MESSAGE"

# ---------------------------------------------------------------------------
# Step 3 -- POST session/autopilot action=on before acceptance (expect refusal)
# ---------------------------------------------------------------------------
_step "Step 3: POST session/autopilot action=on before acceptance -- #AUTO ON refused"
_http POST "session/autopilot" "{\"action\":\"on\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
OUTCOME_3=$(_get_field "$HTTP_BODY" outcome)
STATE_3=$(_get_field "$HTTP_BODY" state)
GATE_MESSAGE_3=$(_get_field "$HTTP_BODY" gate_message)
_check_eq C1 "outcome is refused-gate" "$OUTCOME_3" "refused-gate"
_check_eq C1 "state stays off on refusal" "$STATE_3" "off"
_check_eq C1 "gate_message matches the exact refusal sentence" "$GATE_MESSAGE_3" "$REFUSAL_MESSAGE"

# ---------------------------------------------------------------------------
# Step 4 -- GET session/status again (the refusal changed nothing)
# ---------------------------------------------------------------------------
_step "Step 4: GET session/status again -- the refusal changed nothing"
_http GET "session/status"
_print_response GET "session/status"
AUTOPILOT_4=$(_get_field "$HTTP_BODY" autopilot_state)
_check_eq C1 "autopilot_state is still off after the refusal" "$AUTOPILOT_4" "off"

# ---------------------------------------------------------------------------
# Step 5 -- POST profiles/{connection_id}/policy/accept (Phase 1 endpoint,
# used here only to move the fixture profile into the accepted state)
# ---------------------------------------------------------------------------
_step "Step 5: POST profiles/\$CONNECTION_ID/policy/accept -- move the fixture profile to accepted"
_http POST "profiles/${CONNECTION_ID}/policy/accept"
_print_policy_response POST "profiles/\$CONNECTION_ID/policy/accept"
ACCEPTED_5=$(_get_field "$HTTP_BODY" accepted)
ACCEPTED_VERSION_5=$(_get_field "$HTTP_BODY" accepted_version)
_check_eq C4 "policy/accept records accepted=true" "$ACCEPTED_5" "true"
_check_eq C4 "policy/accept records accepted_version=$POLICY_VERSION" "$ACCEPTED_VERSION_5" "$POLICY_VERSION"

# ---------------------------------------------------------------------------
# Step 6 -- POST session/autopilot action=on after acceptance, no connected
# game (expect refused-no-session; D-03)
# ---------------------------------------------------------------------------
_step "Step 6: POST session/autopilot action=on after acceptance, no connected game -- refused-no-session"
_http POST "session/autopilot" "{\"action\":\"on\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
OUTCOME_6=$(_get_field "$HTTP_BODY" outcome)
STATE_6=$(_get_field "$HTTP_BODY" state)
_check_eq C1 "outcome is refused-no-session once the gate passes but nothing is connected" "$OUTCOME_6" "refused-no-session"
_check_eq C1 "state stays off (the reason for refusal changed, the position did not)" "$STATE_6" "off"

# ---------------------------------------------------------------------------
# Step 7 -- POST session/autopilot action=status after acceptance (#AUTO
# STATUS's two halves: the gate result and the switch position)
# ---------------------------------------------------------------------------
_step "Step 7: POST session/autopilot action=status after acceptance -- the #AUTO STATUS line"
_http POST "session/autopilot" "{\"action\":\"status\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
GATE_ALLOWED_7=$(_get_field "$HTTP_BODY" gate_allowed)
POLICY_VERSION_7=$(_get_field "$HTTP_BODY" policy_version)
STATE_7=$(_get_field "$HTTP_BODY" state)
_check_eq C1 "gate_allowed is true after acceptance" "$GATE_ALLOWED_7" "true"
_check_eq C1 "policy_version is $POLICY_VERSION" "$POLICY_VERSION_7" "$POLICY_VERSION"
_check_eq C1 "state is off" "$STATE_7" "off"

# ---------------------------------------------------------------------------
# Step 8 -- POST session/autopilot action=off (already off; D-04 no-op)
# ---------------------------------------------------------------------------
_step "Step 8: POST session/autopilot action=off -- already-off no-op (D-04)"
_http POST "session/autopilot" "{\"action\":\"off\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
OUTCOME_8=$(_get_field "$HTTP_BODY" outcome)
STATE_8=$(_get_field "$HTTP_BODY" state)
_check_eq C1 "outcome is already-off" "$OUTCOME_8" "already-off"
_check_eq C1 "state stays off" "$STATE_8" "off"

# ---------------------------------------------------------------------------
# Step 9 -- POST session/autopilot action=on against a connection the caller
# does not own (T-2-02, ownership-scoped IDOR proof)
# ---------------------------------------------------------------------------
_step "Step 9: POST session/autopilot action=on against a not-owned connection_id -- refused, never engaged (T-2-02)"
_http POST "session/autopilot" "{\"action\":\"on\",\"connection_id\":\"${NOT_OWNED_CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
OUTCOME_9=$(_get_field "$HTTP_BODY" outcome)
GATE_ALLOWED_9=$(_get_field "$HTTP_BODY" gate_allowed)
_check_eq C4 "a not-owned connection_id is refused (outcome=refused-gate), not engaged" "$OUTCOME_9" "refused-gate"
_check_eq C4 "gate_allowed is false for a not-owned connection_id" "$GATE_ALLOWED_9" "false"

# ---------------------------------------------------------------------------
# Step 10 -- POST session/autopilot with an invalid action (expect 400)
# ---------------------------------------------------------------------------
_step "Step 10: POST session/autopilot action=sideways -- invalid action rejected with HTTP 400"
_http POST "session/autopilot" "{\"action\":\"sideways\",\"connection_id\":\"${CONNECTION_ID}\"}"
_print_response POST "session/autopilot"
_check_status C1 "an invalid action is rejected with HTTP 400" "400"

# ---------------------------------------------------------------------------
# Step 11 -- GET session/status -- the badge's source of truth and the
# switch endpoint agree, which is the refresh-correctness claim (D-10)
# ---------------------------------------------------------------------------
_step "Step 11: GET session/status -- agrees with step 8's answer (D-10 refresh correctness)"
_http GET "session/status"
_print_response GET "session/status"
AUTOPILOT_11=$(_get_field "$HTTP_BODY" autopilot_state)
if [ -n "$AUTOPILOT_11" ] && [ "$AUTOPILOT_11" != "null" ]; then
  _check C1 "autopilot_state is present (got: $AUTOPILOT_11)" 0
else
  _check C1 "autopilot_state is present (got: $AUTOPILOT_11)" 1
fi
_check_eq C1 "autopilot_state is off, matching step 8's answer" "$AUTOPILOT_11" "off"

# ---------------------------------------------------------------------------
# Steps 12-13 -- websocket/live-MUD behaviours this script cannot reach
# ---------------------------------------------------------------------------
_step "Step 12: C2 (wheel-grab) -- out of this script's reach"
_skip C2 "wheel-grab requires a connected game session (D-03); proven instead by evidence/07-wheel-grab.png, evidence/08-trigger-keeps-on.png, and the [AI-PLAYER] autopilot ... cause=wheel-grab lines in evidence/04-staging-ai-player.log"

_step "Step 13: C3 (disconnect/waiting/resume) -- out of this script's reach"
_skip C3 "disconnect-while-engaged and reconnect-resume require a connected game session (D-03); proven instead by evidence/09-waiting-after-drop.png, evidence/10-resume-after-reconnect.png, evidence/11-off-while-waiting-stays-off.png, and the cause=disconnect and cause=resume lines in evidence/04-staging-ai-player.log"

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

# ---------------------------------------------------------------------------
# Go suite: folded into the same report so one file answers both halves of
# the ROADMAP Phase Validation diagnostic line. Skipped in self-test modes
# (the fixtures are the subject of that run, not the Go suite) and when
# --no-tests is passed. Output already flows into $REPORT via the `exec >
# >(tee "$REPORT")` redirection set up above, so no separate `tee -a` is
# needed here -- that would duplicate every line into the report.
# ---------------------------------------------------------------------------
GO_TEST_EXIT=0
echo
echo "================================================================"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "GO TEST: skipped (--self-test mode -- the fixtures are the subject of this run, not the Go suite)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "GO TEST: skipped (--self-test-negative mode -- the fixtures are the subject of this run, not the Go suite)"
elif [ "$NO_TESTS" -eq 1 ]; then
  echo "GO TEST: skipped (--no-tests)"
else
  echo "GO TEST: go test ./... -v"
  echo "================================================================"
  ( cd "$(git rev-parse --show-toplevel 2>/dev/null || echo .)" && go test ./... -v )
  GO_TEST_EXIT=$?
  echo "GO TEST EXIT: $GO_TEST_EXIT"
fi
echo "================================================================"

if [ "$FAIL_COUNT" -eq 0 ] && [ "$GO_TEST_EXIT" -eq 0 ]; then
  exit 0
else
  exit 1
fi
