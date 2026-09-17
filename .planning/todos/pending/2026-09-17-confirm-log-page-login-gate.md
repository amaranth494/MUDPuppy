---
created: 2026-09-17
source: Phase 4 security review, owner's condition on register item R-10 (AR-4-08 in .planning/RISK-REGISTER.md), Phase 4 Risk Register https://claude.ai/artifact/SvSdbXCa3YXmFDryiXpAaK
resolves_phase: 5
---

# Confirm the log page itself is gated behind sign-in

The owner accepted R-10 ("Nobody has checked who can read the staging log") at the Phase 4 security review with this note, verbatim:

> Not concerned about Admins, however the linked display needs to be gated behind the login of the person in the browser tool

## What is already known

The log display's data is gated. Every log and decision read is under `/api/v1/`, and `sessionMiddleware` in `cmd/server/main.go` answers 401 without a valid session, so a signed-out browser cannot read anyone's log or decisions.

## The task

Confirm that the `/logs/:connectionId` page itself redirects to sign-in when the visitor is signed out, rather than rendering an empty or broken page around the refused data requests.

- Read the frontend route for `/logs/:connectionId` and check whether it sits behind the same signed-in guard the rest of the app uses.
- Evidence must be an end-user screenshot or a canned report (a database query does not count): open the log page URL in a browser with no session and capture where it lands.
- Do NOT sign the owner out to take the screenshot. Use a private window or a separate browser profile; signing out is the owner's step only.
- If the page does not redirect, add the guard and a test, and record the outcome against AR-4-08 at the Phase 5 security review.
