# Lineagis specifications

Requirements for the Lineagis **lineage graph engine** (v1.0) and **repository self-analysis** (v1.1+). These documents define *what* to build—not implementation details.

For milestones, stories, and pull requests, see [docs/sdlc/README.md](../sdlc/README.md) and [AGENTS.md](../../AGENTS.md).

## Reading order

1. [lineage-engine-mvp.md](lineage-engine-mvp.md) — **Authoritative v1.0 spec**: ingest, trace, why, data model, conformance fixtures
2. [self-analysis.md](self-analysis.md) — **Authoritative v1.1+ spec**: `analyze`, code-graph model, SA-P1–SA-P10 phases
3. [lineagis_design.md](../lineagis_design.md) — Product roadmap v1.0–v1.2 (informative)
4. [lineagis_architecture_overview.md](../lineagis_architecture_overview.md) — Layers, repository layout (informative)
5. [Self-Analysis Design Plan](../plans/Self-Analysis%20Design%20Plan.md) — Informative vision (normative requirements in `self-analysis.md`)

## Spec template

New specs follow [_template.md](_template.md): summary, goals, non-goals, personas, user stories, `FR-*` / `NFR-*`, acceptance criteria, open questions.

## Requirement IDs

| Prefix | Area |
|--------|------|
| `FR-LIN-*` / `AC-LIN-*` | Lineage graph engine (v1.0) |
| `US-LIN-*` | Lineage user stories |
| `FR-SA-*` / `AC-SA-*` | Repository self-analysis (v1.1+) |
| `US-SA-*` | Self-analysis user stories |

Delivery priority (Must / Should / Deferred) is defined in [lineage-engine-mvp.md#delivery-matrix](lineage-engine-mvp.md#delivery-matrix).
