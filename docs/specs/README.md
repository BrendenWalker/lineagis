# Lineagis specifications

Requirements for the Lineagis **lineage graph engine** (v1.0). These documents define *what* to build—not implementation details.

For milestones, stories, and pull requests, see [docs/sdlc/README.md](../sdlc/README.md) and [AGENTS.md](../../AGENTS.md).

## Reading order

1. [lineage-engine-mvp.md](lineage-engine-mvp.md) — **Authoritative v1.0 spec**: ingest, trace, why, data model, conformance fixtures
2. [lineagis_design.md](../lineagis_design.md) — Product roadmap v1.0–v1.2 (informative)
3. [lineagis_architecture_overview.md](../lineagis_architecture_overview.md) — Layers, repository layout (informative)

## Spec template

New specs follow [_template.md](_template.md): summary, goals, non-goals, personas, user stories, `FR-*` / `NFR-*`, acceptance criteria, open questions.

## Requirement IDs

| Prefix | Area |
|--------|------|
| `FR-LIN-*` / `AC-LIN-*` | Lineage graph engine |
| `US-LIN-*` | Lineage user stories |

Delivery priority (Must / Should / Deferred) is defined in [lineage-engine-mvp.md#delivery-matrix](lineage-engine-mvp.md#delivery-matrix).
