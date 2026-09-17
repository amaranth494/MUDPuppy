# scripts/fixtures/phase5/ -- correct-server fixture set

One file per HTTP call `scripts/verify-phase5.sh --self-test` makes, in call
order (the harness's `FIXTURE_FILES` array). Each file's first line is the
HTTP status code; every line after that is the response body, exactly as
`_http`'s fixture branch expects (mirroring `scripts/fixtures/phase4/`'s
convention). Shaped from the real handlers' response structs in
`internal/session/handler.go` and `internal/profiles/handler.go` -- not
from imagination -- so a later shape change in those files is traceable
back to the fixture that would need updating.

| File | Handler mirrored | Notes |
|------|-------------------|-------|
| `01-session-status-get.json` | `Status` | `autopilot_state: "on"` so the self-test exercises the full C1 pause/resume path rather than skipping it |
| `02-autopilot-pause.json` | `Autopilot` (`case "pause"`) | Engaged switch pauses: `state=waiting`, `outcome=paused`, `paused_by_owner=true` |
| `03-autopilot-pause-again.json` | `Autopilot` (`case "pause"`) | A repeated pause changes nothing: `outcome=already-waiting`, both reasons unchanged |
| `04-autopilot-resume.json` | `Autopilot` (`case "resume"`) | Resume clears both reasons: `state=on`, `outcome=resumed`, both false |
| `05-ai-coaching-get.json` | `GetCoaching` | A non-empty `coaching` array (never `null`) |
| `06-ai-conversation-get.json` | `GetConversation` | A non-empty `lines` array of `{id, speaker, text, timestamp}` objects (never `null`) |
| `07-ai-coaching-put-not-allowed.json` | the `ai-coaching` route's method switch in `cmd/server/main.go` (no PUT handler exists) | Plain-text `405` body, matching `http.Error`'s own output -- not JSON |
| `08-ai-coaching-delete-not-allowed.json` | same route, DELETE | Plain-text `405` |
| `09-ai-conversation-put-not-allowed.json` | the `ai-conversation` route's method switch (no PUT handler exists) | Plain-text `405` |
| `10-ai-conversation-delete-not-allowed.json` | same route, DELETE | Plain-text `405` |
| `11-ai-settings-get-original.json` | `GetAISettings` | The pre-run settings the restore step (step 17) puts back; `rate_limit_per_second` starts blank (`null`) |
| `12-ai-settings-put-inbounds.json` | `PutAISettings` | Accepts `rate_limit_per_second: 5`, inside the 1-20 bound |
| `13-ai-settings-get-after-inbounds.json` | `GetAISettings` | The set value (5) reads back. **This is the one file `scripts/fixtures/phase5-negative/` changes.** |
| `14-ai-settings-put-blank.json` | `PutAISettings` | Accepts `rate_limit_per_second: null` |
| `15-ai-settings-get-after-blank.json` | `GetAISettings` | Blank round-trips as `null` -- resolved to the server default in Go, not the browser (D-25) |
| `16-ai-settings-put-out-of-bounds.json` | `PutAISettings` (`validateAISettings`) | `rate_limit_per_second: 25` is refused with `400` and the exact bounds sentence (T-5-17) |
| `17-ai-settings-put-restore.json` | `PutAISettings` | Restoring the original (blank) rate limit |
| `18-ai-settings-get-after-restore.json` | `GetAISettings` | The restore matches what step 11 found |
| `19-ai-coaching-get-not-owned.json` | `GetCoaching` via `getProfileByConnectionID` | A not-owned/non-existent connection is refused with `400`, never `200` (T-4-09) |
| `20-ai-conversation-get-not-owned.json` | `GetConversation` via `getProfileByConnectionID` | Same refusal shape as above |

No file in this directory contains a session-cookie value, an API key, or
any game text (T-4-05) -- confirmed by the acceptance check that greps this
directory for credential-shaped substrings and expects zero matches.
