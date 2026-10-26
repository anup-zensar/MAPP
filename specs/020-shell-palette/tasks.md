---

description: "Tasks for 020 Shell Palette"
---

# Tasks: Shell Palette

**Input**: Design documents from `specs/020-shell-palette/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md)

**Tests**: no new tests (spec, Assumptions). Contrast is proven by axe in the existing end-to-end
tests and by a one-off axe crawl of every page in both themes; no existing test asserted old colour
values.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an unfinished task)
- **[Story]**: the user story the task belongs to (US1–US4)

---

## Phase 1: Setup

- [X] T001 Create the branch `feature/020-shell-palette` from `feature/019-alignment-guardrails` and install the web app's dependencies (`npm ci` in `frontend/`)

---

## Phase 2: Foundational (the tokens every screen reads)

- [X] T002 Give the light tokens Shell's values and add `--raised`, `--link-hover`, `--link-pressed`, `--link-visited` and the Disabled tokens in `frontend/src/styles/tokens.css` (FR-001, FR-002)
- [X] T003 Give the dark tokens Shell's dark values, drop the dimmed yellow and dark red, keep the TODO(DARK_TOKENS) values, in `frontend/src/styles/tokens.css` (FR-002, FR-006)
- [X] T004 Point the navy and dark-blue roles at palette tokens: focus ring, mark, avatars, banner, tooltips, `--blue` as Link; retire `--brand-yellow-tint` and `--danger-bg-hover`; darken team gold, in `frontend/src/styles/tokens.css` (FR-003, FR-004)

**Checkpoint**: every palette role carries Shell's value.

---

## Phase 3: User Story 1 - Every screen shows Shell's exact colours (Priority: P1) 🎯 MVP

**Goal**: no tint, shade or fade of a palette colour remains; red and yellow keep to their meanings.

**Independent Test**: User Story 1's acceptance scenarios in [spec.md](spec.md).

- [X] T005 [P] [US1] Buttons: Disabled tokens instead of opacity, secondary hover on `--hover`, red hover by underline, in `frontend/src/components/Button/Button.module.css`; drop the move buttons' fade in `frontend/src/features/settings/shell-attributes/ShellAttributes.module.css` and the red `brightness` filter and ink-on-red confirm label in `frontend/src/features/settings/shared/SettingDeleteDialog.module.css` (white text on red, found by the crawl) (FR-003, FR-009)
- [X] T006 [P] [US1] Problem chips as a red fill with white text in `frontend/src/components/StatusChip/StatusChip.module.css`, `frontend/src/components/CompletenessChip/CompletenessChip.module.css`, `frontend/src/features/campaigns/AttentionChip.module.css`; problem calendar items without a red tint in `frontend/src/features/all-work/MonthCalendar.module.css` and `CalendarView.module.css` (FR-003, FR-009)
- [X] T007 [P] [US1] Error boxes as framed white boxes with a red bar in `FormPanel`, `FormErrorSummary`, `ErrorState` (`frontend/src/components/`) and in `ImportDialog`, `LinkTransactionDialog`, `AllocationDialog`, `SplitActivityDialog`, `SignInPage`, `PanelFrame`, `rates`, `Notice`, `ApprovalAlert` (`frontend/src/features/`) (FR-003)
- [X] T008 [P] [US1] Selection and blue tints replaced: current sidebar item and highlighted rows on `--hover` (`Sidebar`, `TreeTable`, `LineItemsSection`, `BinRow`, `ShellAttributeFields`), menu highlights on `--hover` (`AccountMenu`, `ActionsMenu`, `Combobox`), a filter in use with Border Strong (`FilterMenu`), search highlight on `--chip` (`Highlight`), info-coloured notes, bars and dots (`GettingStartedCard`, `NoAccess`, `ProgressBar`, `AgeingBars`, `NotificationList`, `WorkflowFormPanel`) (FR-003, FR-004)
- [X] T009 [P] [US1] Focused inputs with Border Strong in `inputs.module.css`, `TextField`, `SearchField`, `Combobox`, `rates`; the ⋯ buttons in Text Subtle instead of faded in `TreeTable` and `campaignRows` (FR-003, FR-008)
- [X] T010 [P] [US1] Mark, avatar, banner and favicon on Text Strong with Surface-coloured contents: `Logo`, `Avatar`, `DemoBanner` and `frontend/public/favicon.svg`; the pin star filled with ink in `frontend/src/features/plans/PinButton.module.css` (FR-004, FR-009)

**Checkpoint**: User Story 1 works on its own.

---

## Phase 4: User Story 2 - Text stays readable wherever it sits (Priority: P1)

**Goal**: secondary text, links and red text are Text Strong on every grey.

**Independent Test**: User Story 2's acceptance scenarios; the axe crawl with rows hovered.

- [X] T011 [US2] Page header, footer and tabs on the grey page in Text Strong: `PageHeader`, `AppLayout`, `Tabs` (FR-005)
- [X] T012 [US2] Hovered and highlighted rows and items redefine Text Subtle, links and red text as Text Strong in the table and list styles of `frontend/src/components/` and `frontend/src/features/` (FR-005)
- [X] T013 [US2] Grey fills inside cards (table headers, totals, read-only boxes, outside days) redefine Text Subtle and links as Text Strong; text on `--chip` in Text Strong (`.calculated`, `.default` tags) (FR-005)
- [X] T014 [US2] Focus rings: Text Strong everywhere; positive offsets of 1 px and 4 px made 2 px; inset rings in scrolling boxes keep their 2 px inset (FR-008)
- [X] T015 [US2] Fix every colour finding of the axe crawl (light theme): the All work filter bar, the Financials source line, the not-found and no-access messages and the Shell attribute editor on the grey page; status chip tints mixed over Surface so they keep 4.5:1 on a hovered row

---

## Phase 5: User Story 3 - The dark theme uses Shell's dark values (Priority: P2)

- [X] T016 [US3] Menus, popovers, dialogs and toasts on `--raised`, with Text Subtle as Text Strong in the dark theme, in every floating layer of `frontend/src/app/`, `frontend/src/components/` and `frontend/src/features/` (FR-006)
- [X] T017 [US3] Fix every colour finding of the axe crawl (dark theme): none found

---

## Phase 6: User Story 4 - The design system says the same as the code (Priority: P3)

- [X] T018 [US4] Update `docs/product/design-system.md` §2.1, §3.1, §3.10, §3.11, §4.1, §4.3, §5, §9, §12 and the change history (FR-010)

---

## Phase 7: Polish & verification

- [X] T019 Correct doc comments that named the old colours (`Sidebar.tsx`, `TreeTable.tsx`, `Logo.tsx`, `Avatar.tsx`, `DemoBanner.tsx`, `ShellAttributeFields.tsx`, `LinkSummaryCard.tsx`, `global.css`) and the note in `CLAUDE.md` that said the tokens still held older values
- [X] T020 Run `npm run format:check`, `typecheck`, `lint`, `test`, `build` and `test:e2e` in `frontend/` (FR-011)

## Dependencies

Phase 2 before every story; US1 and US2 touch the same files and run one after the other; US3
after US1; US4 last, so the document describes the final code.
