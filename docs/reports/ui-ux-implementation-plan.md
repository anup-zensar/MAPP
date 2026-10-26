# Updated UI/UX implementation plan

| | |
|---|---|
| **Deliverable** | Updated UI/UX implementation plan |
| **For** | Product owner, the Shell client team and the MAPP delivery team |
| **Date** | 20 October 2026 |
| **Applies** | [Design system version 2](../product/design-system.md) (binding) and the [UX/UI recommendations](ux-design-recommendations.md) |
| **Closes** | The 97 entries of the [UI audit](ui-audit.md) and the UI-related findings of the [functional review](functional-review.md) |
| **Branch** | `plan-module` (feature 011 on `lane-011` is merged before area 2.8) |

**How items are cited.** Both reports use overlapping IDs, so this plan always names the source: **UI audit**
IDs (X-, S-, A-, H-, P-, PL-, C-, F-, AC-, AF-, LI-, AP-, AW-, ST-, PE-, PR-, M-, PH-, D-, AX-) and **functional
review** IDs written with the prefix **FR·** (FR·P-01, FR·A-02, FR·X-01 …) — not to be confused with the specs'
functional requirements (FR-024).

---

## Summary

The revamp runs in two phases.

- **Phase 1 — foundation (about 16 developer days).** A measuring test harness first; the four layout bugs
  that break key screens; the Shell tokens in light and dark mode; Jost and Arial; the type, spacing, height
  and breakpoint scales with a lint rule that keeps them; every shared component restyled to its spec plus
  five new ones (`Tabs`, data table standard, `DescriptionList`, `InlineAlert`, `TopBar`); and the new app
  shell. Most audit findings close here, because they come from three root causes (X-01, X-02, X-10) and from
  parts built several ways.
- **Phase 2 — page by page (about 15 developer days)**, in eight areas that can run in parallel once phase 1
  is merged: Home; Plans and the plan page; campaign and initiative panels; activities and line items;
  Approvals; All work; Settings, people, teams and profile; Activity log and Recycle bin.

**Every step is accepted the same way** (§0): screenshots at 1440 × 900 and 390 × 844 in light and dark
mode, axe WCAG 2.2 AA with no violations, no sideways scrolling, the measurable layout checks of design
system §11, and the existing tests green.

**Coverage:** all **97 of 97** UI audit entries (99 IDs with M-01 and D-03) have a step (appendix A). Of the **25** functional review
findings, **18** are addressed (FR·P-04 and FR·P-05 in part, several after a product-owner decision) and
**7** are outside UI design and stay with their owners (appendix B).

---

## 0. Ground rules and the definition of done

**Ground rules**

1. Behaviour does not change unless a step says so. Steps that do change behaviour (tabs on record pages,
   the date input, the select rule, who sees Approve) are listed with the decision they need (§6).
2. Shared components have one owner (D-197). In phase 2, areas change only their own feature folders; a
   needed component change goes back to the foundation owner.
3. Specification and code change together: a step that changes a token or a component updates the design
   system in the same change.
4. Servers for checks run on `127.0.0.1` only, on a free port pair such as 18131 / 14131 — never 8080, 5173,
   18080, 14173, 18111 or 14111. Demo data only; no personal data in screenshots, logs or test names.

**Definition of done (every step of both phases)**

| # | Criterion | How it is measured |
|---|---|---|
| D1 | **Screenshots** of every state the step touches at **1440 × 900** and **390 × 844**, **light and dark** (four per state), full page plus a viewport shot for panels, menus and dialogs, reviewed against design system v2 and approved by the product owner | Playwright visual suite (§4.3); baselines updated only in a reviewed change |
| D2 | **axe-core, WCAG 2.2 AA: zero violations** with tags `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa`, on every captured state, both themes, both widths | `expectNoAxeViolations` in `e2e/fixtures.ts` with the new tag |
| D3 | **No horizontal scroll**: `document.documentElement.scrollWidth ≤ window.innerWidth` at 390, 600, 960, 1280 and 1440 px | Layout helper `expectNoSidewaysScroll` |
| D4 | **One primary button** per page, side panel or dialog | Layout helper `expectOnePrimaryPerScope` |
| D5 | **Title position**: the page title's top edge at the same y on every page (138 px at desktop with the banner, ± 1 px) | Layout helper `titleTop` |
| D6 | **Numbers aligned**: every numeric cell's right edge equals its header's (± 1 px) | Layout helper `expectNumbersUnderHeaders` |
| D7 | **Text ≥ 12 px** and **targets ≥ 24 × 24 px** on every captured state | Layout helpers `expectMinTextSize`, `expectTargetSize` |
| D8 | **Keyboard**: the area's main flow done with the keyboard alone; focus always visible and never hidden under a sticky bar or toast | Playwright keyboard journey + manual check |
| D9 | `npm run check` green (types, ESLint, **stylelint**, Prettier, Vitest); `go test ./...` untouched or green; the e2e smoke, core-flow and module-rest journeys green | CI / local run |
| D10 | The §11 checklist of the design system ticked for the touched pages | Reviewer, on the screenshots |

---

## 1. Phase 1 — foundation

Order: 1.1 first (so every later step is measured), then 1.2 (ships at once), then 1.3–1.7. Estimates are
developer days.

### 1.1 Test harness first (2 days)

- `e2e/layout.ts`: the measuring helpers of D3–D7 (sideways scroll, primary count per scope, title top,
  numbers under headers, minimum text size, target size, row height, repeated headings).
- `e2e/visual.spec.ts`: the screenshot suite with four Playwright projects (desktop-light, desktop-dark,
  phone-light, phone-dark; `colorScheme` emulated, appearance reset to "follow the computer" by
  `signInAs`), fixed clock (`page.clock`) and masks for times, so screenshots are stable.
- `e2e/fixtures.ts`: add `wcag22aa` to the axe tags; run axe on every visual state.
- Capture **"before" screenshots** of all states (§4.3) for the before/after review.
- **Decision needed:** this extends D-040 (smoke-only end-to-end) — §6, item 8.

### 1.2 Global fixes: the bugs that break key screens (1 day)

| Fix | Files | Closes |
|---|---|---|
| Replace `overflow-wrap: anywhere` with `overflow-wrap: break-word` (63 rules in 46 files); `white-space: nowrap` on person labels and team tags in table cells; min widths for Owner (160) and Teams columns; tree name cell wraps to 2 lines | `PersonLabel.module.css:17,32`, `TeamPill.module.css:13`, `TreeTable.module.css:124` and the other 43 files | UI audit **X-01** names break per letter (High), **PL-01** owner column collapses (High), **P-01** Plans table breaks (High, with 2.2); FR·**P-01** (High) |
| `position: relative` on every sideways-scrolling table box | `TreeTable.module.css:2`, `PlanTable.module.css:2`, `LineItemsSection.module.css`, `settings/shared/SettingsTable.module.css`, `people/lists.module.css` | UI audit **X-02** 9 phone pages too wide (High), **PH-01** (High), **PH-02** panels off-screen (High), **C-01** campaign panel unusable on phones (High), **ST-07** Settings pages too wide (High); FR·**X-01** phone screens scroll sideways (with the tree's 260 px name cell in 2.2) |
| Line items numbers: fix the specificity bug (`.table td {text-align:left}` outranks `.number`) | `LineItemsSection.module.css:24,37` | UI audit **X-07** (part), **AC-05** numbers not under headers |
| Percent format "38%" (no space) | `lib/format.ts` and callers | UI audit **X-17** percentages break across lines |
| Home "My activities": two-column grid row (text · amount), amount pinned right | `features/home/MyActivitiesCard.module.css:73` | UI audit **H-01** amounts jump (High) |

Acceptance: D1–D10 on the plan page, Plans (table), activity page, Home, Models and People; at 390 px every
page passes D3 and the campaign panel's Close and Save lie inside the viewport.

### 1.3 Tokens, dark mode and fonts (1.5 days)

- `tokens.css`: every value of design system §3.1–§3.8 and §4 in light and dark mode — Shell surfaces, ink
  and greys, `--control-border`, yellow primary with dark text, `--focus-ring`, Shell red for danger,
  status, level and team colours, navy banner, shadows, radii, control heights, spacing, layout sizes, type
  tokens, motion, under the token names of design system §3 (`--brand-yellow*`, `--info`/`--success`/
  `--warning`/`--danger` and their `-text` shades, `--radius-1`…`--radius-4`). The retired
  `--brand-gradient*` tokens are removed with their last use; `--blue` keeps its name as Shell dark blue
  (links and checked controls) and no longer marks focus or selection.
- `global.css`: `font-size: var(--text-sm)`, line height 20 px, `letter-spacing: normal`; focus ring
  `var(--focus-ring)` on `:focus-visible` only; links in `--link` and underlined in sentences; skip link
  in yellow;
  `scroll-padding-top` for the sticky top bar.
- Fonts: add `@fontsource/jost` **5.3.0** (OFL-1.1), import `latin-600.css` and `latin-700.css` once in
  `main.tsx`; `--font: Arial, Roboto, Helvetica, sans-serif`; `--font-heading: 'Jost', …`. No request to
  any font CDN.
- **Contrast test** (Vitest): parses `tokens.css` and asserts every pair listed in design system §3.1 and
  §4.1–§4.3 meets its threshold in both themes.

Closes: UI audit **S-02** gradient header (with 1.7), **D-02** olive banner in dark mode, **F-05** "Calculated"
chip contrast; the input-border contrast failure (patterns research §7); part of **X-10** (no scale).
Acceptance: D1, D2, D9 on every state of the "before" set (all screens change colour at once).

### 1.4 The scales and the lint guard (3 days)

- Replace the 15 font sizes with the seven type tokens, the off-grid spacing (5, 6, 7, 10, 14, 18, 26 px) with
  `--space-*` (6 → 8 or 4, 10 → 8 or 12, 14 → 12 or 16, 18 → 16 or 20, 26 → 24), the 15 radii with the four,
  the seven control heights with 32/40/48, and the 11 media queries with the breakpoints 600 / 960 / 1280.
- Remove upper-case text, letter spacing and 600 weights for Arial.
- Add **stylelint** (dev dependency) to `npm run check` with: no hex colours outside `tokens.css`; font-size,
  line-height, border-radius, box-shadow and control heights only from tokens; spacing only from
  `--space-*` (plus 0, 1 px, 2 px, auto); only the allowed media queries; no `overflow-wrap: anywhere`,
  `text-transform: uppercase` or non-normal `letter-spacing`.

Closes: UI audit **X-10** no spacing or type scale (root cause), **AX-01** text under 12 px, **X-18** (part:
upper-case headers); the seven-heights misalignment behind **P-04** and **AP-02**.
Acceptance: stylelint passes with no exceptions file; D1, D7 on every state.

### 1.5 Shared components (5 days)

Each component follows its spec in design system §5; each change adds or updates its unit tests.

| Component | Change | Closes |
|---|---|---|
| `PageHeader` | New slots: breadcrumb row (always reserved), title, meta row, description, actions (⋯ · secondary · primary last); removes eyebrow and highlight; phone layout (primary + ⋯ only) | UI audit **X-03** five header layouts (component), **X-19** breadcrumb against the banner, **ST-03** title 31 px lower on sub-pages, **F-04** four-line edit header, **LI-02** "LINE ITEM" twice (with 2.4) |
| `Breadcrumb` | 12 px with chevrons, 24 px targets, "‹ Parent" below 600 px | UI audit **PH-06** breadcrumbs wrap on phones, **PH-05** (part) |
| `Button`, `LinkButton` | Yellow primary with ink text, secondary outlined in ink, ghost, red danger for destructive confirmations only; sizes 32/40/48; busy state | Brand change; supports **X-04** |
| `ActionsMenu` | Item and separator spec; Delete last in red text | Part of **X-05** |
| `Tabs` (**new**, replaces `PillTabs`) | Underline tabs with the 4 px yellow bar, counts, sideways scroll instead of wrapping, state in the address | UI audit **AP-04** pill tabs wrap (with 2.5), **PH-03** |
| `SegmentedFilter` | 32 px joined buttons, ink selected state, counts "Active (5)" | Part of **X-16** |
| `SearchField`, `FilterChips` | Radius 8 (no pill), 24 px clear and remove targets | **P-04** (part), **PH-05** (part) |
| Data table standard (**new**, `components/DataTable/`) | One header style, row, number, name, person, team, status, date and actions column rules; totals row; scroll cue; stacked-card mode below 600 px | UI audit **X-07** numbers aligned differently, **X-18** two header styles, **AW-05** (component), **P-02** (rule) |
| `TreeTable` | Data table standard, level bar, expand targets, Expand all / Collapse all in the card header, plan row with avatar | **PL-04** (with 2.2) |
| `DescriptionList` (**new**) | One key–value layout, 200 px labels | UI audit **X-08** three key–value layouts (component) |
| `Card` | Header centred, min 40 px, no divider, equal heights in a row, table and link variants | UI audit **X-11** titles and dividers misaligned, **X-12** (grid part) |
| `SummaryCard`, `LinkSummaryCard` | Subgrid rows so values align, no icon tiles, 4 / 3 / 2-per-row rules | UI audit **H-03**, **AW-02** values sit lower when a label wraps, **D-01** icon tiles in dark mode, **H-06**, **P-06** (layout part) |
| `StatusChip`, `CompletenessChip`, `LevelTag`, `CountBadge`, `TeamPill`, `Avatar`, `PersonLabel`, `ProgressBar` | Sizes, 12 px minimum, sentence case, tags not pills, no gradient, ≤ 2 chips per row; **one status-word mapping** `lib/statusTone.ts` | UI audit **X-14** status colours clash (mapping), **AX-01** (level tags), **PL-02** (rule); FR·**A-02** (once the rename is confirmed) |
| `SidePanel`, `FormPanel` | Action props (`primaryAction`, `secondaryActions`, `cancel`, `moreActions`) → one footer layout; ⋯ in the header; full width below 600 px; keyboard-only focus ring on Close | UI audit **X-05** five panel footers (component), **AX-03** heavy focus ring on Close, **C-02** second breadcrumb (rule) |
| `ConfirmDialog`, `FormDialog`, `ClearValuesDialog` | One width (512), Close ✕, right-aligned buttons, red confirm only when destructive | UI audit **P-05** dialogs differ, no ✕ |
| `FormFooter` | Sticky across the content column, right-aligned, primary last | UI audit **X-06** form buttons move around |
| `FieldShell` and inputs | 40 px fields, 3.4:1 borders, hint always under the label, marking as a small tag | UI audit **F-02** help text above or below; FR·**P-04** (marking more visible) |
| Select rule (`ChoiceField`, `Combobox`) | One select style (native select drawn as an input ≤ 10 options, Combobox above; checkboxes in columns ≤ 6, Combobox above) | UI audit **X-15** three select styles, **AF-01** long checkbox lists (with 2.4) |
| `DateRangeField` | Text inputs in the person's date format with a calendar popover (WAI-ARIA date picker dialog) | UI audit **X-15** two date formats, **C-04** US dates in panels; FR·**P-03** dates on forms ignore the format |
| `ShellAttributeFields`, `ShellAttributeValues` | Never a heading equal to its parent's; `DescriptionList`; yellow highlight in the Settings preview | UI audit **X-09** duplicate headings, **F-03**, **AF-02**; FR·**P-08** |
| `Toast` | Bottom-left of the content column, above sticky footers; full width below 600 px | FR·**P-09** toasts cover the Save bar |
| `InlineAlert` (**new**) | Info / waiting / positive / problem message under a header | Needed by **AC-02**, **ST-02** |
| `EmptyState`, `ErrorState`, `Skeleton`, `Tooltip`, `InfoTip`, `LockedNote`, `FormErrorSummary`, `SplitEditor`, `CheckboxField`, `MoneyField`, `DecimalField`, text fields | Restyle to spec; ⓘ and checkbox targets 24 px; empty-state action secondary when the header holds the primary | UI audit **PH-05** small tap targets, **AX-02** |
| `lib/format.ts` | One money format per screen, "—" for nothing, percent without space, 24-hour times | FR·**P-07** (rule; needs §6 item 3), FR·**X-08** "Nothing" shown three ways (rule) |

Acceptance: D1–D10 on a component gallery page (development-only route, as feature 001's developer page)
showing every component in every state, plus the before set.

### 1.6 App shell (2.5 days)

- `TopBar` (new; replaces `MobileTopBar`): logo zone, reserved search slot, bell, account menu with My
  profile, Appearance and Sign out; Open menu below 960 px.
- `Sidebar`: white, 240 px, 40 px items, the 4 px yellow bar with bold label and grey fill, "More" group
  label in sentence case, Collapse at the bottom; 64 px rail (by choice, or automatic 600–959 px); drawer
  below 600 px. Bell, account and theme leave the sidebar.
- `AppLayout`: grey canvas, content column fluid to 1300 px and centred, 24 / 16 px padding, footer row.
- `DemoBanner`: navy, 32 px, one line (short text on phones).
- `PublicLayout` / sign-in: one card, brand once, no gradient, 48 px fields, full-width yellow **Sign in**.
- `Logo` and favicon: the navy (`--brand-mark`) tile with the white star.

Closes: UI audit **A-02** content stays 1120 px with the sidebar collapsed, **A-03** 100 px of chrome on
phones, **X-13** fixed 1120 px width (shell part), **S-01** brand twice on sign-in, **S-02** gradient header,
**M-01** (account menu and notifications kept tidy in their new place), **D-03** (dark mode inherits every
fix).
Acceptance: D1–D10 on Home, Plans, the plan page, Settings, sign-in, not found, the drawer (390) and the
collapsed sidebar (1440); keyboard: skip link, drawer open/close with focus return, account menu.

### 1.7 Documents and decisions (0.5 day)

- Decision log: record the product-owner decisions of 20 October 2026 (owned by the decision-log editor).
- Constitution principle III: "one yellow primary action", "WCAG 2.2 AA" (amendment by the constitution
  owner).
- Design system: confirm that tokens and components match version 2 after phase 1; note any deviation.

**Phase 1 exit criteria:** every phase 1 step meets D1–D10; the "after" screenshots of all states exist; the
high-severity audit entries (X-01, X-02, PL-01, P-01, C-01, H-01, ST-07, PH-01, PH-02) are closed; stylelint
runs in `npm run check`.

---

## 2. Phase 2 — page by page

Each area applies the templates of design system §2.4 to its pages and closes the listed findings. Areas
start after phase 1 is merged and can run in parallel (one owner each). Every area meets D1–D10 for the
states listed; the extra acceptance criteria below are specific to the area.

### 2.1 Home (1.5 days)

**Pages and files:** `features/home/*` (Home for a planner and an administrator, notifications).

**Changes:** dashboard template (§2.4 E): greeting title with the currency select (one select style) and
the scope switch in the actions slot; 4 link summary cards (2 × 2 on phones); *Needs your attention* and
*Waiting for my approval* as equal 6 + 6 cards with at most 5 rows and **View all (n)**; *Plans at a
glance* on the data table standard; *My activities* rows as a two-column grid; one wording for the same
problem; *Recent changes* hidden until 011 is merged (then 2.8).

| UI audit | Functional review |
|---|---|
| **H-01** amounts jump (done in 1.2; re-checked) · **H-02** uneven columns · **H-03** summary cards differ, ⓘ position · **H-04** currency picker misaligned · **H-05** "Recent changes: Coming soon" in a prime spot · **H-06** four stacked cards on phones · **X-11**, **X-12** (Home) · **X-18** upper-case table header (Plans at a glance) · **D-01** icon tiles in dark mode · **PH-04** (Home part) | **FR·X-07** same problem worded and laid out differently · **FR·X-08** "Nothing" shown three ways (Home part: "—" instead of "$0.00 of $500K") |

**Area acceptance:** summary values at the same y (± 1 px); the two list cards the same height; amounts in
*My activities* share one right edge (± 1 px); no "Coming soon" card; at 390 px four summary cards in 2 × 2
and the first list card visible within 900 px of scroll.

**States:** Home as Priya, Home as Alex, notifications open, Home with an empty scope.

### 2.2 Plans, the plan page and the plan form (3 days)

**Pages and files:** `features/plans/*` (Plans cards and table, empty search, actions menu, copy and
delete dialogs, plan page, plan tree, new and edit plan).

**Changes:** list template with the toolbar inside the table card across its full width; Plans table on the
data table standard without repeated labels; plan cards with fixed internal rows; 4 summary cards (or the
5 → layout rule); record template for the plan page with tabs **Plan tree · Details · History**; the tree
with ≤ 2 chips per row, avatar on the plan row, "Remaining" wording aligned, Total row on one line; Details
as a `DescriptionList` including Shell attributes and Goals (numbers right-aligned, "Calculated" as a tag);
form template for new and edit plan with the right rail; "Edit plan" title with the name in the breadcrumb.

| UI audit | Functional review |
|---|---|
| **P-01** table layout breaks (High; with 1.2) · **P-02** cells repeat their column label · **P-03** card rows don't line up · **P-04** toolbar mixes three styles, stops 92 px short · **P-05** copy and delete dialogs differ (with 1.5) · **P-06** orphan card and 4 toolbar rows on phones · **PL-01** owner column (High; 1.2) · **PL-02** four stacked chips · **PL-03** "Remaining" vs "left" · **PL-04** plan row owner as plain text · **PL-05** Total row wraps · **PL-06** Shell attribute values 24 px off, goals left-aligned · **PL-07** blank block under Goals · **F-01** empty right 40 % · **F-02** help text position (1.5) · **F-03** duplicate heading (1.5) · **F-04** four-line edit header · **F-05** chip contrast (1.3) · **X-07**, **X-08**, **X-12**, **X-13**, **X-16** (Plans parts) | **FR·P-01** owner column (High; with 1.2) · **FR·P-05** stacked chips (expand memory needs §6 item 9) · **FR·A-02** "In progress" on plan summary card (after §6 item 1) · **FR·X-01** tree name cell 260 px min (with 1.2) · **FR·X-08** "Nothing" (Plans and plan page parts) |

**Area acceptance:** no tree or table row taller than 64 px on the demo plans; the 13-record plan tree fits
in one 900 px screen when expanded to campaign level; team tags never wrap; toolbar right edge equals the
card's right edge; in plan cards the owner, progress and footer rows sit at the same y across a row (± 1 px);
Details values start at one x; at 390 px no sideways scroll and the first plan visible within two toolbar
rows.

**States:** Plans cards, Plans table, empty search, actions menu, copy dialog, delete dialog, plan page (tree
collapsed, expanded, Details tab), draft AOP plan, new plan (empty, with errors), edit plan.

### 2.3 Campaign and initiative panels (1 day)

**Pages and files:** `features/campaigns/*` (details, edit, new, new with errors; initiative panel).

**Changes:** `SidePanel` action props (Edit primary, next status action secondary, ⋯ in the header with
Delete last); details in `DescriptionList`; the parent named as a value instead of a second breadcrumb;
section headings all in ink; `DateRangeField` in the person's format.

| UI audit | Functional review |
|---|---|
| **C-01** unusable on phones (High; 1.2 — re-checked) · **C-02** second breadcrumb in the panel · **C-03** grey "Goals" heading looks disabled · **C-04** US date fields (1.5) · **X-05**, **X-08** (campaign parts) | **FR·P-03** dates on forms (panel part; 1.5) · **FR·A-02** roll-up chip on campaign and initiative rows (after §6 item 1) |

**Area acceptance:** at 390 px the panel fills the screen with Close, Save and every value visible; the
footer identical (positions ± 1 px) on campaign and initiative panels; dates read "01 Jan 2026" in view and
edit.

**States:** campaign details, edit, new with errors; initiative details; panel at 390 px.

### 2.4 Activities and line items (3 days)

**Pages and files:** `features/activities/*`, `features/line-items/*`, `features/approvals/ApprovalSection*`
(activity page draft, approved, declined, approver view; activity form new, errors, edit; delete dialog;
line item page; line item new, edit, new with errors).

**Changes:** record template for the activity page with tabs **Overview · Details · Approval · History**;
the header through `PageHeader` (level tag in the meta row, 28 px title, ⋯ not "More ▾"); **one primary by
role** (Edit for the owner, Approve + Decline for the approver, no Approve button for the owner of a
submitted activity); the decline reason and the waiting notice as an `InlineAlert` under the header; the
"Ready to submit" panel only while the activity can still be submitted; one money format on the page;
line items on the data table standard; every key–value block a `DescriptionList`; the browser triangle
replaced by the shared chevron; activity and line item forms on the form template with the rail and the
needed-fields count; multi-choice Shell attributes in columns or a combobox; the line item page on the
same content width as the activity page; the line item summary strip moved into the rail.

| UI audit | Functional review |
|---|---|
| **X-04** two primary buttons · **AC-01** header differs, two primaries · **AC-02** decline reason buried · **AC-03** approver sees "Ready to submit", Decline not beside Approve · **AC-04** money in three formats · **AC-05** numbers not under headers (1.2) · **AC-06** three key–value layouts · **AC-07** browser triangle · **AF-01** 4,092 px form, long checkbox lists · **AF-02** duplicate heading (1.5) · **LI-01** 720 px page vs 1,120 px parent · **LI-02** "LINE ITEM" twice, "More ▾" · **LI-03** finance coding mixes layouts, duplicate heading · **LI-04** pre-filled new line item (keep the summary strip; product owner to confirm whether a pre-filled amount is wanted) · **X-03**, **X-08**, **X-09**, **X-13** (activity and line item parts) | **FR·P-04** too many fields before Submit (progress count and visible markings; copy-down needs §6 item 9) · **FR·P-07** amounts written two ways (after §6 item 3) · **FR·P-08** duplicate heading · **FR·P-10** "Ready to submit ✓" on a submitted activity · **FR·A-01** owner sees Approve as primary (after §6 item 2) |

**Area acceptance:** one primary in every activity state and role; the decline reason visible without
scrolling at 1440 × 900 (top edge above y = 400); one money format in the page text; line item numbers under
their headers (D6); the new activity form at least 30 % shorter than today's 4,092 px at 1440 px; the line
item page and the activity page share the content width (left and right edges ± 1 px).

**States:** activity draft (owner), submitted (owner), submitted (approver), approved, declined; activity
form new, with errors, edit; delete dialog; line item page; line item new with errors; line item edit.

### 2.5 Approvals (1 day)

**Pages and files:** `features/approvals/*` (waiting, all open, decided, empty, preview panel, decision
dialog).

**Changes:** list template; the three views as underline `Tabs` with counts; the table on the data table
standard with person and team columns that never wrap, stacked cards below 600 px; toolbar from the list
toolbar rule (no gap); the preview panel footer **Decline · Approve** (primary last); a marketer lands on
the first view that has items ("Waiting for me" when it has any, otherwise "All open").

| UI audit | Functional review |
|---|---|
| **AP-01** names wrap in a 7-column table · **AP-02** toolbar gap · **AP-03** marketer lands on an always-empty view · **AP-04** pill tabs wrap on phones · **PH-03** | **FR·A-01** Approve offered as primary to the owner (approval section and preview; after §6 item 2) |

**Area acceptance:** at 390 px the tabs stay on one line (scrolling) and each approval is a stacked card;
owner and level names on one line at 1440 px; Approve and Decline adjacent in the panel and on the page.

**States:** waiting (approver), empty (marketer), decided, preview panel, approve dialog, decline dialog with
a comment error.

### 2.6 All work (1.5 days)

**Pages and files:** `features/all-work/*` (grid, calendar, budget, expanded rows, empty search, filters,
views, export).

**Changes:** title "All work" (menu name) with the description; view switch (Grid · Calendar · Budget) as
`Tabs` directly under the header, filters after it; filter chips in a drawer below 600 px; the grid and
budget views on the data table standard with the scroll cue; activity names wrap only in the name column
and the level tag sits in its own column or before the name on one line; calendar legend from the one
status mapping, bar labels using the available width, no chip under each name; an ⓘ on "Unassigned planned
cost" saying it adds positive balances only; `Switch` and `Pager` restyled to the spec of design system §5
(toggle 40 × 24, ink when on).

| UI audit | Functional review |
|---|---|
| **AW-01** title differs from menu name · **AW-02** summary value 19 px lower · **AW-03** view switch buried under 10 chips, 6 chip rows on phones · **AW-04** row heights alternate 40 / 64 px · **AW-05** grid and budget cut mid-value without a cue · **AW-06** calendar colours, truncated labels, chips double row height · **X-14** two "In progress" colours (calendar) · **PH-04** (All work part) | **FR·X-02** totals do not visibly add up · **FR·A-02** "In progress" in the Status column (after §6 item 1) |

**Area acceptance:** the calendar legend shows each status word once, in one colour; every grid row the same
height (40 px, or 56 px with a two-line name); a visible scroll cue whenever the grid is wider than its card;
at 390 px the data starts within the first 600 px of the page.

**States:** grid, grid expanded, calendar, budget, empty search, filters open, filters drawer (390), save view
dialog.

### 2.7 Settings, people, teams and profile (2.5 days)

**Pages and files:** `features/settings/*`, `features/approval-workflows/*`, `features/people/*` (hub,
Models, Activity types, Shell attributes list and details, Currencies, Finance budget items, Approval
workflows and panel, People and panel, Teams and panel, Settings as a marketer, My profile).

**Changes:** settings-section template (§2.4 D): hub cards 4 per row; every section's breadcrumb, title,
segmented filter **Active (n) · Inactive (n) · All (n)** (default Active) and count line "n of m <things>";
tables on the data table standard with money and counts right-aligned; Finance budget items' two tables
sharing fixed column widths; workflow, person and team panels on the one panel layout (Deactivate, Set
inactive and Delete in ⋯); one set-inactive behaviour (§6.7 of the design system); person panel labels
("Owns" and "Approves in"); the view-only `InlineAlert` ready for when roles arrive; My profile titled
"My profile", equal-height cards, links styled like every other link.

| UI audit | Functional review |
|---|---|
| **ST-01** hub hole in last row · **ST-02** marketer sees the admin hub (alert ready; applies when roles land) · **ST-03** breadcrumb plus eyebrow (1.5) · **ST-04** two tables size columns separately · **ST-05** money and counts left-aligned · **ST-06** panels with own footers · **ST-07** (1.2; re-checked) · **PE-01** People toolbar differs · **PR-01** profile cards of different heights · **PR-02** "Your profile" vs "My profile", underlined link · **X-05**, **X-07**, **X-08**, **X-16** (Settings and people parts) | **FR·S-01** same action behaves differently (filters, counts, panels; set-inactive confirmation after §6 item 4) · **FR·S-02** labels in the person details panel |

**Area acceptance:** every Settings list shows the same filter order, default and count wording; Code and
Status columns of the two Finance budget item tables start at the same x (± 1 px); the three panels' footers
identical in layout; profile cards equal in height; at 390 px no Settings page scrolls sideways.

**States:** hub, each of the six sections, Shell attribute details, workflow panel, person panel, deactivate
dialog, team panel, Settings as a marketer, My profile.

### 2.8 Activity log and Recycle bin (1.5 days, after the 011 merge)

**Prerequisite:** feature 011 (`lane-011`) merged into `plan-module`, including the hand merge of
`history/service.go` and the open tasks T005, T019 (demo history seed), T032–T035 (History sections), as
listed in the functional review §6. The merge itself is not part of this estimate.

**Pages and files:** `features/activity-log/*`, `features/recycle-bin/*`, the History sections of the plan,
campaign, initiative, activity and line item pages, Home's *Recent changes*, `features/placeholders/*`.

**Changes:** the 011 pages (built before version 2) on the list template and data table standard; restore,
restore-with-parents, delete-permanently and empty-bin dialogs on the one dialog layout (red confirm only
for permanent deletion); History as the last tab of each record page; *Recent changes* on Home as a card of
the dashboard; "View history" opening the record's History tab; the delete dialog and Undo toast promising
the recycle bin now that it can be opened; Not found and any remaining Coming soon page on one centred
layout.

| UI audit | Functional review |
|---|---|
| **A-01** sidebar items and the delete dialog lead to "Coming soon" · **H-05** (with 2.1) · **M-02** Coming soon in a card, Not found without | **FR·P-02** deleted records can't be brought back after Undo (High) · **FR·X-03** View history, Activity log and Recent changes lead to "Coming soon" · **FR·X-04** demo data has no log or bin content (fixed by 011 T019, owned by 011) |

**Area acceptance:** no menu item, link or message leads to "Coming soon"; a record deleted in the demo can be
restored from the Recycle bin; the activity log shows the seeded history; D1–D10 on the new pages.

**States:** activity log (filtered and empty), record History tab, recycle bin (with items and empty),
restore dialog, restore-with-parents dialog, delete-permanently dialog, Home with Recent changes, not found.

---

## 3. Order, effort and dependencies

| Step | Effort (dev days) | Depends on | Can run in parallel with |
|---|---|---|---|
| 1.1 Test harness | 2 | – | 1.2 |
| 1.2 Global fixes | 1 | – | 1.1 |
| 1.3 Tokens, dark mode, fonts | 1.5 | 1.1 | – |
| 1.4 Scales and lint guard | 3 | 1.3 | 1.5 (different files after the token swap) |
| 1.5 Shared components | 5 | 1.3 | 1.4, 1.6 |
| 1.6 App shell | 2.5 | 1.3 | 1.5 |
| 1.7 Documents and decisions | 0.5 | – | any |
| **Phase 1** | **≈ 15.5** | | |
| 2.1 Home | 1.5 | phase 1 | 2.2–2.7 |
| 2.2 Plans, plan page, plan form | 3 | phase 1 | 2.1, 2.3–2.7 |
| 2.3 Campaign and initiative panels | 1 | phase 1 | others |
| 2.4 Activities and line items | 3 | phase 1 | others |
| 2.5 Approvals | 1 | phase 1 | others |
| 2.6 All work | 1.5 | phase 1 | others |
| 2.7 Settings, people, teams, profile | 2.5 | phase 1 | others |
| 2.8 Activity log and Recycle bin | 1.5 | phase 1 and the 011 merge | others |
| **Phase 2** | **≈ 15** | | |

With two developers: about two weeks for phase 1 and one and a half for phase 2, plus product-owner review
time for the screenshots.

**Risks and how they are handled**

| Risk | Handling |
|---|---|
| Shell says its internal design system ("Shell DS") is mandatory | Everything visual is in tokens and shared components; swap values and components, keep the pages |
| Feature 011 merge conflicts with restyled components | Merge 011 before area 2.8; the foundation owner reviews the 011 pages against version 2 |
| Screenshot tests are brittle | Self-hosted fonts, fixed clock, masked times, one operating system for baselines, small tolerance (`maxDiffPixelRatio` 0.01) |
| Design changes hide behaviour changes | Behaviour changes are listed (§6) and covered by the existing journeys and new unit tests |
| The custom date input is less accessible than the browser's | Typing stays the main way; the calendar follows the WAI-ARIA date picker dialog pattern; keyboard and screen-reader checks in 1.5 |
| Phase 2 areas drift apart again | Shared components frozen during phase 2; stylelint; the §11 checklist in every review |

---

## 4. Test approach

### 4.1 Unit and component tests (Vitest, Testing Library)

- **Tokens:** a contrast test that reads `tokens.css` and checks every pair of design system §3.1 and §4 in
  both themes (text ≥ 4.5, borders, icons and focus ≥ 3).
- **New components:** `Tabs` (roles, arrow keys, selected state, no wrap), data table standard (column types,
  alignment classes, totals, scroll region named when it overflows), `DescriptionList`, `InlineAlert`
  (roles), `TopBar`, `PageHeader` (slots, at most one primary, no eyebrow), `SidePanel` action props and
  footer order, `DateRangeField` (display and parsing in all four date formats, calendar keyboard, min/max).
- **Rules:** the status-word mapping (every word maps to exactly one family); money, percent and date
  formatters (one format per screen, "—", "38%", 24-hour); a source scan that no feature renders `<h1>`
  outside `PageHeader` and no date is formatted outside `lib/format.ts`.
- Existing tests stay; text expectations change only where the wording rules change them.

### 4.2 Lint (stylelint, ESLint)

stylelint enforces design system §3.12 and §11 rules 7, 8, 10 and 12 on every CSS module, inside
`npm run check`.

### 4.3 End-to-end (Playwright)

- **Visual suite** (`e2e/visual.spec.ts`): the states listed per area plus the app-shell states of 1.6
  (about 75) in four projects (1440 × 900 and 390 × 844, light and dark) — about 300 screenshots — against
  reviewed baselines; viewport shots for panels, menus and dialogs.
- **Layout assertions** on every visual state: D3–D7 and the area acceptance criteria (title position, one
  primary, numbers under headers, minimum text and target size, row heights, card heights, repeated
  headings).
- **Accessibility:** axe with `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa`, `wcag22aa` on every visual state, both
  themes, both widths.
- **Keyboard journeys:** the standard CRUD flow (create a campaign in a panel, edit, delete, Undo), submit and
  approve an activity, and the drawer on a phone, with the keyboard alone, checking focus visibility and that
  no sticky element covers the focused control.
- **Existing journeys** (`smoke`, `core-flow`, `module-rest`) run unchanged except for selectors and texts the
  revamp renames.
- **Data and servers:** a fresh seed in a temporary data folder (`go run ./cmd/seed -data <temp> -reset`), the
  API and Vite on `127.0.0.1` with a free port pair (not 8080, 5173, 18080, 14173, 18111 or 14111), stopped
  afterwards.

### 4.4 Manual checks (per area, recorded in the change description)

- 200 % zoom at 1280 px and 400 % zoom (reflow at 320 CSS px) without lost content or sideways scrolling.
- Windows high-contrast (forced colours): borders, focus and the current-place marker stay visible.
- Screen reader smoke test (NVDA with Edge or Chrome) on create, edit, delete with Undo and approve.
- A real phone or device emulation for touch: targets, drawer, panels, no zoom on input focus.
- Product-owner review of before/after screenshots for the area.

---

## 5. Out of scope for this plan

- Shell's own assets (typeface, pecten, official colours, Shell DS): they switch in later through one token
  or component each (design system §12; recommendations §7).
- New features: bulk actions (012), BCPIR and SP&A views (013), roles and permissions, global search (unless
  the product owner adds it, §6 item 7).
- Functional review items that are not UI design (appendix B).

---

## 6. Decisions needed

| # | Decision | Needed before | Recommendation |
|---|---|---|---|
| 1 | Rename the approval roll-up "In progress" to "Partly approved" / "n of m approved" (FR·A-02) | 1.5 status mapping; closes in 2.2, 2.3, 2.6 | Yes |
| 2 | No Approve button for the owner of a submitted activity; keep it in ⋯ with the warning until the client rules on self-approval (FR·A-01) | 2.4, 2.5 | Yes |
| 3 | One money format per screen, refining 007-P4 (FR·P-07, UI audit AC-04) | 1.5 `lib/format.ts`; 2.4 | Yes |
| 4 | Set inactive: confirm when in use, Undo otherwise (FR·S-01) | 2.7 | Yes |
| 5 | Record pages with tabs (plan and activity) | 2.2, 2.4 | Yes |
| 6 | Merge 011 before the client demo, or hide its menu items and the restore promise until then (UI audit A-01, FR·P-02) | 2.8, or an interim change in 1.6 | Merge first |
| 7 | Global search in the top bar opening All work | Optional, after 1.6 | Optional |
| 8 | Extend D-040 with the visual and accessibility suite | 1.1 | Yes |
| 9 | Behaviour items not decided by design: remember the tree's expand state (FR·P-05), confirm before publishing an empty plan (FR·P-06), copy Shell attributes from the parent (FR·P-04), pre-filled new line item (UI audit LI-04) | Each when decided | Decide separately |
| 10 | Constitution principle III amendment (yellow primary, WCAG 2.2 AA) | 1.7 | Yes |

---

## Appendix A — UI audit traceability (97 entries)

| Step | UI audit entries closed |
|---|---|
| 1.2 Global fixes | X-01 (High), X-02 (High), X-17, H-01 (High), PL-01 (High), P-01 (High, with 2.2), C-01 (High), ST-07 (High), PH-01 (High), PH-02 (High), AC-05 |
| 1.3 Tokens, dark mode, fonts | S-02 (with 1.6), D-02, F-05 |
| 1.4 Scales and lint guard | X-10, AX-01 |
| 1.5 Shared components | X-03 (component; pages in phase 2), X-05 (component), X-06, X-07, X-08 (component), X-09, X-11, X-14 (mapping), X-15, X-18, X-19, F-02, F-03, AF-02, P-05, PH-05, PH-06, AX-02, AX-03 |
| 1.6 App shell | A-02, A-03, S-01, M-01, D-03, X-13 (shell part) |
| 2.1 Home | H-02, H-03, H-04, H-05 (with 2.8), H-06, X-12, D-01, PH-04 |
| 2.2 Plans, plan page, plan form | P-02, P-03, P-04, P-06, PL-02, PL-03, PL-04, PL-05, PL-06, PL-07, F-01, F-04, X-16 |
| 2.3 Campaign and initiative panels | C-02, C-03, C-04 |
| 2.4 Activities and line items | X-04, AC-01, AC-02, AC-03, AC-04, AC-06, AC-07, AF-01, LI-01, LI-02, LI-03, LI-04 |
| 2.5 Approvals | AP-01, AP-02, AP-03, AP-04, PH-03 |
| 2.6 All work | AW-01, AW-02, AW-03, AW-04, AW-05, AW-06 |
| 2.7 Settings, people, teams, profile | ST-01, ST-02, ST-03, ST-04, ST-05, ST-06, PE-01, PR-01, PR-02 |
| 2.8 Activity log and Recycle bin | A-01, M-02 |

Count: 11 + 3 + 2 + 19 + 6 + 8 + 13 + 3 + 12 + 5 + 6 + 9 + 2 = **99 IDs, each listed once**: the audit's 97
counted entries plus M-01 (no problem found; kept tidy in 1.6) and D-03 (dark mode repeats the light-mode
problems; counted there by the audit). Each ID sits under the step that closes it; P-01 and S-02 are finished
by a second step named in brackets. Entries that a phase 1 step builds and a phase 2 area finishes on its
pages (X-03, X-05, X-07, X-08, X-12, X-13, X-16) are verified again in each area that names them.

## Appendix B — functional review traceability (25 findings)

| Finding | Severity | Where | Status |
|---|---|---|---|
| FR·P-01 owner column one letter per line | High | 1.2, 2.2 | Closed by this plan |
| FR·P-02 delete promises a recycle bin that can't be opened | High | 2.8 (with the 011 merge) | Closed by this plan with 011 |
| FR·P-03 dates on forms ignore the date format | Medium | 1.5 | Closed by this plan |
| FR·P-04 too many fields before Submit | Medium | 1.5, 2.4 | **Part**: visible markings and a progress count; copy-down is a product decision (§6 item 9) |
| FR·P-05 activity rows hidden when the plan page opens | Medium | 1.5, 2.2 | **Part**: ≤ 2 chips per row; remembering the expand state is a product decision (§6 item 9) |
| FR·P-06 a published plan can be empty | Medium | – | Outside: product decision (§6 item 9); `ConfirmDialog` ready if chosen |
| FR·P-07 amounts written two ways | Medium | 1.5, 2.4 | After §6 item 3 |
| FR·P-08 "Shell attributes" heading twice | Low | 1.5, 2.4 | Closed by this plan |
| FR·P-09 toasts cover the Save bar | Low | 1.5 | Closed by this plan |
| FR·P-10 "Ready to submit ✓" on a submitted activity | Low | 2.4 | Closed by this plan |
| FR·A-01 submitter can approve from a primary button | High | 2.4, 2.5 | After §6 item 2; the client still rules on self-approval |
| FR·A-02 "In progress" means two things | Medium | 1.5, 2.2, 2.3, 2.6 | After §6 item 1 |
| FR·A-03 approval notices stay unread after the decision | Low | – | Outside: backend (`approvals/rounds.go`), feature 008 owner |
| FR·A-04 no bulk approve | Low | – | Outside: feature 012 (deferred) |
| FR·S-01 the same action behaves differently in Settings | Medium | 2.7 | Filters, counts and panels closed; set-inactive after §6 item 4 |
| FR·S-02 labels in the person details panel | Low | 2.7 | Closed by this plan |
| FR·X-01 phone screens scroll sideways | Medium | 1.2, 2.2 | Closed by this plan |
| FR·X-02 All work totals do not visibly add up | Medium | 2.6 | Closed by this plan (ⓘ explanation) |
| FR·X-03 View history, Activity log and Recent changes lead to "Coming soon" | Medium | 2.8 | Closed by this plan with 011 |
| FR·X-04 demo data has no log or bin content | Medium | – | Outside: 011 task T019 (seed); area 2.8 acceptance depends on it |
| FR·X-05 task checkboxes do not show what is built | Medium | – | Outside: `tasks.md` and feature map upkeep |
| FR·X-06 the contract lists endpoints that answer 404 | Low | – | Outside: API documentation |
| FR·X-07 Home wording and layout of the same problem | Low | 2.1 | Closed by this plan |
| FR·X-08 "Nothing" shown three ways | Low | 1.5, 2.1, 2.2 | Closed by this plan |
| FR·X-09 sign-in logs a 401 in the console | Low | – | Outside: `SessionProvider` (app code, not design) |

**18 addressed** (FR·P-01, P-02, P-03, P-04 part, P-05 part, P-07, P-08, P-09, P-10, A-01, A-02, S-01, S-02,
X-01, X-02, X-03, X-07, X-08) · **7 outside** (FR·P-06, A-03, A-04, X-04, X-05, X-06, X-09).
