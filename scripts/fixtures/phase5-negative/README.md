# scripts/fixtures/phase5-negative/ -- one deliberately wrong response

An exact copy of `scripts/fixtures/phase5/` except for one file,
`13-ai-settings-get-after-inbounds.json`, whose `rate_limit_per_second`
reads back `7` instead of the `5` that was just PUT in
`12-ai-settings-put-inbounds.json`. This makes
`scripts/verify-phase5.sh --self-test-negative` produce `FAIL C3` ("the set
rate limit round-trips through PUT and GET") with a non-zero exit, proving
the harness can fail before its report is trusted (T-5-45).

It is still the ONLY wrong response: files 11, 12, 14, 15, 17 and 18 are
byte-identical to the correct set's, so the whole-body restore comparison
code review WR-14 of Phase 5 added passes here and the run ends with exactly
one `FAIL C3`. (That comparison's own ability to fail is a one-character
change to file 18 away; it was exercised when the check was written and is
described in `05-REVIEW-FIX.md`.)

See `scripts/fixtures/phase5/README.md` for every other file's shape, and for
why the settings text in files 11 to 18 is awkward on purpose.
