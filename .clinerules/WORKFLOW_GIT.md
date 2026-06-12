## Git Workflow (MANDATORY)

**Rule:** All merges into `main` happen **exclusively on GitHub through Pull
Requests**. No local merge into `main`.

- Work happens on a dedicated branch (`phaseX/y-issue`).
- Integration into `main` = through a PR merged on GitHub.
- Local `main` is only used for `git pull --ff-only` after a GitHub merge.
- Before opening a PR: `git fetch origin && git rebase origin/main`
  **on the feature branch** to synchronize it.

**Why:** a local merge into `main` while a PR evolves on GitHub creates
divergence and corrupts the shared history.

### Commands forbidden by default

- `git merge` (or equivalent) on local `main` — except `git pull --ff-only`
  after a GitHub merge.
- `git push --force` (or `--force-with-lease`) on `main`.
- `git commit` directly on local `main`.

### Release Candidate flow for multi-client repositories

For a repository whose releases can impact several clients, the RC flow
details must be carried by the repository-specific or EPIC-specific rules.
For this provider, the normative rule lives in `WORKFLOW_GIT_EPIC.md`.

Generic guardrails:
- no direct merge of automation commits into `main`;
- no tag, GitHub release or publication from an RC branch;
- final merge into `main` on GitHub only, by human review;
- a mandatory RC validation step between the RC branch and `main`.

### Nominal cycle

```bash
# 0. Start the issue: self-assign and set the Project status to "In Progress"
gh issue edit <N> --add-assignee "@me"
# Then change the GitHub Projects "Status" field through the GitHub API.
# Forbidden: gh issue edit <N> --add-label "status:in-progress"

# 1. Start from a clean main
git checkout main && git pull --ff-only
# For a multi-client RC flow, start from the target RC branch, not main.

# 2. Create the feature branch
git checkout -b phaseX/y-issue-slug

# 3. Work, commit atomically
git add ... && git commit -m "..."

# 4. Before PR: rebase on up-to-date main
git fetch origin && git rebase origin/main
# For a multi-client RC flow, rebase on origin/<rc-branch>, not origin/main.

# 5. Push + open the PR with gh
git push -u origin phaseX/y-issue-slug
gh pr create --base main --title "..." --body "..."
# For a multi-client RC flow, replace `main` with the target RC branch,
# for example: gh pr create --base rc/vX.Y.Z --title "..." --body "..."
# In that case, use `Refs #<N>` in the feature PR; the `Closes #<N>`
# keywords are reserved for the RC -> main PR.
# Outside the RC flow: the body MUST contain a "Closes #<N>" line at the top
# (see the "PR <-> Issue link" section below).

# 6. After GitHub merge: clean up locally
git checkout main && git pull --ff-only
# For a multi-client RC flow, switch back to the target RC branch and refresh
# it with `git pull --ff-only`, not main.
git branch -d phaseX/y-issue-slug
```

### PR <-> Issue link (MANDATORY)

**Rule:** any PR that resolves an issue MUST contain a GitHub closing keyword
(`Closes`, `Fixes`, `Resolves`) followed by the issue number, **in the PR
body**, ideally on the first line.

```text
Closes #<N>
```

Multi-client RC exception: for a feature PR targeting an RC branch, do not use
a closing keyword in the PR body or in commit messages. Use `Refs #<N>`. The
reliable `Closes #<N>` keywords are carried by the RC -> `main` PR, per the
repository-specific or EPIC-specific rule.

**Why:**

- Only a keyword in the **body** of a PR (or in a commit message merged into
  the default branch) triggers the automatic closing of the issue by GitHub.
  A `Closes #N` in the PR **title** is not enough — GitHub does not parse
  closing keywords there.
- It populates the issue's `closedByPullRequestsReferences` API field, which
  propagates the link to other issues' "Development" views, to notifications,
  and to release-notes tooling.
- It avoids forgetting to manually close the issue after merge.

**Closing keywords accepted by GitHub** (case-insensitive):
`close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`,
`resolved`. Prefer `Closes #<N>` for consistency.

**For a PR that references an issue without closing it** (dependency,
context, partial work): use `Refs #<N>` or `Related to #<N>`, which create a
soft link without automatic closing.

**Post-creation check:**

```bash
gh issue view <N> --json closedByPullRequestsReferences
# The field must contain the created PR. If it is empty, the keyword is
# missing or malformed — fix it with `gh pr edit <PR#> --body "..."`.
```

Multi-client RC exception: do not apply this check to a feature PR targeting an
RC branch, since it references the issue with `Refs #N`. The
`closedByPullRequestsReferences` check is done on the RC -> `main` PR after
aggregating the `Closes #N`.

## GitHub Issues Workflow (MANDATORY)

**Rule:** each issue follows an explicit lifecycle on GitHub, and the
exchanges are **channeled** between the issue and its PR to preserve the
readability of both histories.

### When starting work on an issue

1. **Self-assign the issue** with the configured gh account:

   ```bash
   gh issue edit <N> --add-assignee "@me"
   ```

2. **Set the Project status to In Progress through the GitHub API:**

   - Never use a `status:in-progress` label to represent progress status. A
     label is not the issue status.
   - Change the GitHub Projects `Status` field of the item linked to the issue
     to the `In Progress` option.
   - Use the GitHub Projects v2 API, for example through `gh api graphql`,
     after resolving `PROJECT_ID`, `PROJECT_ITEM_ID`, `STATUS_FIELD_ID` and
     `IN_PROGRESS_OPTION_ID` for the relevant project.

   ```bash
   gh api graphql -f query='
   mutation(
     $projectId: ID!
     $itemId: ID!
     $statusFieldId: ID!
     $inProgressOptionId: String!
   ) {
     updateProjectV2ItemFieldValue(input: {
       projectId: $projectId
       itemId: $itemId
       fieldId: $statusFieldId
       value: { singleSelectOptionId: $inProgressOptionId }
     }) {
       projectV2Item { id }
     }
   }' \
     -F projectId="$PROJECT_ID" \
     -F itemId="$PROJECT_ITEM_ID" \
     -F statusFieldId="$STATUS_FIELD_ID" \
     -F inProgressOptionId="$IN_PROGRESS_OPTION_ID"
   ```

   If the issue is not yet in the project, add it first through the GitHub
   Projects v2 API (`addProjectV2ItemById`), then apply the status mutation
   above.

### During implementation (before opening the PR)

All exchanges related to the **design** of the solution stay in the issue.
Post there:

- the technical decisions made and their rationale;
- the implementation trade-offs arbitrated;
- the clarifications requested and the answers obtained;
- the divergences identified from the initial plan.

```bash
gh issue comment <N> --body "..."
```

### After opening the PR

As soon as a PR is opened to resolve the issue, **code review discussions**
move into the PR. The issue stops being the exchange channel:

- reviewer comments (findings, change requests, refactor suggestions) go
  **into the PR**, not into the issue;
- the implementer's replies (applied fixes, justifications, counter-arguments
  on a remark) also go **into the PR**, not into the issue;
- the issue only receives high-level synthetic updates when necessary (major
  blocker, scope change).

```bash
# General comment in the PR
gh pr comment <PR#> --body "..."

# Formal review (approve / request-changes / comment)
gh pr review <PR#> --comment --body "..."
```

### PR verification / review (MANDATORY)

When the user asks to **verify**, **re-read**, **review**, **check** or
**validate** a GitHub PR, treat the request as a PR review, not as a simple
local analysis.

**Rule:** every review conclusion must be published on GitHub in the PR before
the final answer, unless the user explicitly says otherwise (`local only`,
`do not post`, draft, etc.).

Mandatory checklist before answering:

1. Inspect the PR (`gh pr view`, `gh pr diff`, CI checks, issue link if
   applicable) and run the relevant local tests.
2. Formulate findings with severity, files/lines, impact, and the expected
   fix.
3. Publish the review in the PR:
   - blocking finding: first try `gh pr review <PR#>
     --request-changes --body "..."`;
   - if a blocking review must go through a comment, use a canonical marker
     such as `Verdict: REQUEST-CHANGES (comment)`; the absence of the marker
     does not neutralize a comment whose content requires a fix before merge;
   - non-blocking findings or informative review: `gh pr review <PR#>
     --comment --body "..."`;
   - no finding: post at least a comment/review stating the checks performed
     and the residual risk.
4. If GitHub refuses the formal review (e.g. the `gh` account authored the PR
   and cannot `request-changes`), switch immediately to
   `gh pr comment <PR#> --body "..."` with the same content.
5. The final answer must contain the link to the published GitHub
   comment/review, the checks performed, and any limitations.

### Project impact of a review

The normative Project status rule is defined in `WORKFLOW_GIT_EPIC.md`. Do not
redefine it here. After publishing the GitHub review, apply that rule to the
Project.

The reviewer model is responsible for aligning the Project right after the
review is published. If it lacks Project rights, it must report it to the
pilot model in a GitHub comment on the PR, with a `Project-move-request:`
prefix, the relevant item and the expected status. In that case, the pilot
model is the terminal owner of the Project move; the piloting task is only
done once the Project move is done.

The pilot model must verify Project alignment after any published review, even
if the reviewer did not report a gap. This verification is mandatory before
answering that the review or the lot is done.

When the fix is pushed, the return from `In Progress` to `Review` requires an
explicit signal defined in `WORKFLOW_GIT_EPIC.md`. A simple push is not
enough.

A stage review is a review done at the end of a lot, before moving to
`Review`, before merge, or after a blocking finding. Each stage review must be
launched explicitly in these cases.

Minimal checks of a stage review:
- the tracked PR exists in the Project if `WORKFLOW_GIT_EPIC.md` requires it;
- GitHub comments and reviews since the last Project move have been re-read,
  including those without a canonical marker;
- any comment whose content requires a fix before merge is treated as a
  blocking finding;
- the Project status of the PR reflects the latest published review;
- the source issue stays independent from the PR status;
- the main EPIC stays in `Plan`;
- the `Plan` column does not contain the backlog.

**Guardrail:** an answer only in the chat after a PR review request is
incomplete. Never end the task without a GitHub trace, unless explicitly asked
not to post.

**Why:** an issue captures the *problem* and the decision to tackle it; a PR
captures the *execution* and the code review. Mixing the two dilutes both
histories and complicates later re-reading (audit, post-mortem, new
contributor).
