# ADR-0003: Go API as one layered program

**Status:** Accepted; superseded in part by [ADR-0011](0011-gin-transport-and-opentelemetry.md) (Gin
replaces the standard-library HTTP server; one layered program stays) · **Date:** 2026-10-18

## Context
The product owner asked for a Go backend that stays easy to understand. Shell's target architecture uses a
Go API with packages per business domain.

## Decision
One Go program (`backend/cmd/api`) using the standard-library HTTP server. One package per business area
(`plans`, `activities`, `lineitems`, …), each with the same three layers: handler → service → repository.
All business rules live in services. The API is REST + JSON described by `backend/api/openapi.yaml`.

## Consequences
- New developers learn one pattern and find it in every package.
- No microservices, message queues or caches now; adding one needs a new ADR.
- The future GSAP worker (Shell target) will be a second program in the same module, reusing packages.
