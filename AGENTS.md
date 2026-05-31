# AI agent instructions — Lineagis

Instructions for humans and AI assistants working on this repository. Tool-agnostic: apply in Cursor, Claude Code, Copilot, Codex, or any agent that can read repo docs.

**Detailed SDLC:** [docs/sdlc/README.md](docs/sdlc/README.md)

---

## Project context

- **Product:** Open-source trust platform for OCI artifact publishing, signing, provenance, and policy (see [README.md](README.md)).
- **Requirements:** [docs/specs/](docs/specs/) — authoritative for *what* to build (FR/NFR, acceptance criteria, MVP Must/Should/Deferred).
- **Implementation:** Go services and CLI under `cmd/`, `internal/`; local stack via `docker-compose.yml` and `Makefile`.

Before coding, read the relevant spec(s) and [00-overview.md](docs/specs/00-overview.md) delivery matrix so work stays in MVP scope.

---

## SDLC overview

```
Plan → Spec → Build → Review (human) → Merge
```

| Phase | Owner | AI role |
|-------|--------|---------|
| **Plan** | Human + AI | Clarify goal, scope, risks; propose approach; **wait for human approval** on large or ambiguous work. |
| **Spec** | Human (approve), AI (draft) | Break work into epics/milestones/stories; tie to `FR-*` / `AC-*`; update or add spec sections when behavior changes. |
| **Build** | AI (implement), Human (review) | Branch, implement, test, open PR; **do not merge** without human approval. |
| **Review** | Human | Review PR, request changes, merge to `main`. |

---

## Plan

1. State the **goal** and **success criteria** (what “done” means).
2. Map to **spec IDs** (`FR-*`, `US-*`, `AC-*`) or flag gaps that need spec work first.
3. Check **MVP priority** (Must / Should / Deferred) in [00-overview.md](docs/specs/00-overview.md#mvp-delivery-matrix).
4. List **touch points** (packages, APIs, migrations, docs, CI).
5. Propose a **short plan** (bullets). For multi-file or cross-cutting changes, get explicit human sign-off before Spec/Build.

---

## Spec

Organize work in four levels (see [docs/sdlc/README.md](docs/sdlc/README.md) for templates):

| Level | Purpose | Typical artifact |
|-------|---------|------------------|
| **Epic** | Large capability (e.g. “Artifact publishing”) | Issue label + overview spec link |
| **Milestone** | Shippable increment (e.g. “Phase 1 — Must items”) | GitHub milestone or [milestone doc](docs/sdlc/_template-milestone.md) |
| **Story** | User-visible slice, testable alone | Issue / [story doc](docs/sdlc/_template-story.md) |
| **Task** | Implementation step | Checklist on story or PR |

**Rules:**

- New behavior → update or add requirements in `docs/specs/` (use [_template.md](docs/specs/_template.md)); stories reference spec IDs, not duplicate long requirements.
- Every story has **acceptance criteria** mapped to `AC-*` or explicit given/when/then bullets.
- Do not implement **Deferred** items unless the human explicitly expands scope.

---

## Build

### Branching

| Granularity | Branch pattern | When to use |
|-------------|----------------|-------------|
| Milestone | `milestone/<short-name>` | Several stories, one integrated deliverable |
| Story | `story/<id>-<short-slug>` | Single story, one PR (preferred default) |
| Fix / tiny | `fix/<short-slug>` | One concern, no milestone |

Branch from latest `main`. Rebase or merge `main` before opening PR if the branch is long-lived.

### Commits and PRs

- **Commits:** Only when the human asks. Small, logical commits; message explains *why* (repo style: short imperative subject, optional body).
- **PRs:** One story per PR when possible. PR description must include:
  - Linked story/issue
  - Spec references (`FR-*`, `AC-*`)
  - Summary of changes and **test plan**
- **CI:** `go test`, lint, and smoke targets per [Makefile](Makefile) / [.github/workflows/ci.yml](.github/workflows/ci.yml) must pass before requesting review.
- **Secrets:** Never commit `.env`, keys, or tokens. Use [.env.example](.env.example).

### Human gates (mandatory)

- Do **not** merge PRs unless the human explicitly requests it.
- Do **not** `git push` unless the human explicitly requests it.
- Do **not** force-push `main` / `master`.
- Spec or scope changes that affect MVP boundaries need **human approval** before Build continues.

---

## Code conventions

- Match existing layout and naming in touched packages.
- Minimal diff; no drive-by refactors.
- Comments only for non-obvious logic.
- Tests for real behavior; avoid trivial assertions.
- Run `go test ./...` (and `make lint` if available) before marking work ready for review.

---

## Quick links

| Resource | Path |
|----------|------|
| SDLC (full) | [docs/sdlc/README.md](docs/sdlc/README.md) |
| Spec index | [docs/specs/README.md](docs/specs/README.md) |
| MVP overview | [docs/specs/00-overview.md](docs/specs/00-overview.md) |
| Story template | [docs/sdlc/_template-story.md](docs/sdlc/_template-story.md) |
| Milestone template | [docs/sdlc/_template-milestone.md](docs/sdlc/_template-milestone.md) |
