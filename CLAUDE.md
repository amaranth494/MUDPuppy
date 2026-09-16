# MUDPuppy — AI Game Player (branch `ai-player`)

## Development method (binding)

All planning and execution on this branch follows `.specify/memory/phase-based-development-approach.md`. Read it before creating or checking a roadmap, plan, or verification report. The hierarchy is **Phase → Wave → Plan → Task**, and progress is measured by demonstrated changes in capability, not by work performed.

Distilled rules every GSD agent must honor:

1. **Phases describe outcomes.** A phase goal states what will be true afterward that is not true now. Never "work on X".
2. **Every phase is provable.** Success criteria must be verifiable by player-observable behavior (in the browser client against a live MUD) or diagnostic verification (`go test`, a diagnostic `#` directive, an API call, database inspection, logs). Each ROADMAP phase carries a **Phase Validation** line saying exactly how the criteria will be demonstrated. A phase is complete only when that demonstration has happened, not when its tasks are done.
3. **Plans change the system.** A plan is named for the capability it creates, never for a layer or category of work ("backend", "database work", "frontend wiring" are not plans). Each plan must leave an observable, independently evaluable difference in the system.
4. **Every plan has acceptance criteria** that describe observable results, executable through tests, developer tools, diagnostic commands, the UI, logs, or database inspection. "Code is complete" is not a criterion. In PLAN.md this lives in `must_haves.truths` and `<success_criteria>`; per-task `<acceptance_criteria>` are additional, not a substitute.
5. **Waves are dependency order only.** A plan sits in the earliest wave where all its `depends_on` are satisfied. Plans in the same wave must be independently executable. Never order waves for convenience.
6. **Tasks are execution units** that implement a plan; they need not carry their own acceptance criteria but must be concrete and traceable to the plan.
7. **Prefer a diagnostic surface per plan.** Where no player-facing UI exists yet, add the test, directive, endpoint, or inspection step that proves the capability.

Plan-checker: reject plans that are categories of work, plans whose criteria only assert that code exists, and wave assignments not derived from dependencies.
Verifier: verify against the phase's Phase Validation line and success criteria, not against task completion.

## Project facts

- Source-of-truth documents: `.specify/specs/ai-game-player-design-v3.md` and `.specify/specs/safety-and-abuse-policy-v1.md`. All other `.specify/` specs and the constitution predate this effort and are treated as null; learn the existing system from the code.
- The product is taken as-is. The AI driver runs server-side in Go and issues commands through the ICM automation context. The browser is the supervision surface. Reconnect is a connection-profile toggle, never an AI decision.
- Environment: Railway project `mudpuppy`, environment `staging`. Production is untouched until final acceptance.
- Push `ai-player` to GitHub (`git push origin ai-player`) at every phase close, after the security review commits land; the owner asked for this at the Phase 3 review (DR-3 R-13). Railway's dashboard Deploy rebuilds from GitHub, so an unpushed branch makes a dashboard deploy crash staging; day-to-day staging deploys stay `railway up` from the local checkout.
- Stack: Go 1.26 backend, React + TypeScript + Vite frontend served as a static SPA, Postgres via golang-migrate migrations, Redis.
