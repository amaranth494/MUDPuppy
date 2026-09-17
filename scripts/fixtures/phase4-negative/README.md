# scripts/fixtures/phase4/ -- correct-server fixture set

One file per HTTP call `scripts/verify-phase4.sh --self-test` makes, in call
order (the harness's `FIXTURE_FILES` array). Each file's first line is the
HTTP status code; every line after that is the response body, exactly as
`_http`'s fixture branch expects (mirroring `scripts/fixtures/phase3-1/`'s
convention). Shaped from the real handlers' response structs in
`internal/profiles/handler.go` and `internal/profiles/decisions.go` -- not
from imagination -- so a later shape change in those files is traceable
back to the fixture that would need updating.

| File | Handler mirrored | Notes |
|------|-------------------|-------|
| `01-ai-goal-get-original.json` | `GetGoal` | The pre-run goal the restore step (step 16) puts back |
| `02-ai-goal-put-a.json` | `PutGoal` | A fresh goal names its Quest `"created"` (`GoalResponse.Quest`, D-04) |
| `03-ai-goal-get-after-a.json` | `GetGoal` | `GetGoal` never sets `quest` (omitted via `omitempty`) |
| `04-ai-goal-put-b.json` | `PutGoal` | A second, different goal -- proves an edit takes (D-03) |
| `05-ai-goal-get-after-b.json` | `GetGoal` | |
| `06-ai-goal-put-reactivate.json` | `PutGoal` | The first goal repeated, re-cased and padded, normalizes to the same Quest and names it `"reactivated"`, not `"created"` (D-04) |
| `07-ai-goal-put-too-long.json` | `PutGoal` (`maxGoalLength` validation) | A 1001-character goal is refused with `400` and the exact cap sentence (T-4-10) |
| `08-ai-memory-get.json` | `GetSessionMemory` | A well-formed `session_memory` array (D-10) |
| `09-ai-memory-put-not-allowed.json` | the `ai-memory` route's method switch in `cmd/server/main.go` (no PUT handler exists) | Plain-text `405` body, matching `http.Error`'s own output -- not JSON |
| `10-decisions-get-before.json` | `GetDecisions` | The most recent decision before the delete: non-empty `reasoning`, an `id` to compare against after |
| `11-captured-text-delete.json` | `DeleteCapturedText` | Both retention counts (`snapshots_cleared`, `transcript_lines_deleted`), D-21 |
| `12-decisions-get-after.json` | `GetDecisions` | The same decision, `reasoning`/`command`/`outcome`/`failure_kind` unchanged -- the audit record survives the prune (D-21). **This is the one file `scripts/fixtures/phase4-negative/` changes.** |
| `13-ai-goal-get-not-owned.json` | `GetGoal` via `getProfileByConnectionID` | A not-owned/non-existent connection is refused with `400`, never `200` (T-4-09) |
| `14-ai-memory-get-not-owned.json` | `GetSessionMemory` via `getProfileByConnectionID` | Same refusal shape as above |
| `15-captured-text-delete-not-owned.json` | `DeleteCapturedText` via `getProfileByConnectionID` | Same refusal shape as above -- the destructive endpoint is never reached for a connection the caller does not own |
| `16-ai-goal-put-restore.json` | `PutGoal` | Restoring the original goal (step 1) reactivates its own prior Quest, if any |

No file in this directory contains a session-cookie value, an API key, or
any game text (T-4-05) -- confirmed by the acceptance check that greps this
directory for credential-shaped substrings and expects zero matches.
