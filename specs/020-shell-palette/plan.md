# Implementation Plan: Shell Palette

**Branch**: `feature/020-shell-palette` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/020-shell-palette/spec.md`

## Summary

Give every colour token in `frontend/src/styles/tokens.css` the exact value of constitution X in
both themes, keeping today's names, and add the missing palette tokens (`--raised`, the link
states, the Disabled tokens). Then change only the CSS that the palette's rules require: retire
tints, shades and fades of palette colours; put menus, popovers, dialogs and toasts on Raised; and
switch secondary text, links and red text to Text Strong wherever they sit on a background where
they would fall below 4.5:1. Finally align `docs/product/design-system.md`.

## Technical Context

**Language/Version**: TypeScript 5.9, CSS Modules (no change to the stack)

**Primary Dependencies**: none added

**Storage**: N/A

**Testing**: existing Vitest suite; Playwright end-to-end tests with axe (WCAG 2.2 AA); a one-off
axe crawl of every page in both themes for verification (not committed)

**Target Platform**: the web app in current browsers

**Project Type**: web application (frontend only)

**Constraints**: palette values exact; WCAG 2.2 AA; no file moves (026), no comment pruning (021), no
backend, contract or `src/api/` change (022)

**Scale/Scope**: one token file, about 90 CSS Modules, one SVG, a few doc comments, one document

### How the contrast rules are applied

A CSS custom property set on a container applies to everything inside it, so a container whose
background makes Text Subtle, links or red text too faint redefines them as Text Strong:

| Where | Override |
|---|---|
| Hover Background (hovered rows and items, highlighted rows) | `--ink-2`, `--link`, `--danger-text`, `--status-problem-text` → `var(--ink)` |
| Foundation Base inside cards (`--gray-bg`: table headers, totals, read-only boxes) and the page header | `--ink-2`, `--link` → `var(--ink)` |
| Raised, dark theme only (menus, popovers, dialogs) | `--ink-2` → `var(--ink)` |
| Text set directly on Press Background (`--chip`) | `color: var(--ink)` |

Red on any tint is replaced by a red fill with white text (chips, highlighted destructive menu
items) or by a framed white box with a red bar (error boxes).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle / standard | Check | Result |
|---|---|---|
| I Spec-first | Spec with FR-001–FR-011 and acceptance scenarios; branch `feature/020-shell-palette` | Pass |
| II Plain language | No wording on screen changes | Pass |
| III Documented design | Design system §3.1, §3.11, §4, §5, §9 and §12 updated with the code | Pass |
| IV Two parts | Frontend only; the server is untouched | Pass |
| V Simplicity | No new dependency or tool; one override pattern used everywhere | Pass |
| VI Meaningful tests | No test written for its own sake; axe in the end-to-end tests proves contrast | Pass |
| VII Security | No data, network or secret change | Pass |
| VIII Clean construction | New files are documents only; comments added explain why (contrast) | Pass |
| IX Experience | Hierarchy kept; one yellow action; nothing greyed out without a reason | Pass |
| X Shell palette | Every palette role carries Shell's exact value in both themes; usage and contrast rules applied; TODO(DARK_TOKENS) values kept and listed | Pass |
| F6 Accessibility | Contrast ≥ 4.5:1 text, ≥ 3:1 borders and focus; colour never the only signal | Pass |
| Governance → Alignment | Only colours change; files keep their folders and comment style | Pass |

**Post-design re-check**: unchanged — Pass. No Complexity Tracking entries.

## Project Structure

### Documentation (this feature)

```text
specs/020-shell-palette/
├── spec.md
├── plan.md
└── tasks.md
```

### Source Code (repository root)

```text
frontend/
├── public/favicon.svg              # the mark's tile in Text Strong
└── src/
    ├── styles/tokens.css           # palette values, both themes
    ├── styles/global.css           # focus ring comment
    ├── app/*.module.css            # shell: sidebar, menus, footer
    ├── components/**/*.module.css  # shared components
    └── features/**/*.module.css    # feature screens: grey fills, menus, red boxes
docs/product/design-system.md
CLAUDE.md                           # the palette note (the tokens are aligned now)
```

**Structure Decision**: no file is added to or moved within `frontend/`; every change is inside an
existing file.

## Complexity Tracking

None.
