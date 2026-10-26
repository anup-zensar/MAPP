# ADR-0006: MAPP's own design language

**Status:** Accepted; the visual part is replaced by the product owner's decisions D-310–D-314 (Shell's
enterprise design language, design system version 2, constitution 1.1.0); the layout and CRUD patterns
stay · **Date:** 2026-10-18

## Context
Shell users find Hive9 complex and dated. The product owner wants MAPP to be friendly to beginners,
with a simple, modern design (white, a sidebar, summary cards). Shell's target architecture mentions the
Shell Design System (RCF), which is not available to the team now.

## Decision
MAPP has its own design language, written down in `docs/product/design-system.md` (tokens, layout,
components, CRUD pattern, statuses, wording). Hive9 is the functional reference only: its screens are not
copied. If the Shell Design System becomes available, its components may replace ours behind the same
tokens and patterns.

## Consequences
- Every feature's screens are reviewed against the design system.
- Shared components are built once in feature 001 and reused.
