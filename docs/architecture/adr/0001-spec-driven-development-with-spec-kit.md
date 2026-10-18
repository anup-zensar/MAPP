# ADR-0001: Spec-driven development with GitHub Spec Kit

**Status:** Accepted · **Date:** 2026-10-18

## Context
NorthStar replaces a large Shell application. Shell needs to see and agree what will be built, and the
team uses AI coding agents. Without a written, versioned intent, AI-generated code drifts and is hard to
review.

## Decision
Use GitHub Spec Kit. Every feature has `specs/NNN-short-name/` with `spec.md` (what/why), `plan.md`
(how), `tasks.md` (work) and tests traced to requirements. The Spec Kit skills for Claude Code are
installed in `.claude/skills/`. The project rules are in `.specify/memory/constitution.md`.

## Consequences
- Behaviour changes start in the spec; reviews compare code with the spec.
- Baseline specs for the whole Plan module are written up front on `main`; each feature gets its own
  branch when planning starts (see `docs/sdd/sdd-process.md`).
