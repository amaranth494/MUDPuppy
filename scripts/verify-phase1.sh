#!/usr/bin/env bash
#
# scripts/verify-phase1.sh -- Phase 1 canned-report harness (AI Player:
# profile foundation and policy gate).
#
# Drives the five ai-settings/policy/policy-accept/engage-gate/timers
# endpoints against a running MUDPuppy server and writes a report file that
# states, criterion by criterion (C1..C4, matching ROADMAP.md's Phase 1
# Success Criteria 1-4), whether Phase 1 holds -- with the raw request path
# and response body printed under every step. This IS the canned report the
# owner's evidence rule requires; no database query is used anywhere in this
# script.
#
# Four invocations (run with no arguments to print full usage):
#
#   1. Live run, against any running server:
#        BASE_URL=https://mudpuppy-staging.up.railway.app \
#        SESSION_COOKIE="session=abc123..." \
#        CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
#        scripts/verify-phase1.sh evidence/03-canned-report.txt
#
#   2. Self-test, no server, proves the assertion logic against
#      scripts/fixtures/phase1/ (a correct-server fixture set):
#        scripts/verify-phase1.sh --self-test /tmp/phase1-selftest.txt
#
#   3. Negative self-test, proves the harness can FAIL against
#      scripts/fixtures/phase1-negative/ (one deliberately wrong response):
#        scripts/verify-phase1.sh --self-test-negative /tmp/phase1-negative.txt
#
#   4. Skip the appended `go test ./... -v` run (live mode only):
#        BASE_URL=... SESSION_COOKIE=... CONNECTION_ID=... \
#        scripts/verify-phase1.sh --no-tests report.txt
#
# CONNECTION_ID must name a profile that has NEVER accepted the Safety and
# Abuse policy -- steps 1-5 assert the unaccepted state before step 6
# accepts it.
#
set -uo pipefail   # errexit is intentionally omitted: a failed check must not
                    # abort the report -- the report must list every result,
                    # not stop at the first FAIL.

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
SELF_TEST=0
SELF_TEST_NEGATIVE=0
NO_TESTS=0
REPORT=""

usage() {
  cat >&2 <<'USAGE'
Usage: scripts/verify-phase1.sh [--self-test|--self-test-negative] [--no-tests] <report-file>

Drives the Phase 1 AI Player HTTP sequence and writes <report-file> with a
PASS/FAIL line per ROADMAP criterion (C1..C4) and every request/response
body underneath.

Live run reads its inputs from the environment:

  BASE_URL        server root, e.g. https://mudpuppy-staging.up.railway.app
  SESSION_COOKIE  full Cookie header value of an authenticated session
  CONNECTION_ID   connection_id of a profile that has NEVER accepted the
                  Safety and Abuse policy -- steps 1-5 assert the unaccepted
                  state before step 6 accepts it, so a profile that has
                  already accepted will show step 1/2/5 as FAIL.

  BASE_URL=https://mudpuppy-staging.up.railway.app \
  SESSION_COOKIE="session=abc123..." \
  CONNECTION_ID=11111111-2222-3333-4444-555555555555 \
  scripts/verify-phase1.sh evidence/03-canned-report.txt

--self-test          run against scripts/fixtures/phase1/ instead of a live
                      server; no environment variables required.
--self-test-negative  run against scripts/fixtures/phase1-negative/ (one
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
  FIXTURE_DIR="scripts/fixtures/phase1"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  FIXTURE_DIR="scripts/fixtures/phase1-negative"
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

# ---------------------------------------------------------------------------
# Report file: everything printed from here on goes to both stdout and the
# report file, so the file is a verbatim transcript of this run.
# ---------------------------------------------------------------------------
exec > >(tee "$REPORT") 2>&1

RUN_TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "================================================================"
echo "Phase 1 canned-report harness -- scripts/verify-phase1.sh"
echo "Run started (UTC): $RUN_TS"
if [ "$SELF_TEST" -eq 1 ]; then
  echo "Mode: --self-test (fixtures: scripts/fixtures/phase1/, no server)"
elif [ "$SELF_TEST_NEGATIVE" -eq 1 ]; then
  echo "Mode: --self-test-negative (fixtures: scripts/fixtures/phase1-negative/, no server)"
else
  echo "Mode: live"
fi
echo "BASE_URL: $BASE_URL"
echo "CONNECTION_ID: $CONNECTION_ID"
echo "git rev-parse --short HEAD: $GIT_SHA"
echo "================================================================"
echo

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

# _get_field <json> <field>
# Extracts one field's value from a flat or one-level-nested JSON object.
# <field> may be dotted (e.g. "ai_settings.model_name") for the jq path; the
# non-jq fallback greps on the leaf name only (sufficient here because every
# leaf name this script reads is unique across the response shapes it uses).
# Prints "null" for a JSON null. Prints the empty string via the non-jq
# fallback when the key is altogether absent from the text (e.g. an
# `omitempty` field that was omitted).
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

# Ordered list of fixture files, one per _http call in the sequence below
# (15 calls across the 10 numbered steps -- several steps make more than
# one call, e.g. step 3's PUT then GET). Used only when FIXTURE_DIR is set.
FIXTURE_FILES=(
  "01-policy-unaccepted.json"
  "02-engage-gate-refused.json"
  "03-ai-settings-put-blank.json"
  "04-ai-settings-get-blank.json"
  "05-ai-settings-put-overlength.json"
  "06-ai-settings-put-forgery.json"
  "07-policy-post-forgery.json"
  "08-policy-accept-first.json"
  "09-policy-accept-repeat.json"
  "10-engage-gate-allowed.json"
  "11-ai-settings-put-populated.json"
  "12-ai-settings-get-populated.json"
  "13-timers-get.json"
  "14-timers-put.json"
  "15-ai-settings-get-post-timers.json"
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
declare -A CRIT_FAILED
declare -A CRIT_SEEN

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

_step() {
  echo
  echo "---- $* ----"
}

# _print_response <method> <display-path>
# Prints the raw response body. Never call this for a policy response -- use
# _print_policy_response instead so the full policy text (screenshot
# evidence, not report content, per threat T-1-11) never lands in this file.
_print_response() {
  local method="$1" path="$2"
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$HTTP_BODY"
}

# _print_policy_response <method> <display-path>
# Same as _print_response but truncates the policy "text" field to its first
# 80 characters before printing (T-1-11) -- the full policy body is the
# owner's screenshot evidence, not report content.
_print_policy_response() {
  local method="$1" path="$2"
  local truncated
  truncated=$(printf '%s' "$HTTP_BODY" | sed -E 's/("text"[[:space:]]*:[[:space:]]*")([^"]{0,80})([^"]*)(")/\1\2...[truncated, full text not printed to report - T-1-11]\4/')
  echo "$method /api/v1/$path -> HTTP $HTTP_STATUS"
  echo "$truncated"
}

REFUSAL_MESSAGE="AI Player has not been configured for this connection. Accept the Safety and Abuse policy in AI Player settings before engaging autopilot."
OVERLENGTH_MESSAGE="Conduct rules must be 20000 characters or less"
POLICY_VERSION="1.0"

# ---------------------------------------------------------------------------
# Step 1 -- GET policy (must be unaccepted)
# ---------------------------------------------------------------------------
_step "Step 1: GET policy on a fresh, never-accepted profile"
_http GET "profiles/${CONNECTION_ID}/policy"
_print_policy_response GET "profiles/\$CONNECTION_ID/policy"
ACCEPTED_1=$(_get_field "$HTTP_BODY" accepted)
VERSION_1=$(_get_field "$HTTP_BODY" version)
_check_eq C3 "policy shows accepted=false before any acceptance" "$ACCEPTED_1" "false"
_check_eq C4 "policy version equals $POLICY_VERSION" "$VERSION_1" "$POLICY_VERSION"

# ---------------------------------------------------------------------------
# Step 2 -- GET engage-gate before acceptance (expect refusal)
# ---------------------------------------------------------------------------
_step "Step 2: GET engage-gate before acceptance (expect refusal)"
_http GET "profiles/${CONNECTION_ID}/engage-gate"
_print_response GET "profiles/\$CONNECTION_ID/engage-gate"
ALLOWED_2=$(_get_field "$HTTP_BODY" allowed)
MESSAGE_2=$(_get_field "$HTTP_BODY" message)
_check_eq C3 "engage-gate refuses before acceptance (allowed=false)" "$ALLOWED_2" "false"
_check_eq C3 "engage-gate refusal message matches the exact Phase 2 contract string" "$MESSAGE_2" "$REFUSAL_MESSAGE"

# ---------------------------------------------------------------------------
# Step 3 -- PUT ai-settings blank, then GET -- blanks round-trip unchanged
# ---------------------------------------------------------------------------
_step "Step 3: PUT ai-settings blank, then GET -- blanks round-trip unchanged"
_http PUT "profiles/${CONNECTION_ID}/ai-settings" '{"conduct_rules":"","approach_guidance":"","ai_settings":{"model_name":"","call_cap":null,"disengage_threshold":null}}'
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
MODEL_3=$(_get_field "$HTTP_BODY" ai_settings.model_name)
CALLCAP_3=$(_get_field "$HTTP_BODY" ai_settings.call_cap)
THRESH_3=$(_get_field "$HTTP_BODY" ai_settings.disengage_threshold)
_check_eq C1 "GET after blank PUT returns empty model_name" "$MODEL_3" ""
_check_eq C1 "GET after blank PUT returns call_cap=null (not 0)" "$CALLCAP_3" "null"
_check_eq C1 "GET after blank PUT returns disengage_threshold=null (not 0)" "$THRESH_3" "null"
AI_SETTINGS_OBJ_3=$(printf '%s' "$HTTP_BODY" | grep -oE '"ai_settings"[[:space:]]*:[[:space:]]*\{[^}]*\}' | sed -E 's/^"ai_settings"[[:space:]]*:[[:space:]]*//')
KEY_COUNT_3=$(printf '%s' "$AI_SETTINGS_OBJ_3" | grep -oE '"[a-z_]+"[[:space:]]*:' | sort -u | wc -l | tr -d ' ')
_check_eq C1 "ai_settings object has exactly the three keys model_name/call_cap/disengage_threshold" "$KEY_COUNT_3" "3"

# ---------------------------------------------------------------------------
# Step 4 -- PUT ai-settings with conduct_rules over 20000 characters
# ---------------------------------------------------------------------------
_step "Step 4: PUT ai-settings with conduct_rules of 20001 characters -- expect 400 rejection"
OVERLENGTH=$(printf 'x%.0s' $(seq 1 20001))
_http PUT "profiles/${CONNECTION_ID}/ai-settings" "{\"conduct_rules\":\"${OVERLENGTH}\",\"approach_guidance\":\"\",\"ai_settings\":{\"model_name\":\"\",\"call_cap\":null,\"disengage_threshold\":null}}"
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_check_status C1 "over-length conduct_rules rejected with HTTP 400" "400"
EXPECTED_ERROR_BODY="{\"error\":\"${OVERLENGTH_MESSAGE}\"}"
_check_eq C1 "over-length rejection body matches the exact error string" "$HTTP_BODY" "$EXPECTED_ERROR_BODY"

# ---------------------------------------------------------------------------
# Step 5 -- forged acceptance fields on an ai-settings PUT must not stick
# ---------------------------------------------------------------------------
_step "Step 5: PUT ai-settings carrying forged acceptance fields, then GET policy -- acceptance must not be forgeable"
_http PUT "profiles/${CONNECTION_ID}/ai-settings" '{"conduct_rules":"","approach_guidance":"","ai_settings":{"model_name":"","call_cap":null,"disengage_threshold":null},"policy_accepted_at":"2020-01-01T00:00:00Z","policy_version_accepted":"9.9"}'
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_http GET "profiles/${CONNECTION_ID}/policy"
_print_policy_response GET "profiles/\$CONNECTION_ID/policy"
ACCEPTED_5=$(_get_field "$HTTP_BODY" accepted)
ACCEPTED_VERSION_5=$(_get_field "$HTTP_BODY" accepted_version)
_check_eq C4 "policy still shows accepted=false after an ai-settings PUT carrying forged acceptance fields" "$ACCEPTED_5" "false"
_check_eq C4 "accepted_version is still null after the forgery attempt (not 9.9)" "$ACCEPTED_VERSION_5" "null"

# ---------------------------------------------------------------------------
# Step 6 -- POST policy/accept -- first acceptance
# ---------------------------------------------------------------------------
_step "Step 6: POST policy/accept -- first acceptance"
_http POST "profiles/${CONNECTION_ID}/policy/accept"
_print_policy_response POST "profiles/\$CONNECTION_ID/policy/accept"
ACCEPTED_6=$(_get_field "$HTTP_BODY" accepted)
ACCEPTED_VERSION_6=$(_get_field "$HTTP_BODY" accepted_version)
ACCEPTED_AT_6=$(_get_field "$HTTP_BODY" accepted_at)
_check_eq C4 "first accept records accepted=true" "$ACCEPTED_6" "true"
_check_eq C4 "first accept records accepted_version=$POLICY_VERSION" "$ACCEPTED_VERSION_6" "$POLICY_VERSION"
if [ -n "$ACCEPTED_AT_6" ] && [ "$ACCEPTED_AT_6" != "null" ]; then
  _check C4 "first accept records a non-empty accepted_at timestamp (got: $ACCEPTED_AT_6)" 0
else
  _check C4 "first accept records a non-empty accepted_at timestamp (got: $ACCEPTED_AT_6)" 1
fi
FIRST_ACCEPTED_AT="$ACCEPTED_AT_6"

# ---------------------------------------------------------------------------
# Step 7 -- POST policy/accept again -- one-time acceptance
# ---------------------------------------------------------------------------
_step "Step 7: POST policy/accept again -- one-time acceptance, proven by a repeat call"
_http POST "profiles/${CONNECTION_ID}/policy/accept"
_print_policy_response POST "profiles/\$CONNECTION_ID/policy/accept"
ACCEPTED_AT_7=$(_get_field "$HTTP_BODY" accepted_at)
ACCEPTED_VERSION_7=$(_get_field "$HTTP_BODY" accepted_version)
_check_eq C4 "repeat accept echoes the original accepted_at timestamp unchanged" "$ACCEPTED_AT_7" "$FIRST_ACCEPTED_AT"
_check_eq C4 "repeat accept still shows accepted_version=$POLICY_VERSION" "$ACCEPTED_VERSION_7" "$POLICY_VERSION"

# ---------------------------------------------------------------------------
# Step 8 -- GET engage-gate after acceptance (expect allow)
# ---------------------------------------------------------------------------
_step "Step 8: GET engage-gate after acceptance (expect allow)"
_http GET "profiles/${CONNECTION_ID}/engage-gate"
_print_response GET "profiles/\$CONNECTION_ID/engage-gate"
ALLOWED_8=$(_get_field "$HTTP_BODY" allowed)
MESSAGE_8=$(_get_field "$HTTP_BODY" message)
_check_eq C3 "engage-gate allows after acceptance (allowed=true)" "$ALLOWED_8" "true"
if [ -z "$MESSAGE_8" ] || [ "$MESSAGE_8" = "null" ]; then
  _check C3 "engage-gate carries no refusal message once allowed (message absent/empty)" 0
else
  _check C3 "engage-gate carries no refusal message once allowed (message absent/empty, got: $MESSAGE_8)" 1
fi

# ---------------------------------------------------------------------------
# Step 9 -- PUT ai-settings with populated values, then GET -- round-trip
# ---------------------------------------------------------------------------
_step "Step 9: PUT ai-settings with populated values, then GET -- values round-trip identically"
_http PUT "profiles/${CONNECTION_ID}/ai-settings" '{"conduct_rules":"no PKing","approach_guidance":"level cautiously","ai_settings":{"model_name":"gemini-2.5-pro","call_cap":25,"disengage_threshold":2}}'
_print_response PUT "profiles/\$CONNECTION_ID/ai-settings"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
CONDUCT_9=$(_get_field "$HTTP_BODY" conduct_rules)
APPROACH_9=$(_get_field "$HTTP_BODY" approach_guidance)
MODEL_9=$(_get_field "$HTTP_BODY" ai_settings.model_name)
CALLCAP_9=$(_get_field "$HTTP_BODY" ai_settings.call_cap)
THRESH_9=$(_get_field "$HTTP_BODY" ai_settings.disengage_threshold)
_check_eq C2 "conduct_rules round-trips identically" "$CONDUCT_9" "no PKing"
_check_eq C2 "approach_guidance round-trips identically" "$APPROACH_9" "level cautiously"
_check_eq C2 "model_name round-trips identically" "$MODEL_9" "gemini-2.5-pro"
_check_eq C2 "call_cap round-trips identically" "$CALLCAP_9" "25"
_check_eq C2 "disengage_threshold round-trips identically" "$THRESH_9" "2"

# ---------------------------------------------------------------------------
# Step 10 -- unrelated timers save leaves AI fields intact (non-interference)
# ---------------------------------------------------------------------------
_step "Step 10: GET timers, PUT the same body back unchanged, then GET ai-settings -- an unrelated save leaves AI fields intact"
_http GET "profiles/${CONNECTION_ID}/timers"
_print_response GET "profiles/\$CONNECTION_ID/timers"
TIMERS_BODY="$HTTP_BODY"
_http PUT "profiles/${CONNECTION_ID}/timers" "$TIMERS_BODY"
_print_response PUT "profiles/\$CONNECTION_ID/timers"
_http GET "profiles/${CONNECTION_ID}/ai-settings"
_print_response GET "profiles/\$CONNECTION_ID/ai-settings"
CONDUCT_10=$(_get_field "$HTTP_BODY" conduct_rules)
APPROACH_10=$(_get_field "$HTTP_BODY" approach_guidance)
MODEL_10=$(_get_field "$HTTP_BODY" ai_settings.model_name)
CALLCAP_10=$(_get_field "$HTTP_BODY" ai_settings.call_cap)
THRESH_10=$(_get_field "$HTTP_BODY" ai_settings.disengage_threshold)
_check_eq C2 "conduct_rules unchanged after an unrelated timers save" "$CONDUCT_10" "no PKing"
_check_eq C2 "approach_guidance unchanged after an unrelated timers save" "$APPROACH_10" "level cautiously"
_check_eq C2 "model_name unchanged after an unrelated timers save" "$MODEL_10" "gemini-2.5-pro"
_check_eq C2 "call_cap unchanged after an unrelated timers save" "$CALLCAP_10" "25"
_check_eq C2 "disengage_threshold unchanged after an unrelated timers save" "$THRESH_10" "2"

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo
echo "================================================================"
echo "SUMMARY"
echo "================================================================"
for label in C1 C2 C3 C4; do
  if [ "${CRIT_SEEN[$label]:-}" != "1" ]; then
    echo "$label: NOT EXERCISED"
  elif [ "${CRIT_FAILED[$label]:-}" = "1" ]; then
    echo "$label: FAIL"
  else
    echo "$label: PASS"
  fi
done
echo "TOTAL CHECKS: $TOTAL_COUNT  FAILURES: $FAIL_COUNT"

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
