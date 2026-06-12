# Cline's Memory Bank — Live Memory + Graph Memory (Advanced Template)

> **Audience** : workspaces connected to **both** Live Memory and Graph Memory MCP servers.
> **For Live-Memory-only workspaces**, use [`WORKSPACE_CLINE_RULES.md`](WORKSPACE_CLINE_RULES.md) instead.

My memory resets completely between sessions. I use the **Memory Bank** as the mandatory session bootstrap, **Graph Memory** for durable semantic recall across documents, and **repository files** as the canonical, detailed source of truth.

## 🧭 Responsibility separation (read this first)

| Layer                  | Role                                                                                 | Authority                          |
| ---------------------- | ------------------------------------------------------------------------------------ | ---------------------------------- |
| **Memory Bank** (Live) | Compact session bootstrap — current focus, open decisions, recent milestones         | Mutable, agent-curated via notes   |
| **Graph Memory**       | Durable semantic index, cross-document relations, historical/incident locator        | Authoritative *index*, not content |
| **Repository files**   | Canonical source for RFCs, incidents, runbooks, scripts, command transcripts, code   | Final authority                    |

Two non-negotiable principles:

- **Graph Memory complements the bank; it does not replace it.**
- **Graph Memory localizes; canonical repository files confirm.**

Consequences:

- The **Live Memory consolidator** only updates the Memory Bank. It never ingests into Graph Memory, never mirrors Graph Memory content, and never receives ingestion instructions.
- **Graph Memory ingestion is an agent/tooling responsibility**, driven from canonical repository documents — never from the Memory Bank.
- Entity classification / ontology is a **Graph Memory server-side concern**. These agent rules must not declare any ontology, entity type, or classification rule.

## 🔌 Configuration

My persistent memory is managed by the **Live Memory** MCP server.
Long-lived semantic recall is managed by a **Graph Memory** MCP server.

> **⚙️ Values to customize per project:**
>
> - **LIVE_MCP_SERVER** = `live-mem`
> - **SPACE** = `terraform-provider`
> - **GRAPH_MCP_SERVER** = `graph-mem`
> - **GRAPH_MEMORY_ID** = `TERRAFORM-PROVIDER`
>
> All instructions below use `{LIVE_MCP_SERVER}`, `{SPACE}`, `{GRAPH_MCP_SERVER}`, and `{GRAPH_MEMORY_ID}` — I automatically substitute them with the values above.
> The agent name is **auto-detected** from the authentication token (no configuration needed).
>
> The Live Memory tools (`space_*`, `live_*`, `bank_*`, `graph_*`) are exposed by the `{LIVE_MCP_SERVER}` MCP server. The Graph Memory tools (e.g. `memory_search`) are exposed by `{GRAPH_MCP_SERVER}`. When two MCP servers expose tools with the same name, always disambiguate by targeting `{LIVE_MCP_SERVER}` for Live Memory operations.
>
> ⚠️ **Never put tokens, endpoints, or sensitive server parameters in these rules.** They live in MCP client configuration / vault / env.

> ⚠️ **Namespace discipline for this workspace:** use the standard MCP servers `live-mem` and `graph-mem` only. Do not use personal `my-live-mem` or `my-graph-mem` namespaces for this provider workspace.

## 📖 At Session Start (MANDATORY)

At the start of a new session, after a context reset, or after a consolidation:

1. Call `space_rules("{SPACE}")` to read the rules (bank structure, sub-files index)
2. Call `bank_read_all("{SPACE}")` to load **all** consolidated context in a single call
3. Call `live_read(space_id="{SPACE}")` to read **unconsolidated notes**
4. Read the content carefully before starting
5. Identify the current focus in `activeContext.md`

> ⚠️ NEVER start working without reading the bank first.
>
> ⚠️ **Do NOT read sub-files individually** at startup. `bank_read_all` returns the entire bank including sub-directories in one call. Reading files individually wastes context window tokens and API calls. Only use `bank_read` for a specific file if you need to re-read it after a consolidation.
>
> 💡 **Why read live notes?** Between sessions, notes may have been written (by me or other agents) without being consolidated into the bank. These notes contain recent context that does not yet appear in bank files. Ignoring them = risking redoing work already done or missing recent decisions.

## 🏗️ Bank Structure

The exact bank structure is defined by the **rules of `{SPACE}`** (loaded via `space_rules`). Do not assume a fixed layout in these agent rules — the consolidator follows the space rules, not this document.

Typical hierarchy (illustrative only):

```
projectbrief.md      ← Foundation (rarely modified)
productContext.md    ← Why, how, positioning
systemPatterns.md    ← Architecture (may index sub-files)
techContext.md       ← Tech stack (may index sub-files)
activeContext.md     ← CURRENT FOCUS (read this first after bank_read_all)
progress.md          ← Recent milestones + backlog
```

If the bank uses sub-directories (e.g. `systemPatterns/*.md`), the index files list what each sub-file covers. Re-read a specific sub-file only when working on that component.

## 🧹 Bank hygiene & compaction discipline

The Memory Bank must stay compact. It is a **session bootstrap**, not a long-term archive.

### Size targets (defaults, adjust in `space_rules` if needed)

- `activeContext.md`: target **8–12 KB**, hard warning above **15 KB**
- `progress.md`: target **20–30 KB**
- Total bank loaded by `bank_read_all`: target **40–60 KB**

When these limits are exceeded, consolidation must **move or summarize** information instead of appending more text. If `bank_read_all` exceeds the total target, **do not add more historical detail to the bank**. Prefer one of:

- move durable details to canonical repository documents and keep only short pointers in the bank;
- route stable cross-document history to Graph Memory ingestion (agent-side, see below);
- request or perform an explicit compaction pass before adding new long-lived context.

### `activeContext.md` scope

`activeContext.md` must contain only:

- Current focus
- Open decisions
- Active risks / blockers
- Next concrete actions
- Very short pointers to recent completed work, only when needed to understand the current focus

`activeContext.md` must NOT contain:

- Closed incidents (except a one-line pointer if a follow-up action remains)
- Long deployment logs
- Detailed RFC history
- Completed implementation details
- Repeated summaries already present in `progress.md` or in sub-files
- Long Graph Memory query results

### `progress.md` scope

`progress.md` is a **bounded recent journal**, not a full archive.

- Keep concise dated milestones.
- Keep only the last relevant operational period.
- Move durable patterns and lessons to the appropriate sub-files defined by `space_rules`.
- Replace long closed-event narratives with short pointers to canonical repository files.

### Consolidation discipline (notes the consolidator will read)

When writing notes for consolidation:

1. Prefer pointing to the specific sub-file over appending to `activeContext.md`.
2. If a note describes completed work, summarize it for `progress.md` and remove it from `activeContext.md` unless it creates a new active follow-up.
3. If a note describes a durable technical lesson, route it to the relevant sub-file defined by the space rules.
4. If the same fact appears in multiple places, keep the most canonical location and replace other copies with short pointers.
5. Never duplicate large incident, RFC, or runbook details already present in the repository — point to them.

## 🔭 Graph Memory routing

Graph Memory is available for `{SPACE}` via the `{GRAPH_MCP_SERVER}` MCP
server. The Live Memory space is not assumed to be graph-connected unless
`graph_connect` has been explicitly configured:

- Live Memory MCP server: `{LIVE_MCP_SERVER}` (hosts `{SPACE}`)
- Graph Memory MCP server: `{GRAPH_MCP_SERVER}`
- Graph Memory ID: `{GRAPH_MEMORY_ID}`
- Live↔Graph connection: optional; configure only with explicit user request
  and connection parameters

Purpose: durable semantic index for stable cross-document facts (incidents, change records, runbooks, billing rules, infrastructure components, operational decisions, prior RFCs, etc.). It accelerates *finding* the relevant canonical document. It does **not** replace it.

### Operational rules

1. Use the Graph Memory MCP server directly with `memory_id="{GRAPH_MEMORY_ID}"`
   for Graph operations. Use `graph_status(space_id="{SPACE}")` only if a
   Live↔Graph connection has been explicitly configured.
2. Use Graph Memory search/query when looking for historical or cross-document facts **instead of** expanding `activeContext.md` or `progress.md`.
3. **Do not run a full `graph_push` as a routine end-of-session action.** Ingestion is slow (LLM extraction + embeddings per document) and must remain an explicit, scoped operation.
4. **Do not call `graph_connect` or alter the Live Memory ↔ Graph Memory binding unless the user explicitly asks for a connection change.**
5. **Do not automatically ingest the bank after consolidation.** The bank is a moving compact summary; indexing it would index the bank's drift, not the canonical facts.
6. **Never push `activeContext.md` or `progress.md` to Graph Memory.** These files are volatile by design: `activeContext.md` is a session focus snapshot, `progress.md` is a bounded recent journal. Indexing them would teach Graph Memory transient or already-superseded summaries. They must remain Memory-Bank-only and never end up in a Graph ingestion call.
7. Prefer selective ingestion of stable, high-value canonical documents.
8. If a Graph query or ingestion times out, check the Graph Memory job or
   document status before retrying; partial ingestion may already have occurred.
9. Avoid duplicate document ingestion. Use the document's stable `source_path` as the deduplication key.

### Graph-first lookup policy

For historical context, prior incidents, RFC decisions, root causes, operational lessons, billing rules, infrastructure relationships, runbooks, or any cross-document fact:

- **Query Graph Memory first**, before widening the Memory Bank or scanning broad repository trees.
- Treat Graph Memory as a **semantic locator**: use it to find relevant entities, relations, and `source_path` values.
- **Before changing code, infrastructure, documentation, or an operational procedure, re-read the canonical repository file** referenced by Graph Memory.
- If Graph Memory returns stale, incomplete, or conflicting facts, **prefer the canonical repository document** and note the inconsistency (as a `live_note` issue/insight).
- **Do not copy Graph query results into `activeContext.md` or `progress.md`** — keep only a short pointer (entity name, document path, decision date) tied to an active decision or action.
- If Graph Memory is unavailable, fall back to a targeted repository search and keep the bank compact.

### Agent-side ingestion (NOT consolidator-side)

- Graph Memory ingestion is **my responsibility as the agent / tooling layer**, never the consolidator's.
- Regular Graph ingestion starts from **canonical repository documents**, not from the Memory Bank.
- Use the project's dedicated ingestion script or pipeline (if any) — declared in the project's `techContext.md` or operational docs, not here.
- A re-ingestion replaces the previous document for the same stable `source_path`. If the content hash is unchanged, do not re-ingest. If it changed, re-ingestion is acceptable and should overwrite the prior graph document.
- Prefer asynchronous / batched ingestion when supported, to avoid long synchronous client calls.
- **Never put bearer tokens, endpoints, or other secrets in commands, commits, logs, or these rules.** Use config/env/vault.

## 📝 During Work

Write atomic notes with `live_note` only for **durable facts, decisions, issues, completed milestones, or follow-up actions** that are not already captured in canonical repository files:

```
live_note(space_id="{SPACE}", category="<category>", content="...")
```

**Categories**:
- `observation` — Factual findings, command outputs
- `decision` — Technical choices and their justification
- `progress` — Advancement, what is completed
- `issue` — Problems encountered, bugs
- `todo` — Identified tasks to do
- `insight` — Learnings, patterns discovered
- `question` — Points to clarify, pending decisions

**Do not write live notes** for:

- Transient polling or routine command chatter
- Long verbatim Graph Memory query results (keep a short pointer instead)
- Facts already captured in a canonical repository document, unless they change the active plan

## 🧠 At Session End (or after a significant block of work)

Ask the user before running consolidation. Run it only after explicit validation.

```
bank_consolidate(space_id="{SPACE}")
```

The consolidator will update the bank files according to `space_rules("{SPACE}")`. **It will not push anything to Graph Memory** — that remains an explicit, agent-driven step started from canonical repository files, only when needed.

`bank_consolidate` is fire-and-forget: it returns an async job ack (`running` / `queued`) with `next_action="return_to_user_without_polling"`. **Call it once and return to the user.** `bank_consolidation_status(job_id)` exists for **explicit manual checks only**.

## ⚠️ Mandatory rules

1. **Do not write directly to the bank in normal operation** — only the LLM consolidation does that. Exception: explicit, user-approved maintenance with `manage` rights.
2. **Always pass `space_id="{SPACE}"`** in every call.
3. **Write atomic notes only for durable information not already canonicalized** — 1 note = 1 fact, 1 decision, or 1 task.
4. **At session end, write a short summary note and request/confirm consolidation** — consolidate only after user validation, then return without polling.
5. **Read the bank at startup** — never work without context.
6. **Use `bank_read_all` once** — never read sub-files individually at startup.
7. **Never ingest the bank into Graph Memory.** Ingest only canonical repository documents, and only when explicitly needed.
8. **Never run `graph_push` / `graph_connect` / binding changes in routine flows** — only on explicit user request.
9. **Never declare ontologies or entity classifications here** — that is a Graph Memory server-side concern.
10. **Never expose tokens, endpoints, or sensitive server parameters in these rules.**

## 🔄 When the user asks "update memory bank"

1. Write `live_note` notes summarizing the current state of work (short, atomic).
2. Call `bank_consolidate(space_id="{SPACE}")` and return without polling.
3. After the user confirms the consolidation completed, optionally verify with `bank_read_all("{SPACE}")`.
4. **Do not trigger any Graph Memory operation** as part of this routine.

## 📊 Useful Commands

| Action                          | Command                                                                                  |
| ------------------------------- | ---------------------------------------------------------------------------------------- |
| Read all bank context           | `bank_read_all("{SPACE}")`                                                               |
| Read the rules                  | `space_rules("{SPACE}")`                                                                 |
| Write a note                    | `live_note(space_id="{SPACE}", category="...", content="...")`                           |
| Consolidate                     | `bank_consolidate(space_id="{SPACE}")`                                                   |
| View recent notes               | `live_read(space_id="{SPACE}")`                                                          |
| View another agent's notes      | `live_read(space_id="{SPACE}", agent="other-agent")`                                     |
| Space info                      | `space_info("{SPACE}")`                                                                  |
| Graph status                    | `graph_status(space_id="{SPACE}")`                                                       |
| Graph semantic search           | `memory_search(memory_id="{GRAPH_MEMORY_ID}", query="...", limit=10)`                    |

## 🌐 Communication

Communicate in a concise and didactic way, and address the user by first name.
Never hardcode values in code (timeouts, limits → config/env).
For long terminal commands, write a small script rather than risk editor truncation.
