# Implementation Plan: Search the whole hierarchy

**Branch**: `feature/039-search-whole-hierarchy` | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/039-search-whole-hierarchy/spec.md`

## Summary

The plan tree's search and the Plans list's search match every level below the plan, line items included. The
plans service gains a `Searcher` port that reads a set of plans' campaigns, initiatives, activities and line items
with the four batched store reads that exist already (`CampaignsOfPlans`, `InitiativesOfCampaigns`,
`ListByInitiatives`, `ListByActivities`) and keeps those whose name or ID contains the text (research R1, R2).
`GET /api/v1/plans/{planId}/search` answers the plan tree; the Plans list's `q` also lists plans through their
contents and says where it matched (`foundIn`). The web app adds the matching line items as rows under their
activities while a search is typed.

## Technical Context

**Language/Version**: Go (pinned in `go.mod`); TypeScript 5.9, React 19

**Primary Dependencies**: Gin (handler), TanStack Query, openapi-fetch; nothing new

**Storage**: PostgreSQL through the existing stores; no migration, no new query

**Testing**: Go `testing` with the area fakes, contract tests through `apitest`; Vitest + Testing Library + MSW;
Playwright for the P1 story

**Target Platform**: the local API and web app (constitution, Current Phase & Scope)

**Project Type**: web application (`backend/`, `frontend/`)

**Performance Goals**: SC-001 (1 s for a 2,000-line-item plan), SC-002 (004 SC-009, 1 s for 200 plans)

**Constraints**: scope from the signed-in actor only (VII); four statements whatever the number of plans (B5)

**Scale/Scope**: 2 screens (plan page, Plans page), 1 new endpoint, 1 new response field

## Constitution Check

| Principle | How this feature meets it | Status |
|---|---|---|
| I Spec first | Spec 039 with acceptance scenarios; amends 004 FR-003, FR-013, FR-014 (D-366) | Pass |
| II/IV Ownership, adapters | No GSAP data; plans reads its levels through a port wired in `internal/app` | Pass |
| III Documented design | research R1–R4, contract, data model, quickstart; D-366 | Pass |
| V Simplicity | Existing batched reads, matching in Go as All work does; no new table, query or package | Pass |
| VI Tests | Service rules on fakes; contract test for the endpoint and the list field; component tests for the rows, the note and the card; e2e for US1 | Pass |
| VII Security, scope | The plan must be in the actor's scope (404 otherwise); the list searches only plans in scope; new route rule `view()`; no new store read, so no new isolation test | Pass |
| VIII Readability | Comments within 5 %, no requirement IDs in comments | Pass |
| IX Experience | Rows open their path; one quiet note on failure; "Found in" in words, never colour alone | Pass |
| X Palette | No new colour; the row uses the LINE ITEM level tag | Pass |

## Project Structure

### Documentation (this feature)

```text
specs/039-search-whole-hierarchy/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/api.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code

```text
backend/
├── api/openapi.yaml                         # GET /plans/{planId}/search; PlanSummary.foundIn
├── internal/plan/search.go                  # SearchMatch, the Searcher port, Service.Search, the list's contents match
├── internal/plan/handler.go                 # GET /plans/:planId/search
├── internal/plan/list.go                    # itemsOf lists plans matched through their contents
├── internal/access/routes_planning.go       # route rule
└── internal/app/plan_search.go              # the Searcher adapter over the four level stores
frontend/src/
├── features/plans/api.ts                    # usePlanSearch
├── features/plans/tree/usePlanSearchRows.tsx # debounced search → line item rows by activity
├── features/plans/tree/PlanTree.tsx         # rows below activities, path opening, the note
├── features/plans/FoundIn.tsx               # "Found in …" on the card and the table
└── features/plans/PlanPage.tsx              # wires the tree's search
```

**Structure Decision**: the existing `plan` area owns both searches, since both are searches of plans; the levels
are read through a port, as the level registry does (D-195).

## Complexity Tracking

None.
