# Git Workflow Epic / GitHub Project

These rules complement `WORKFLOW_GIT.md` for piloting a GitHub EPIC tracked in
a GitHub Project v2.

## Principle

The Project is used to pilot execution. It must not become an unreadable audit
matrix.

- `Status` describes only the workflow state.
- `Lot` describes the workstream.
- `Risk` describes the technical nature of the risk.
- `Priority` describes the processing order.
- `TF prod breaker` is a private, binary operational indicator.

Do not use `Status` to classify a topic by theme, severity, lot or risk. That
information has its dedicated fields.

## Piloting scope

The provider tracking may temporarily live in a broader program board until a
dedicated provider board is stabilized. This is an exception, not the target
organization.

Rules:
- keep it explicit that the provider is a blocking workstream temporarily
  tracked from a broader program board;
- create a dedicated provider project once the setup is stabilized;
- do not mix, in the same EPIC, provider bugs, broader-program tasks and
  operational work without an explicit classification field;
- keep the piloting Project private if the tracking contains internal
  priorities, risks or trade-offs.

## Language of provider artifacts

The Cloud Temple Terraform provider is a multi-client component. Any public
artifact, or any artifact likely to be read by clients, must be written in
English.

Rules:
- public issues of the provider repo: English;
- PR titles, bodies and comments: English;
- GitHub reviews published on provider PRs: English;
- commit messages, release notes, changelogs and documentation: English;
- rules copied into the provider repo: English.

Internal piloting in the private Project may stay in French, as long as it is
not published as-is into the provider repo. Any information copied from the
private Project into a public issue or PR must be reworded into factual
English, without anxious internal signals.

## Adoption in the provider repo

The provider repo must eventually carry its own workflow rules.

Rules:
- create, in the provider repo, a `RULES/` directory with a `WORKFLOW_GIT.md`
  adapted to the provider, written in English;
- do this adoption through issue + branch + PR, never by directly modifying
  `main`;
- include a review by the reviewer model in this step;
- do not block critical provider fixes only because this rule transfer is not
  done yet.

## Provider Terraform RC flow

The provider must use a Release Candidate flow for automation commits with no
immediate client impact.

Rules:
- use a dedicated RC branch, in the `rc/vX.Y.Z` format, as the target of the
  delivery train's automation PRs;
- treat `rc/vX.Y.Z` as an RC integration branch, not as a tag;
- do not use a branch name like `vX.Y.Z-rc`, too close to a SemVer
  pre-release tag;
- do not open automation PRs directly against `main` when they belong to the
  RC train;
- do not create a tag, GitHub release or provider publication from the RC
  branch;
- use `Refs #N` in feature PRs targeting the RC branch; reserve the `Closes #N`
  keywords for the RC -> `main` PR, except for a direct hotfix to `main`
  explicitly approved;
- forbid GitHub closing keywords (`Closes`, `Fixes`, `Resolves`) in commit
  messages of feature PRs targeting the RC branch;
- then open an RC -> `main` PR on GitHub;
- add an intermediate RC validation step between the last merge into the RC
  branch and the merge into `main`;
- the RC -> `main` PR carries the English release notes, the upgrade risks,
  and the `Closes #N` keywords of the included issues;
- the RC -> `main` merge is a human decision: a human reviewer must validate
  and merge on GitHub, never an automation agent;
- `main` must be the repo's default branch; the RC branch must never become
  the default branch;
- `main` must be protected: human review required, merge/push rights reserved
  to authorized humans, and no automation token may merge the RC -> `main` PR;
- the provider repo must disable `Rebase and merge`, or make it unavailable
  for the RC -> `main` PR through an equivalent protection;
- release tags `v*` must be protected: creation reserved to authorized humans,
  signed tag when the repo requires it, and the tagged commit present in
  `main`;
- the provider release workflow must refuse any publication if the targeted
  commit is not reachable from `main`.

The RC -> `main` PR is the reliable source for closing issues. Do not rely on
the bodies of feature -> RC PRs, nor on commit messages, to close GitHub
issues. For any allowed merge mode, closing must be carried by the body of the
RC -> `main` PR.

For the RC -> `main` PR, `Rebase and merge` is forbidden. Use `Squash and
merge` or `Create a merge commit`, with the aggregated `Closes #N` in the body
of the RC -> `main` PR. The provider repo must disable or forbid rebase merge
for this PR.

The RC validation step must be materialized in the RC -> `main` PR by a GitHub
comment containing the canonical marker `RC-VALIDATION: OK <branch>
<commit-sha>`, the list of executed checks, the list of aggregated issues, and
the upgrade risks. Without this comment, the RC -> `main` PR cannot be merged.

If the RC branch receives a new commit after the `RC-VALIDATION: OK <branch>
<commit-sha>` marker is published, that marker is invalidated and the RC
validation must be replayed on the new SHA.

Minimal RC validation before merge into `main`:
- full CI green;
- relevant Terraform checks re-run on the RC state;
- explicit verification of `TF prod breaker` risks;
- release notes and upgrade notes in English;
- aggregated and deduplicated list of included issues, with `Closes #N` in the
  body of the RC -> `main` PR for each issue to close;
- the issue-aggregation script output attached or copied into the RC -> `main`
  PR, and reviewed by the human reviewer;
- confirmation that no still-open fix must go back to the RC.

Issue-aggregation procedure before the RC -> `main` PR:
1. run a dedicated aggregation script in the provider repo, or an equivalent
   release script versioned with the repo;
2. list all feature PRs merged into the RC branch since its creation;
3. collect the `Refs #N`, `Closes #N`, `Fixes #N` and `Related to #N` present
   in their bodies;
4. deduplicate the issues;
5. verify that each issue is in the RC train scope;
6. report the intended closes in the body of the RC -> `main` PR with
   `Closes #N`;
7. verify that the `Closes #N` list in the RC -> `main` body matches exactly
   the script output;
8. after the RC -> `main` merge, verify each issue on GitHub and immediately
   fix any phantom issue that stayed open.

The human reviewer of the RC -> `main` PR must verify that the aggregation
script output and the PR body match. If an issue is missing or an out-of-scope
issue is listed, the RC -> `main` PR cannot be merged.

After the RC -> `main` merge, delete the RC branch and never reuse the same
branch name for another delivery train.

Any provider release tag is created exclusively on a commit present in `main`
after the RC -> `main` merge.

### Direct hotfix to `main`

A direct hotfix to `main` is a strict exception to the RC flow.

Conditions:
- security incident, blocking client regression, or `TF prod breaker` `RED`
  risk validated as urgent;
- public or private tracking issue, depending on sensitivity;
- PR in English directly to `main`;
- GitHub comment `HOTFIX-APPROVED: <issue>` posted by an authorized human
  before merge;
- mandatory human review;
- merge on GitHub by a human, never by an automation agent.

After the hotfix merge:
- create the tag or release only from the hotfix commit present in `main`, if
  an immediate release is required;
- port the hotfix into any active RC branch through a dedicated PR;
- replay the RC validation on the new RC branch SHA;
- update the RC release notes in English.

## Project statuses

### Blocking finding

A finding is blocking when its content implies that a PR must not be merged
without a fix.

Blocking examples:
- risk of breaking or changing an existing client Terraform state;
- uncontrolled provider/API drift;
- functional regression or violated Terraform invariant;
- unrequested or destructive API patch;
- secret, sensitive data or transient state persisted in the state;
- a required test or check broken by the PR;
- incomplete issue/PR mapping when it prevents risk traceability.

Non-blocking examples:
- renaming, style, readability or refactor without behavioral impact;
- future improvement;
- desirable but not merge-required test or documentation;
- informative remark about an already-accepted residual risk.

Content prevails over format. An unmarked comment is still blocking if its
content requires a fix before merge. The canonical marker
`Verdict: REQUEST-CHANGES (comment)` serves readability; it is not the
condition that creates the blocking.

### Canonical markers

The following markers are canonical for GitHub comments related to piloting:
- `Verdict: REQUEST-CHANGES (comment)`: blocking review published as a
  comment;
- `READY FOR RE-REVIEW`: fix pushed and ready to be re-reviewed;
- `Project-move-request: <item> -> <status>`: Project move request when the
  reviewer lacks the rights;
- `Awaiting independent reviewer`: PR in `Review`, awaiting validation by
  another model or a human;
- `RC-VALIDATION: OK <branch> <commit-sha>`: RC validation completed on the
  indicated branch and commit.

### `Plan`

Strict and limited use.

Put in `Plan` only:
- the main EPIC;
- an active scoping item;
- a governance decision under definition.

The main EPIC stays in `Plan` while its child issues or PRs move to `Todo`,
`In Progress`, `Review`, `Blocked` or `Done`. Do not move the EPIC by
contagion from a child item.

For the Terraform Provider Cloud Temple audit, the public EPIC #257 is also an
item of the private piloting Project. It stays the scoping item in `Plan`;
operational progress is carried by child issues, PRs and drafts. Detailed risk
flags, including `TF prod breaker`, are piloted at the child level when they
concern a specific fix or proof.

Do not put in `Plan`:
- a known bug not started;
- an issue without a PR;
- a backlog task;
- a private draft that is not the active scoping.

A Project whose `Plan` column contains the backlog is considered badly piloted.

### `Todo`

Put in `Todo` any identified but not-started work:
- open bug without an active PR;
- issue to handle later;
- inactive governance task;
- private requalification item that is not in progress.

### `Draft`

Put in `Draft` only:
- a GitHub draft PR that is not in active correction;
- a deliverable being written that already exists as a concrete draft.

Do not use `Draft` to mean "to think about".
If a draft PR is actively worked on, or if a review requested a blocking fix,
its Project status is `In Progress`.
A draft PR is considered active if it has an open blocking fix, an assignee
acting on it, an explicit team signal, or recent work commits. Otherwise it
stays `Draft` and must be requalified if it carries a `TF prod breaker` risk.

### `In Progress`

Put in `In Progress` when work has actually started:
- assigned and picked-up issue;
- active work branch;
- PR, including draft, in active correction or execution;
- execution in progress on the team side.

An issue can stay `In Progress` while its PR is in `Review`.
A PR returns to `In Progress` as soon as a review publishes a blocking
finding, a mandatory fix before merge, or an equivalent `request-changes`.
This return applies to any blocking review published as a comment, whatever
the reason for the comment. The case where GitHub refuses the formal status
because the reviewer also authored the PR is only an example.

The equivalent comment should use the canonical marker
`Verdict: REQUEST-CHANGES (comment)` when possible. The absence of the marker
does not neutralize a blocking finding.

Do not use `Blocked` for a fix requested by review: as long as the team can
push a fix, the correct state is `In Progress`.

### `Review`

Reserve `Review` for PRs ready to re-read, or a deliverable whose only
remaining activity is review.

An item stays in `Review` only if the feedback is non-blocking, or if the
reviewer concludes the PR can move forward without a mandatory fix.

An item can stay in `Review` while awaiting a mandatory independent
validation. In that case, post a GitHub comment with the marker
`Awaiting independent reviewer`. Do not use `Blocked` for this wait.

After fixing a blocking finding, the item returns to `Review` only when the
pilot model, the PR author, or the team explicitly indicates the PR is ready
for re-review. A simple push is not enough. Accepted signals:
- `READY FOR RE-REVIEW` comment;
- GitHub re-review request;
- explicit comment resolving each blocking finding.

If the PR author is also the pilot model, this signal only authorizes the
return to `Review`. It is not a validation: the reviewer model must still
re-read or explicitly acknowledge.

If the reviewer is also the GitHub author of the PR, or if several models act
through the same GitHub account, validation relies on the separation of model
roles. The re-review comment must indicate which model holds the reviewer
role. If the same model fixed and reviewed, a validation by another model or a
human is mandatory before merge.

An issue's status never automatically follows its PR's status. The issue
carries the problem; the PR carries the execution. By default, do not move an
issue to `Review` only because its PR is in review.

A source issue linked to a PR with a blocking finding stays `In Progress` by
its own logic, because the problem it carries is not solved, even if the PR
status evolves separately. If the issue was moved out of `In Progress` by
mistake, fix the status. The issue moves to `Done` when the problem is really
closed per the normal GitHub workflow, usually after merge or explicit
closing.

### `Blocked`

Put in `Blocked` when the item cannot move forward without an external event:
- PR stacked on another PR;
- API/support decision awaited;
- blocking upstream bug;
- external quota/tooling temporarily blocking.

The blocking must be explicit in the issue, the PR, or the governance item.

### `Done`

Put in `Done` when the tracked object is really finished:
- closed issue;
- merged PR or closed without follow-up;
- executed governance task.

After a blocking finding, do not shortcut `In Progress` -> `Done`. The PR must
have a trace of positive re-review or independent validation before merge.
After merge, the PR item moves to `Done`. Do not leave a merged PR item in
`Review`.

## Issues and PRs

By default, an issue and its PR can both be in the Project if it helps
traceability.

For any PR tracked by the EPIC, requested for review, or attached to a
`TF prod breaker` risk, the PR item is mandatory in the Project. If the PR is
missing from the Project, add it before applying the expected status.

A PR is considered tracked by the EPIC if it meets at least one of these
criteria:
- its body contains `Closes #N`, `Fixes #N` or `Refs #N` toward an EPIC child
  issue;
- it is explicitly listed in the EPIC scope, plan, or a comment;
- it implements a decision or proof attached to an EPIC lot;
- it was requested for review within the EPIC.

Rules:
- the issue describes the problem, need, risk or scope;
- the PR describes the technical execution;
- the PR carries over `Lot`, `Risk`, `Priority` and `TF prod breaker` from the
  issue it implements;
- the `Status` of the issue and of the PR can diverge;
- a PR that closes an issue must contain `Closes #N` in its body, as defined
  in `WORKFLOW_GIT.md`.

## Piloting fields

### `Lot`

`Lot` classifies the workstream. It is never a status.

Lots used for the Cloud Temple Terraform provider:
- `S0 Stabilisation`
- `S1 PR train`
- `S2 VPC`
- `S3 Proofs E1-E6`
- `S4 Missing issues`
- `S5 State-safety`
- `S6 Agentic adoption`

### `Risk`

`Risk` describes the technical nature of the problem, not its urgency.

Common values:
- `none`
- `drift`
- `scope-gap`
- `schema-mismatch`
- `transient-retry`
- `destructive`
- `state-upgrade`
- `state-secret`
- `unknown`

Examples:
- permanent Terraform drift: `drift`
- provider schema incompatible with API response: `schema-mismatch`
- missing retry on a transient error: `transient-retry`
- provider action that can cause a dangerous change: `destructive`
- secret stored in the state: `state-secret`

Do not use `state-upgrade` unless the topic explicitly concerns a Terraform
`SchemaVersion` or `StateUpgrader`.

### `Priority`

`Priority` describes the processing order.

- `P0`: must be handled before considering the EPIC safe.
- `P1`: important, but can follow the P0s.
- `P2`: useful improvement or structuring debt.
- `P3`: low-urgency backlog.

### `TF prod breaker`

Private Project field. Do not make it a public label.

Values:
- `RED - peut casser le TF client`
- `NO - pas identifie`

`RED` means: if the case affects a production client, without any change to
their Terraform code, they may see a `plan`, `apply`, `refresh` fail, diverge
permanently, change an unintended resource, or make the state non-convergent.

`NO` means: no direct risk of operational breakage of a production client's
Terraform is identified at this stage.

This field does not replace `Risk`.

Examples:
- datasource that systematically fails: `RED`
- permanent drift that prevents convergence: `RED`
- import that creates a non-convergent state: `RED`
- secret in the state: `NO` for `TF prod breaker`, but `Risk=state-secret`
- missing feature without regression of an existing usage: `NO`
- aggregated EPIC: `NO`, the children carry the flag

## Owner

The Project `Owner` field represents piloting, not necessarily the native
GitHub assignment.

Rules:
- use the native GitHub assignment for the person who implements;
- use `Owner` for the group responsible for piloting;
- do not deliberately contradict an explicit GitHub assignee;
- if a divergence exists, document it.

## Public / private

The provider repo and the swagger may be public. The piloting Project may stay
private.

Rules:
- do not publish anxious labels or comments if the signal is an internal
  piloting tool;
- keep internal flags in the private Project;
- public issues must stay factual, reproducible, and useful to implementation;
- consider the public swagger usable for audit and review;
- do not publish, in public issues, the internal `TF prod breaker`
  classifications, arbitration priorities or private piloting comments.

## Mandatory reviews

Each important EPIC step must plan a separation between a pilot model and a
reviewer model.

The roles are not tied to a specific tool:
- one model can pilot while another reviews;
- the roles can be swapped from one step to the next;
- another model can hold one of the two roles if the context allows.

Rules:
- do not have the same model carry both the piloting decision and its review
  when a second model is available;
- explicitly note which model pilots and which reviews;
- the review must challenge the mapping, statuses, risks, priorities and
  Project impacts;
- any GitHub PR requested for review must receive a review or a GitHub
  comment, per `WORKFLOW_GIT.md`;
- if the external reviewer cannot be called for confidentiality reasons, or if
  the tooling refuses the call even after user authorization, do not bypass;
- in that case, do a local review by the available model, note the exception,
  and continue only if the decision stays reversible or explicitly validated.

## Project modification discipline

Before modifying the Project:
1. Read the real GitHub state.
2. Produce an explicit mapping.
3. Have the reviewer model review, or document the authorized local exception
   if the external reviewer is unavailable.
4. Apply through an idempotent script.
5. Verify the result by an independent GitHub read.
6. Update memory.

After each lot, verify that the `Plan` column does not contain the backlog.

## GitHub Project execution discipline

GitHub Project modifications consume the GraphQL quota and can fail mid-lot.

Rules:
- check the GraphQL quota before Project mutations;
- do not launch a series of mutations if the remaining quota is too low;
- if a mutation fails mid-lot, read the exact state before retrying;
- resume only the missing or incomplete items;
- do not conclude a lot on the sole output of a write script;
- verify by an independent read, targeted if needed to save quota;
- avoid partial updates that leave an inconsistent Project view.

## `Plan` column hygiene

After any Project operation, check the `Plan` column.

Rules:
- the EPIC may stay in `Plan`;
- truly active scoping items may stay in `Plan`;
- not-started bugs and tasks must be in `Todo`;
- inactive private drafts must be in `Todo`;
- if more than a few non-scoping items appear in `Plan`, fix it before adding
  a new lot.
