# ADR-0006: ZenReach design language

**Status:** Accepted · **Date:** 2026-10-18

## Context
Shell users find Hive9 complex and dated. The product owner wants NorthStar to be friendly to beginners and
to use ZenReach's design (white, modern, sidebar, summary cards). Shell's target architecture mentions the
Shell Design System (RCF), which is not available to the team now.

## Decision
NorthStar follows the ZenReach design language, written down in `docs/product/design-system.md` (tokens,
layout, components, CRUD pattern, statuses, wording). Hive9 is the functional reference only. If the Shell
Design System becomes available, its components may replace ours behind the same tokens and patterns.

## Consequences
- Every feature's screens are reviewed against the design system.
- Shared components are built once in feature 001 and reused.
