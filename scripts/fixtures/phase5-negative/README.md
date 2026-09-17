# scripts/fixtures/phase5-negative/ -- one deliberately wrong response

An exact copy of `scripts/fixtures/phase5/` except for one file,
`13-ai-settings-get-after-inbounds.json`, whose `rate_limit_per_second`
reads back `7` instead of the `5` that was just PUT in
`12-ai-settings-put-inbounds.json`. This makes
`scripts/verify-phase5.sh --self-test-negative` produce `FAIL C3` ("the set
rate limit round-trips through PUT and GET") with a non-zero exit, proving
the harness can fail before its report is trusted (T-5-45).

See `scripts/fixtures/phase5/README.md` for every other file's shape.
