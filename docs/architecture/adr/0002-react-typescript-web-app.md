# ADR-0002: React + TypeScript web app

**Status:** Accepted · **Date:** 2026-10-18

## Context
The product owner asked for a React frontend. Shell's target architecture uses React + TypeScript. The
ZenReach reference implementation, whose design NorthStar reuses, is React.

## Decision
A single-page web app in React with TypeScript in strict mode, built with Vite, in `frontend/`. API types
are generated from the OpenAPI file. Component and styling libraries are chosen in feature 001's plan,
favouring a small set of well-known libraries.

## Consequences
- Type errors between the app and the API are caught at build time.
- The app is static files and can later be served by any web server or Azure Container Apps.
