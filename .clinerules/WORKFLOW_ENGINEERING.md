# Engineering Working Pattern (MANDATORY)

This file describes HOW changes are designed, reviewed, tested and gated. It is
the engineering counterpart to the process files and defers to them for their
own mechanics:

- `WORKFLOW_GIT.md` — branches, commits, PR/issue lifecycle, publishing a review.
- `WORKFLOW_GIT_EPIC.md` — project board, integration/release flow, status model,
  the pilot/reviewer principle, protection of the default/release branch.
- `WORKSPACE_CLINE_ADVANCE_RULES.md` — memory bank protocol.

Those files own VCS, board and memory. **This file owns the engineering
discipline**: the adversarial review cycle, the reviewer invocation, the test
discipline, the state-safety doctrine, and the human gates. It is intentionally
**generic and project-agnostic** — concrete tools, commands, paths, technologies
and naming conventions belong in each project's own configuration, not here. The
single project-specific value is the reviewer declared in §3.

## 1. Roles: pilot and reviewer

Per `WORKFLOW_GIT_EPIC.md` ("Mandatory reviews"), every change separates a
**pilot** (produces the change) from an independent **reviewer** (challenges
it). The roles are not tied to a specific tool, and the same agent never holds
both when a second one is available; the default reviewer is an **independent
adversarial model** (declared in §3). Every change is reviewed — including
trivial ones — and the review *depth* scales with risk (the full cycle below,
with a PLAN review, for state-touching or risky work).

## 2. The mandatory review cycle

Steps 2–8 are mandatory for every change and are never compressed. Step 1 is
additionally mandatory whenever the change touches persisted state (adding,
dropping or migrating tracked entries; read / build / serialize paths) or
carries non-trivial risk.

1. **PLAN review (before coding)** — submit the plan to the adversarial reviewer
   with the attack questions enumerated explicitly. Resolve every NO-GO before
   writing code.
2. **Implement** on a dedicated branch (see `WORKFLOW_GIT.md`).
3. **Pre-commit adversarial review** of the working change.
4. **Commit / push**.
5. **Independent reviewer pass on the committed diff** — feed the real diff, the
   verified context, the state-safety doctrine (§5), the test evidence (§4: the
   mutation exercised, the RED-without-fix / GREEN-with-fix proof) and an
   explicit, enumerated list of attack questions ending in a structured GO /
   NO-GO request.
6. **Verdict = verbatim** — record the reviewer's verdict as-is. Never write the
   verdict on the reviewer's behalf; never soften, paraphrase or compress it.
7. **Validate the adjudication, do not impose it** — check each finding against
   the real code; address every finding by severity (blocking / minor / nit);
   re-review the delta when the change is non-trivial.
8. **Merge is conditioned on a favorable verdict AND on the human GO** (§6).
   **Hold the merge** for any state-critical change or remaining doubt.

Sub-agent or otherwise delegated deliverables ALWAYS get an independent review
with the verdict recorded verbatim — never rubber-stamp an untraced "reviewer
GO".

## 3. Reviewer invocation

> **Reviewer (project configuration — set this per repository):**
> - **Model:** `Codex — gpt-5.5, reasoning effort: high, service tier: fast`
> - **Invocation:** `codex exec --sandbox read-only - < /tmp/prompt.md`
>   — the model/effort/tier are authoritative in the reviewer's own config file;
>   do NOT pass ad-hoc model/effort overrides on the command line.
>
> This block is the only project-specific part of this file. Replace it with the
> reviewer and invocation the project uses; everything below stays tool-agnostic.

Independent of the reviewing tool (CLI, API, CI job, review service):

- **Read-only** — the reviewer inspects, it does not modify the workspace.
- **Non-interactive** — drive it through a non-interactive entry point (for
  example, the prompt supplied on a non-interactive input stream); an
  interactive call can hang in a non-interactive execution environment.
- **Full context in, structured verdict out** — the prompt carries the verified
  context, the diff or plan, the state-safety doctrine (§5), the test evidence
  (§4), and an explicit, enumerated list of attack questions ending in a GO /
  NO-GO request.
- **Configuration is centralized** — the model, reasoning effort and any service
  options live in the reviewer's own configuration (declared above), not
  scattered as ad-hoc per-call overrides.
- **No bypass** — if the reviewer cannot be called (confidentiality, or the
  tooling refuses even after user authorization), do a local review, record the
  exception, and continue only if the decision stays reversible or is explicitly
  validated (per `WORKFLOW_GIT_EPIC.md`).

## 4. Test discipline (non-complacent, mutation-proven)

- **Every fix or guard ships with a test that goes RED without it.** Reproduce
  the real failure first (crash, wrong state change, missing error), then prove
  the guard turns it GREEN. Record the mutation exercised and hand it to the
  reviewer (§2, step 5).
- **No complacent or vacuous tests.** A test that passes regardless of the fix is
  forbidden. Pin the exact behaviour — assert the specific outcome, not merely
  "it failed".
- **Extend the existing test harnesses and fixtures** rather than writing ad-hoc
  one-offs.
- **Coverage ratchet** — gate a coverage floor in CI on the logic that shapes
  persisted state. Raise the floor after coverage improves; NEVER lower it by
  hand.

## 5. State-safety doctrine (the invariant reviews enforce)

Applies to any system that persists, synchronizes, derives or deletes state
(reconciliation engines, infrastructure tooling, sync jobs, caches, ledgers, …):

- **Never-drop / never-orphan, SYMMETRIC across read AND delete.** A destructive
  decision — removing a tracked record from the persisted representation, or
  assuming an external object is gone — is taken ONLY on **strict positive
  evidence**, never on an ambiguous or unconfirmed signal (e.g. an access-denied
  answer mapped to "absent", or a partial/paginated listing that cannot prove an
  absence). When in doubt, **fail closed** with an actionable diagnostic; never
  silently succeed leaving tracked fields stale, and never auto-remove a record.
- **Intent comes from the declared input compared to the external authority** —
  not from defaults or zero values the framework cannot distinguish from an unset
  field.
- **Keep the destructive decision out of the mapping/serialization layers** —
  the code that shapes the persisted representation must not be able to remove a
  record; structure it so an accidental drop is impossible there.
- **For a destructive absence decision, require authoritative live evidence** —
  documentation or a spec alone is never proof of absence.

## 6. Human gates and irreversibility

- **Every merge to a shared/integration branch and every operation that touches a
  live or production system requires an explicit human GO.** A GO in one context
  never extends to the next (per-action, per-session).
- **Merges to the default/release branch, release tags and published releases
  are performed by a human by default.** The agent may execute them ONLY under an
  **explicit, per-action human authorization that names the exact target** (the
  specific PR, tag, commit/SHA or release) — never presumed, never generalized
  from a previous one, never inferred from a broad phrasing. The authorization
  changes only WHO executes:
  every approval gate still applies in full (mandatory human review, the required
  approval markers, the validation step, CI) regardless of who performs the step.
  See `WORKFLOW_GIT_EPIC.md` for the normative gates.
- **No heavy load loops against shared or production systems.** Use bounded,
  qualitative probes with a circuit breaker, and stop at the first sign of
  distress.

## 7. Commit & changelog hygiene

- Commit messages are factual; the precise subject format and the issue-closure
  keywords are defined in `WORKFLOW_GIT.md`.
- **Forbidden anywhere:** automated authorship trailers/footers added by the
  tooling (e.g. an agent `Co-Authored-By:` line, or a "Generated with …" footer).
- A changelog entry is mandatory for every user-facing change.
