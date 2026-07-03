# AI agent instructions — Lineagis

Instructions for humans and AI assistants working on this repository.

**SDLC:** [docs/sdlc/README.md](docs/sdlc/README.md)

---

## Project context

- **Product:** Open-source **software supply-chain lineage graph engine** — ingest SBOM/git/build signals, query provenance with `trace` and `why` (see [README.md](README.md)).
- **Requirements:** [docs/specs/lineage-engine-mvp.md](docs/specs/lineage-engine-mvp.md) — authoritative for v1.0 (`FR-LIN-*`, `AC-LIN-*`); [docs/specs/self-analysis.md](docs/specs/self-analysis.md) — v1.1+ self-analysis (`FR-SA-*`, `AC-SA-*`).
- **Implementation:** Go CLI under `cmd/lineagis/`; graph core under `internal/core/`, ingest under `internal/ingest/`, in-memory store under `internal/storage/memory/`.

Before coding, read the lineage spec and stay within v1.0 Must scope unless the human expands it.

---

## SDLC overview

```
Plan → Spec → Build → Review (human) → Merge
```

| Phase | Owner | AI role |
|-------|--------|---------|
| **Plan** | Human + AI | Clarify goal, scope, risks; **wait for human approval** on large or ambiguous work. |
| **Spec** | Human (approve), AI (draft) | Tie work to `FR-LIN-*` / `AC-LIN-*`; update spec when behavior changes. |
| **Build** | AI (implement), Human (review) | Branch, implement, test, open PR; **do not merge** without human approval. |
| **Review** | Human | Review PR, request changes, merge to `main`. |

---

## Plan

1. State **goal** and **success criteria**.
2. Map to **spec IDs** (`FR-LIN-*`, `US-LIN-*`, `AC-LIN-*`) or flag spec gaps first.
3. Check **Must / Should / Deferred** in [lineage-engine-mvp.md#delivery-matrix](docs/specs/lineage-engine-mvp.md#delivery-matrix).
4. List **touch points** (packages, examples, conformance tests, docs).
5. Propose a **short plan**; get sign-off before large cross-cutting changes.

---

## Build

### Branching

| Granularity | Pattern | When |
|-------------|---------|------|
| Milestone | `milestone/<name>` | Integrated deliverable |
| Story | `story/<id>-<slug>` | Default (one PR) |
| Fix | `fix/<slug>` | Small fix |

### Commands

| Action | Command |
|--------|---------|
| Build CLI | `make build` |
| Lineage tests | `make test-lineage` |
| All tests | `make test` |
| Lint | `make lint` |
| Lineage smoke | `make smoke-lineage` |

### Human gates

- Do **not** merge or **push** unless the human explicitly requests it.
- Do **not** add heavy dependencies without approval (see spec OQ-LIN-003).
- Spec or MVP boundary changes need **human approval** before Build continues.

---

## Code conventions

- Graph logic → `internal/core/` only (no SBOM parsing in `graph/`).
- Format parsers → `internal/ingest/<source>/`.
- CLI wiring → `cmd/lineagis/` (thin; delegate to `internal/core/query`).
- v1.0 is **offline-capable** (no API/DB required for ingest/trace/why).

---

## Quick links

| Resource | Path |
|----------|------|
| Lineage MVP spec | [docs/specs/lineage-engine-mvp.md](docs/specs/lineage-engine-mvp.md) |
| Self-analysis spec | [docs/specs/self-analysis.md](docs/specs/self-analysis.md) |
| Architecture | [docs/lineagis_architecture_overview.md](docs/lineagis_architecture_overview.md) |
| Design / roadmap | [docs/lineagis_design.md](docs/lineagis_design.md) |
| SDLC | [docs/sdlc/README.md](docs/sdlc/README.md) |
