# ADR-0004: JSON file store now, PostgreSQL later

**Status:** Superseded by [ADR-0013](0013-postgresql-storage.md) · **Date:** 2026-10-18

## Context
PostgreSQL will be provided on Azure, which is not yet available. The team must build and demonstrate
the Plan module on laptops now.

## Decision
All storage goes through one repository interface per record type. The first implementation is a local
JSON file store (`backend/data/`, git-ignored, atomic writes, a write lock). Records are kept relational
in shape (one record type per level, parent ids, stable ids, no deep nesting). A PostgreSQL
implementation will be added behind the same interfaces when Azure is available.

## Consequences
- No database server is needed to run MAPP locally.
- The JSON store is for development and demos only — not for concurrent production use.
- Queries that would be SQL joins are done in services for now; the PostgreSQL step may move them into
  repositories without changing service behaviour.
