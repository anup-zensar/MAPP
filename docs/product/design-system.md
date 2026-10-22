# Design System — MAPP (version 2)

| | |
|---|---|
| **Version** | 2.0 · 20 October 2026 |
| **Status** | Binding for every screen of the web app. Code that disagrees with this document is a defect. |
| **Replaces** | Version 1 (Apple-inspired look: Apple blue `#0071e3`, blue-to-purple gradient, system font stack, pill shapes) |
| **Decided by** | Product owner, 20 October 2026: Shell yellow primary button with dark text, at most one per page or dialog (D-310); light-grey canvas with white cards (D-311); Jost headings and Arial body, no Shell logo (D-312); WCAG 2.2 AA (D-313); red only for danger and errors (D-314) — decision log D-310–D-314 |
| **Evidence** | [Shell brand research](../reports/ux-research-shell-brand.md) · [Shell UI patterns research](../reports/ux-research-shell-ui-patterns.md) · [UI audit](../reports/ui-audit.md) · [Functional review](../reports/functional-review.md) · [Recommendations](../reports/ux-design-recommendations.md) · [Implementation plan](../reports/ui-ux-implementation-plan.md) |
| **Code** | Tokens: `frontend/src/styles/tokens.css` · base styles: `frontend/src/styles/global.css` · shared components: `frontend/src/components/` · app shell: `frontend/src/app/` |

MAPP looks like a **Shell enterprise tool**: calm, structured, symmetric and ready for daily
production use. It follows Shell's current web design language (grey ink text, white cards on a light
grey canvas, Shell yellow only where you act and where you are, red only for danger and errors) and
applies it to a working application. Version 1's layout and interaction patterns stay (left sidebar,
summary cards, side panels, the standard CRUD pattern). Shell's Hive9 is the *functional* reference only:
its screens are never copied (D-005, functional part).

**What changed from version 1:** colours, fonts, type scale, spacing scale, control heights, radii,
shadows, breakpoints and page width are new tokens (§3); the app shell gains a white top bar and a white
sidebar with the yellow "you are here" bar (§2.1); there is one page-header layout and five page templates
(§2.3, §2.4); pill tabs become underline tabs; every shared component has a measured spec (§5); the
accessibility target is WCAG 2.2 AA (§9); and the consistency rules are written down as a checklist
(§11). The behaviour rules of §6–§8 and the usability targets of §10 are unchanged except where noted.

Section numbers §1–§10 are kept from version 1, so references in specs and code stay valid.

---

## 1. Principles

1. **A beginner understands every screen.** One purpose per page, plain words, nothing unexplained.
2. **One way to do each thing.** The same list, form, details, delete and status patterns everywhere.
3. **Status and next step are always visible.** Every record shows where it is (status chip) and what is
   missing (completeness), with a button for the next action.
4. **Show, don't hide.** Unavailable actions stay visible and explain why; read-only values say where they
   come from.
5. **Safe.** Confirm destructive actions, protect unsaved changes, offer Undo, keep a recycle bin.
6. **Calm.** White cards on a light grey canvas, grey ink text, one accent (Shell yellow) used only where
   you act and where you are, colour otherwise used only for meaning (status, level, money problems).
7. **Looks like Shell, behaves like MAPP.** Shell's colours, type pairing, grid, radii and wayfinding;
   MAPP's own name mark, plain words and CRUD pattern. No Shell logo, typeface or trade dress until
   Shell grants them (§3.10, §12).
8. **One of everything.** One ink colour, one accent, one height per control size, one radius per kind of
   object, one spacing scale, one page header, one panel footer, one table style, one key–value layout,
   one date format. Inconsistency reads as "unfinished"; the UI audit traced most of its 97 findings to
   things built several ways.
9. **Symmetric and aligned.** Every page shares one left edge and one right edge; controls side by side
   have the same height; cards in a row have the same height; numbers line up under their headers.

---

## 2. Layout: app shell and page templates

### 2.1 App shell

```
┌──────────────────────────────────────────────────────────────────────────────────────────────┐
│ Demo environment — sign-in and data are for demonstration only            (banner 32, navy)  │
├──────────────────────────────────────────────────────────────────────────────────────────────┤
│ [★] MAPP          (search slot, reserved)                       🔔3   (AB) Priya ▾      │ top bar 56, white
├────────────────────┬─────────────────────────────────────────────────────────────────────────┤
│▌⌂ Home             │   Plans › FY2026 Lubricants Europe                     (breadcrumb 12)  │
│ ▤ Plans            │   Helix launch                           [⋯] [Submit for approval] [Edit]│ title 28 Jost
│ ✓ Approvals    (4) │   Campaign · Draft · 3 fields needed · C-1003 · Priya Raman · Feb–Nov   │ meta row
│ ▦ All work         │                                                                         │
│                    │   ┌────────────┐ ┌────────────┐ ┌────────────┐ ┌────────────┐           │ summary cards
│ More               │   └────────────┘ └────────────┘ └────────────┘ └────────────┘           │
│ ⚙ Settings         │   Overview   Initiatives (3)   Details   History                         │ underline tabs
│ ↺ Recycle bin      │   ┌──────────────────────────────────────────────────────────────────┐  │
│ ☰ Activity log     │   │ white card: table, 40 px rows, money right-aligned                │  │
│                    │   └──────────────────────────────────────────────────────────────────┘  │
│ « Collapse sidebar │   MAPP · demo data                                     (footer 12) │
└────────────────────┴─────────────────────────────────────────────────────────────────────────┘
 sidebar 240, white          content: #F5F5F5 canvas, column ≤ 1300 px centred, 24 px padding
 ▌ = 4 px yellow bar + bold label + pale-yellow fill on the current section
```

| Element | Rule |
|---|---|
| **Skip link** | First focusable element: "Skip to main content". Hidden until focused, then shown top-left: `--brand-yellow` fill, `--on-brand-yellow` text, 14 px bold, padding 16, radius 8. |
| **Environment banner** | `DemoBanner`. 32 px, full width, Shell system navy `--banner-bg` with white 12 px text, one line, centred. Below 600 px the shorter text "Demo environment — demonstration data only". Not sticky. |
| **Top bar** | 56 px, `--surface`, 1 px `--hairline` bottom border, full width, sticky at the top. **Left:** at ≥ 960 px the logo zone is 240 px wide, aligned with the sidebar (`Logo` + "MAPP"); below 960 px a 40 px **Open menu** icon button, then the logo mark. **Centre-left:** the global search slot (360–480 px), empty until a search feature is specified; never a fake control. **Right**, 8 px apart: notifications bell (40 px icon button with `CountBadge`), the account button (32 px avatar + first name at ≥ 960 px; avatar only below). The account menu holds the name and role, **My profile**, **Appearance** (Follow the computer · Light · Dark; D-038) and **Sign out**. A help button joins the right group, always in the same place (WCAG 3.2.6), when a help feature exists. |
| **Sidebar** | 240 px, `--surface`, 1 px `--hairline` right border, sticky under the top bar, scrolls on its own. Items 40 px high, padding 0 16 0 20, 20 px icon + 12 px gap + 14 px label in `--ink`. Hover `--hover`, no movement. **Current section: a 4 px `--brand-yellow` bar on the start edge (full item height), `--brand-yellow-tint` fill, bold label, `aria-current="page"`.** Group label "More" 12 px bold `--ink-2`, sentence case, 24 px above and 8 px below. Count badges sit at the end of the item. At the bottom: **Collapse sidebar** (40 px, ghost). |
| **Collapsed rail** | 64 px; icons centred; tooltips on the right; the yellow bar stays on the current item; badges become a small count at the icon's top end. Chosen by the person (saved, D-038) at ≥ 960 px; automatic between 600 and 959 px. |
| **Drawer** | Below 600 px the sidebar is hidden. **Open menu** opens a 280 px drawer from the left with `--overlay`, focus kept inside, Escape and the close button close it and return the focus. It shows the same items with labels, no tooltips. Bell and account stay in the top bar. |
| **Content area** | `--page-bg` (#F5F5F5) canvas. One column, at most `--content-max` (1300 px), centred in the space right of the sidebar, padding `--gutter` (24 px; `--gutter-sm` 16 px below 600 px), 24 px from the top bar to the breadcrumb row, 48 px bottom padding. When the sidebar collapses, the column grows (up to 1300 px). |
| **Footer** | Inside the content column, after the page: 1 px `--hairline` top border, 16 px padding, one 12 px `--ink-2` row: "MAPP · demo data" (this phase). No marketing footer. |

Primary navigation for this phase: **Home · Plans · Approvals · All work** and, under **More**,
**Settings · Recycle bin · Activity log**. (Finance and Measure appear later.) My profile is reached from
the account menu.

**Layers (z-index):** sticky table header 10 · sticky form footer 50 · sidebar 90 · top bar 100 ·
overlay 200 · side panel and dialog 210 · menus and popovers 300 · toasts 400 · tooltips 500 · skip link 1000.

### 2.2 Grid, page widths and breakpoints

| Rule | Value |
|---|---|
| Content column | Fluid up to `--content-max` 1300 px, centred; side padding 24 px (16 px below 600 px) |
| Grid | 12 columns, `--grid-gap` 20 px between columns and between cards |
| Common spans | Summary cards: 4 per row (3 columns each), or 3 per row (4 each); details: 8 + 4 at ≥ 1280 px; two lists side by side: 6 + 6 |
| Forms and reading text | One column of at most `--form-max` 720 px, aligned to the **left edge** of the content column (never centred), with an optional right rail (300 px) at ≥ 1280 px |
| Breakpoints | **< 600** phone · **600–959** tablet · **≥ 960** desktop · one extra **≥ 1280** for the 8 + 4 split and the form rail. No other widths may appear in a media query (§3.12). |
| Alignment | Logo zone, breadcrumb, title, toolbars, cards, tables and footer start on one left edge; page actions, toolbar ends and footer buttons end on one right edge. Every control in a row has the same height, so text baselines line up. |
| Equal heights | Cards in one grid row stretch to the same height. A list inside a card shows at most 5 rows and a **View all (n)** link, so neighbouring cards stay even. |

### 2.3 Page header (the only one)

Every signed-in page uses `PageHeader`; features never build their own header (§11, rule 3).

```
 Plans › FY2026 Lubricants Europe                                     ← breadcrumb row, 18 px, always reserved
 Helix launch                                     [⋯]  [Secondary]  [Primary]   ← title 28/36 Jost 700; actions
 Campaign · ▣ Draft · ▣ 3 fields needed · C-1003 · Priya Raman · 01 Feb – 30 Nov 2026   ← meta row (record pages)
 Every plan, campaign and activity in one place.                      ← description (list pages, optional)
```

| Part | Spec |
|---|---|
| Breadcrumb row | Always present and always one 12/18 line high, followed by 8 px, so the title sits at the **same position on every page** (desktop with the banner: title top at 138 px from the window top = banner 32 + top bar 56 + 24 + 18 + 8). Empty on top-level pages (Home, Plans, Approvals, All work, Settings). |
| Title | `<h1>`, the only one on the page; Jost 700, 28/36 (20/28 below 600 px); `--ink`; wraps to at most 2 lines. It is the menu name on top-level pages ("All work", "My profile"), the record's name on record pages, and "New activity" / "Edit activity" on form pages (the record's name is then in the breadcrumb). No eyebrow, no grey "highlight" phrase, no upper case. Home is the only exception: its title greets ("Good morning, Priya") in 32/40. |
| Actions | Right-aligned on the title's first line (vertically centred on it). Order from left to right: **⋯ More** (icon button), secondary buttons, **the primary button last**. At most one primary (§11, rule 1). Below 600 px the actions move under the title; only the primary and ⋯ stay, secondary actions move into ⋯. |
| Meta row | Record pages only, 8 px under the title, items 8 px apart, 14 px `--ink-2`, single line that wraps: level tag, status chip, at most one completeness or problem chip, ID, owner, dates (one date format, §4.5). |
| Description | List and settings pages only, optional, one sentence, 14/20 `--ink-2`, at most 720 px, 4 px under the title. |
| Spacing | 24 px from the header to the content below it. |

### 2.4 Page templates

Every page is one of these five templates.

**A. List page** (Plans, Settings lists, People, Teams, Approvals)

```
 (breadcrumb row)
 Plans                                                                    [+ New plan]
 Every plan of the year in one place.
 ┌ summary card ┐ ┌ summary card ┐ ┌ summary card ┐ ┌ summary card ┐       ← optional, ≤ 4
 ┌───────────────────────────────────────────────────────────────────────────────────┐
 │ [🔍 Search name or ID ] [All (6)|Draft (2)|Published (4)] [Filters (1)]  4 of 6 plans │ ← toolbar 40 px
 │ (Year: 2026 ×) Clear filters                                                        │ ← filter chips
 │ Name ▲            Owner          Teams              Status      Planned cost    ⋯   │ ← table, §5.4
 └───────────────────────────────────────────────────────────────────────────────────┘
```

- The toolbar sits **inside** the table card (16 px 24 px padding) and spans its full width: search,
  segmented filter, **Filters** button, then — pushed to the right end — the result count ("4 of 6
  plans"), view switch and **Export** where the page has them. Filter chips go on a second row only while
  filters are in use.
- Empty list: `EmptyState` inside the card; filters hiding everything: "Nothing matches these filters" with
  **Clear filters**.
- Below 600 px: search takes the first toolbar row; inbox-type lists (Approvals, Home lists) become stacked
  cards; other tables scroll sideways inside their card with a scroll cue.

**B. Record page** (plan, activity, line item, Shell attribute)

- Page header with breadcrumb, meta row and actions (§6.3 decides which action is primary).
- An `InlineAlert` directly under the header when the record waits or was declined ("Declined by Daniel
  Okoro on 12 Mar 2026: The planned cost is above this quarter's budget…") — never buried further down.
- Up to 4 summary cards (money and progress facts).
- **Underline tabs** when the record has three or more content areas; the first tab is the working view:
  plan → *Plan tree · Details · History*; activity → *Overview · Details · Approval · History*. The chosen
  tab is kept in the address (`?tab=details`). Records with fewer areas stack their cards in one column.
- At ≥ 1280 px a details area may split 8 + 4 (main content + a summary card); cards in the split stretch
  to equal height.

**C. Form page** (new and edit plan, activity, line item)

```
 Plans › FY2026 Lubricants Europe › Edit
 Edit plan
 ┌ 2 problems need your attention ┐                            ┌ On this page ───────┐
 ┌ Basics ────────────────────────┐  (form column ≤ 720 px)    │ Basics          ✓   │ ← rail 300 px,
 │ fields …                       │                            │ Dates & cost   2 ▸  │   sticky, ≥ 1280
 └────────────────────────────────┘                            │ Shell attributes 6 ▸│
 ┌ Dates & cost ──────────────────┐                            │ 9 of 15 needed      │
 …                                                             └─────────────────────┘
 ────────────────────────────────────────────────── [Cancel] [Save and add another] [Save]  ← sticky footer
```

- One `FormSection` card per section, 20 px apart, in the order of §7.
- The right rail (≥ 1280 px) lists the sections as links with how many needed fields each still lacks, and
  the overall count ("9 of 15 needed fields filled"); the line item form shows its summary strip there
  (planned cost · unassigned now · after saving). Below 1280 px the overall count shows as one line under
  the header.
- `FormFooter` is sticky at the bottom of the window across the full content column; buttons are
  right-aligned on the same right edge as the page actions.

**D. Settings section** (Settings hub and its sections)

- Hub: link cards, 4 per row at ≥ 1280 px, 2 per row from 600 px, 1 below (8 sections fill two full rows).
- A section is a list page (template A) with breadcrumb "Settings › Models", title "Models", segmented
  filter **Active (n) · Inactive (n) · All (n)** (default Active) and the count line "4 of 5 models".
- Short records open in a side panel; long ones (Shell attributes) on a record page (§6.2).
- When someone may only view a section, an `InlineAlert` (info) under the header says so; edit actions stay
  visible and explain why they are unavailable (§6.6).

**E. Dashboard** (Home)

- Header: the greeting title; the display-currency select and the My teams / Everything switch in the
  actions slot (no primary button: Home has no create action).
- A row of 4 link summary cards (2 × 2 below 600 px).
- Two equal columns (6 + 6) at ≥ 960 px: *Needs your attention* and *Waiting for my approval*, each at most
  5 rows plus **View all (n)**; then full-width *Plans at a glance* (table) and *My activities* (list);
  *Recent changes* only once the activity log exists.

### 2.5 Navigation and flows

| Situation | Rule |
|---|---|
| Where a record opens | **Side panel** for short records without child tables: campaign, initiative, Settings items, person, team, approval workflow, approval preview. **Full page** for records with children or long forms: plan, activity, line item, Shell attribute. |
| Breadcrumb | On every page below the top level, from the section to the current page. A side panel never repeats the page's breadcrumb; it names its parent as a value ("Plan: FY2026 Lubricants Europe"). Below 600 px only "‹ Parent" is shown. |
| After create | Toast "Initiative 'Digital & social' created — View"; the new row is highlighted (`--brand-yellow-tint` and a 4 px yellow start bar) and scrolled into view. |
| After delete | Back to the parent; toast with **Undo** (10 seconds). |
| Leaving with unsaved changes | "Discard changes?" with **Keep editing** (focused) and **Discard changes**. |
| Approver's flow | Approvals → preview panel → **Approve** (primary) and **Decline** side by side → next item. |
| Menus with children | The first item goes to the parent itself ("Go to: Settings"). |

---

## 3. Visual tokens

All values live in `frontend/src/styles/tokens.css`; components use `var(--token)` only (§3.12). Light
values sit on `:root`, dark values on `:root[data-theme='dark']` (set by `theme.ts`, D-038).

### 3.1 Colour

Contrast ratios use the WCAG 2.x formula. AA needs **4.5:1** for text, **3:1** for large text (≥ 24 px,
or ≥ 18.66 px bold), icons, control borders and focus indicators. In the light columns the two ratios are
on white and on grey (#F5F5F5); in the dark columns on `--page-bg` and on `--surface`.

**Surfaces, text and lines**

| Token | Light | Contrast | Dark | Contrast | Use |
|---|---|---|---|---|---|
| `--page-bg` | `#F5F5F5` | – | `#14191F` | – | App canvas behind cards |
| `--surface` | `#FFFFFF` | – | `#1D232A` | – | Cards, panels, dialogs, menus, top bar, sidebar |
| `--gray-bg` | `#F5F5F5` | – | `#252C34` | – | Fills inside surfaces: table header, read-only field, totals row |
| `--ink` | `#404040` | 10.37 · 9.51 | `#E0E0E0` | 13.38 · 12.00 | All text, headings and icons |
| `--ink-2` | `#595959` | 7.00 · 6.42 | `#B0B8C1` | 8.81 · 7.90 | Secondary text: key–value labels, descriptions, meta |
| `--muted` | `#6E6E6E` | 5.10 · 4.68 | `#949FAB` | 6.57 · 5.89 | Placeholders, hints, timestamps, disabled text; safe for small text |
| `--control-border` | `#8C8C8C` | 3.36 · 3.08 | `#76818D` | 4.45 · 3.99 | Borders of inputs, selects, checkboxes, segmented filters, outlined buttons (WCAG 1.4.11) |
| `--control-border-hover` | `#404040` | 10.37 | `#B0B8C1` | 7.90 | Border of a hovered control |
| `--card-border` | `#E0E0E0` | decoration | `#2F3741` | decoration | The edge of a white card on the grey page (also visible in forced-colours mode) |
| `--hairline` | `#D9D9D9` | decoration | `#38414B` | decoration | Dividers, top bar and sidebar borders, table outline |
| `--row-divider` | `#E8E8E8` | decoration | `#2C343C` | decoration | Lines between table rows |
| `--hover` | `rgba(64,64,64,.06)` | ink on it 9.43 | `rgba(255,255,255,.07)` | ink on it 9.78 | Hovered rows, items, ghost and secondary buttons |
| `--chip` | `rgba(64,64,64,.08)` | ink on it 9.10 | `rgba(255,255,255,.09)` | ink on it 9.12 | Neutral chips, filter chips, team tags, progress track, skeleton; pressed buttons |

**Brand and interaction**

| Token | Light | Contrast | Dark | Contrast | Use |
|---|---|---|---|---|---|
| `--brand-yellow` | `#FFC800` (Shell yellow) | 1.55 on white: **fill only** | `#D6A100` | – | The primary button, the 4 px "you are here" bar, the skip link |
| `--brand-yellow-hover` / `-pressed` | `#F2B800` / `#E0A800` | ink on them 5.74 / 4.82 | `#FFC800` / `#B88A00` | dark text 11.37 / 5.61 | Primary hover and pressed |
| `--on-brand-yellow` | `#404040` | **6.67 on yellow** | `#14191F` | **7.54 on dark yellow** | Text and icons on yellow (never white) |
| `--brand-yellow-tint` | `#FFF4CC` | ink 9.41 · link 9.52 | `#3B3723` | ink 9.06 · link 5.57 | Current sidebar item, highlighted row (just created, moved or copied) |
| `--brand-mark` | `#092949` (Shell system navy) | white on it 14.74 | `#0D3A66` | white 11.58 | The MAPP mark's tile, avatars |
| `--button-primary-bg`, `-bg-hover`, `-bg-pressed`, `-text` | aliases of the four yellow tokens | – | aliases | – | The primary button |
| `--focus-ring` | `#092949` (navy) | 14.74 · 13.52; 9.48 against yellow | `#FFC800` | 11.37 · 10.19 | Keyboard focus ring on every control |
| `--blue` | `#003C88` (Shell dark blue) | 10.49 · 9.62 | `#8CB4E6` | 8.23 · 7.38 | Checked checkboxes and switches |
| `--link` | `#003C88` | 10.49 · 9.62 | `#8CB4E6` | 8.23 · 7.38 | Links; underlined inside sentences (WCAG 1.4.1) |
| `--danger-bg` / `-hover` | `#DD1D21` (Shell red) / `#B8171B` | white text 4.92 / 6.61 | `#C42B2F` / `#A82326` | white text 5.62 / 7.16 | The destructive confirm button only |
| `--on-accent` | `#FFFFFF` | – | `#FFFFFF` | – | Text and icons on navy, dark blue and red |
| `--banner-bg` / `--banner-text` | `#092949` / `#FFFFFF` | 14.74 | same | 14.74 | Environment banner |
| `--overlay` | `rgba(0,0,0,.45)` | – | `rgba(0,0,0,.6)` | – | Behind dialogs, side panels and the drawer |
| `--tooltip-bg` / `--tooltip-text` | `#262626` / `#FFFFFF` | 15.13 | `#3A434D` / `#F2F2F2` | 8.98 | Tooltip bubble |

**Feedback colours** (one meaning each; the status families of §4.1 use them)

| Token pair (colour / `-text`) | Meaning | Light | Text on white · grey · its 8 % tint | Dark | Text on page · surface · its 12 % tint |
|---|---|---|---|---|---|
| `--info` / `--info-text` | Information, "running" | `#336094` / `#2B5482` | 7.80 · 7.15 · 6.94 | `#6F9BD1` / `#8FB4E3` | 8.25 · 7.39 · 6.15 |
| `--success` / `--success-text` | Done well, approved | `#008557` / `#00704A` | 6.15 · 5.64 · 5.52 | `#3DBB8C` / `#4CC79A` | 8.37 · 7.50 · 6.11 |
| `--warning` / `--warning-text` | Waits for someone, needs a look (amber, never brand yellow) | `#C77700` / `#8F5400` | 6.11 · 5.61 · 5.60 | `#E09A2A` / `#F0A640` | 8.61 · 7.72 · 6.32 |
| `--danger` / `--danger-text` | Errors and danger only | `#DD1D21` / `#B8171B` | 6.61 · 6.06 · 5.82 | `#FF5C5F` / `#FF7B7E` | 7.06 · 6.32 · 5.44 |

The colours themselves (dots, bars, icons) are ≥ 3:1 on white and grey; ink text on every tint is ≥ 9:1.

**Retired tokens:** `--brand-gradient` and `--brand-gradient-strong` (the logo, avatars and progress bars use
solid colours). `--blue` keeps its name with the Shell dark-blue value and no longer marks selection or focus.

**What each colour may and may not do**

| Colour | Use it for | Never use it for |
|---|---|---|
| Shell yellow | The one primary button of a page or dialog; the 4 px current-place bar on the current sidebar item and the selected tab; the skip link; the pale tint of the current sidebar item and the highlighted row | Text or icons on white or grey (1.55:1); a focus ring on light backgrounds; warnings (amber is used); large areas; chart series; a second yellow button |
| Shell red | The destructive confirm button; error messages and icons; the problem status family (§4.1) | Headings, decoration, main actions, brand accents; red with yellow in stripes, gradients or shell shapes (Shell trade dress) |
| Shell dark blue `#003C88` | Links; checked checkboxes and switches; the plan level | Buttons, headings or large areas |
| Navy `#092949` | The focus ring in light mode; the MAPP mark and avatars; the environment banner | Other surfaces |
| Shell secondary colours (blue, green, purple, teal, magenta) | Meaning only: feedback, status families, planning levels, team dots, charts — always next to a word | Buttons, headings, backgrounds of whole areas |
| Ink and greys | Everything else | – |

### 3.2 Typography

| Token | Value | Use |
|---|---|---|
| `--font` | `Arial, Roboto, Helvetica, sans-serif` | All body and interface text: labels, inputs, buttons, tables, menus. Shell's own body stack; Arial's digits are equal width, so money columns line up. |
| `--font-heading` | `'Jost', Arial, Roboto, Helvetica, sans-serif` | Headings, the product name and summary-card values. Jost 600 and 700, Latin subset, self-hosted from `@fontsource/jost` 5.3.0 (SIL Open Font License 1.1); no font is loaded from the internet. It stands in for Shell's proprietary heading typeface (§12). |
| `--weight-regular` / `--weight-bold` / `--weight-heading` | 400 / 700 / 600 | Arial uses 400 and 700 only; Jost uses 600 and 700 |

**Type scale** — seven sizes.

| Token | Size / line height | Family and weight | Use |
|---|---|---|---|
| `--text-xs` / `--leading-xs` | 12 / 18 | Arial 400; 700 for table headers, chips, tags, badges | Breadcrumb, table headers, chips and tags, count badges, captions, timestamps, the footer, the banner. Never below 12 px. |
| `--text-sm` / `--leading-sm` | **14 / 20** | Arial 400; **700** for buttons, field labels, stand-alone links, emphasis | **Default**: body, table cells, inputs, hints, notes, errors, menu items, sidebar items |
| `--text-md` / `--leading-md` | 16 / 24 | Jost 600 for card titles; Arial 700 for sub-sections | **Card titles** (H2); sub-section headings (H3) inside a card or panel; lead text; input text below 600 px (stops phone zoom) |
| `--text-lg` / `--leading-lg` | 18 / 24 | Jost 600 | Dialog, side-panel and form-section titles; the product name in the top bar |
| `--text-xl` / `--leading-xl` | 20 / 28 | Jost 600 | Section headings between cards, empty-state titles; the page title below 600 px |
| `--text-2xl` / `--leading-2xl` | 28 / 36 | Jost 700 | **Page title (H1)**; summary-card values (tabular figures) |
| `--text-3xl` / `--leading-3xl` | 32 / 40 | Jost 700 | The Home greeting only |

Rules:

- **Weights:** Arial 400 and 700 only (Arial has no real 600; a requested 600 renders as faux bold or as
  700 depending on the browser); Jost 600 and 700 only.
- **Sentence case everywhere**; no upper-case labels, eyebrows or tags; `letter-spacing: normal`.
- Headings and body share `--ink`; hierarchy comes from size, weight and family, not colour.
- Numbers in tables, totals, money and summary values use `font-variant-numeric: tabular-nums`.
- Text is never smaller than 12 px; long reading text (help, descriptions) is 14 px or more.

### 3.3 Spacing

Shell's 4 px grid; the number in the token is the multiple of 4. Only these steps exist; the single
exception is 1–2 px for borders and focus offsets.

| Token | `-1` | `-2` | `-3` | `-4` | `-5` | `-6` | `-7` | `-8` | `-9` | `-10` | `-11` | `-12` | `-14` | `-16` |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| `--space-*` (px) | 4 | 8 | 12 | 16 | 20 | 24 | 28 | 32 | 36 | 40 | 44 | 48 | 56 | 64 |

| Situation | Value |
|---|---|
| Label to control (or label to hint, hint to control) | 8 (4 between label and hint) |
| Control to its note or error | 4 |
| Between fields | 20 |
| Between form sections / sections of a panel | 32 inside one card; 20 between section cards |
| Card padding | 24 (16 below 600 px); summary cards 20 |
| Card header to body; between children of a card | 16 |
| Between cards in a grid | 20 |
| Breadcrumb to title · title to meta row · title to description | 8 · 8 · 4 |
| Page header to content | 24 |
| Between controls in a toolbar or button group; icon to text | 8 |
| Table cells | 8 vertical × 12 horizontal (first and last cell 24 horizontal, aligned with the card padding) |

### 3.4 Corner radius

| Token | Value | Objects |
|---|---|---|
| `--radius-1` (= `--radius-chip`) | 4 px | Status chips, level and team tags, filter chips, field markings, tooltips, menu items, checkboxes |
| `--radius-2` (= `--radius-control`, `--radius-card`) | 8 px | Buttons, inputs, selects, search fields, segmented filters, menus, popovers, toasts, inline alerts, the table outline, **cards and summary cards** |
| `--radius-3` (= `--radius-dialog`) | 12 px | Dialogs |
| `--radius-4` | 16 px | The sign-in card only (kept for parity with Shell's scale) |
| `--radius-pill` | 999 px | Avatars, count badges, progress bars and switches **only** — no pill-shaped buttons, tabs, search fields or chips |

Side panels and the drawer are square (they touch the window edge).

### 3.5 Elevation

| Level | Token | Light | Dark | Objects |
|---|---|---|---|---|
| 0 | `--shadow-sm` | none | none | Cards: flat, with a 1 px `--card-border` on the grey page |
| hover | `--shadow-lift` | `0 4px 12px rgba(0,0,0,.12)` | `0 4px 12px rgba(0,0,0,.45)` | Hover on clickable cards only; the card does not move |
| 1 | `--shadow-md` | `0 2px 8px rgba(0,0,0,.16)` | `0 2px 8px rgba(0,0,0,.5)` | Menus, popovers, tooltips, toasts |
| 2 | `--shadow-lg` | `0 4px 24px rgba(0,0,0,.2)` | `0 4px 24px rgba(0,0,0,.6)` | Dialogs, side panels, the drawer |

Sticky bars (top bar, table header, form footer) use a 1 px `--hairline` border, not a shadow.

### 3.6 Control heights and icon sizes

| Token | Value | Used for |
|---|---|---|
| `--control-sm` | 32 px | Controls inside tables and card headers, segmented filters, filter chips, the ⋯ of a table row |
| `--control-md` | **40 px** | **Default** for buttons, inputs, selects, search, page-header actions, menu items, sidebar items, table rows |
| `--control-lg` | 48 px | Sign-in, form inputs and form-footer buttons below 600 px, tabs |

Icons are 16 px in `sm` controls and chips, 20 px in `md` controls, the sidebar and the top bar, and 48 px
in empty and error states. Controls next to each other always use the same size. Every clickable target is
at least 24 × 24 px (WCAG 2.5.8); small icons (ⓘ, chip ×, clear search) get a 24 px target around a 16 px
icon.

### 3.7 Page widths and layout sizes

| Token | Value | Token | Value |
|---|---|---|---|
| `--content-max` | 1300 px | `--topbar` | 56 px |
| `--form-max` | 720 px | `--sidebar` / `--sidebar-collapsed` | 240 / 64 px |
| `--gutter` / `--gutter-sm` | 24 / 16 px | `--grid-gap` | 20 px |

Component sizes (written in the component's CSS until a second component needs them as a token): banner
32 px; form rail 300 px; drawer 280 px; side panel 480 px (560 px wide variant); dialog 512 px (720 px wide
variant); menus 220–288 px; tooltip max 280 px; popover max 320 px; toast 360–480 px.

Breakpoints (§2.2) are 600, 960 and 1280 px; CSS cannot read tokens inside media queries, so the lint rule
of §3.12 allows only these widths.

### 3.8 Motion

One duration `--dur` 250 ms (Shell's animation time) with `--ease-out` `cubic-bezier(0.22, 1, 0.36, 1)`
for panels, menus, toasts and hover colours. Menus and popovers fade in with an 8 px move; side panels slide
from the right; toasts slide up. Nothing moves on hover. **With reduced motion: no movement, no fades, no
shimmer** (`global.css`).

### 3.9 Icons

One outline set: `lucide-react` (ISC licence), stroke 2, `currentColor` (`--ink` or `--ink-2`), sizes of
§3.6. Icons never carry meaning alone: an icon-only button has an accessible name and a tooltip (§9).
No filled icons, no photography in working screens, no consumer graphic devices (sparkles, halftone dots,
chevron crops).

### 3.10 Logo and brand assets

- **MAPP's own mark** in the logo slot: a 28 px tile, radius 8, filled `--brand-mark` (navy), with a
  white four-point star, and the word "MAPP" in Jost 600 18 px `--ink`. The favicon is the same tile.
  No gradient.
- **No Shell pecten, no "Shell" wordmark, no Shell typeface, no red-and-yellow shell shapes or stripes**
  until Shell gives trade-mark consent and artwork approval (§12). The word "Shell" may appear as plain
  text where it describes the business ("Shell attributes").
- If Shell approves the pecten later, it replaces the tile exactly as Shell places it (symbol only, full
  colour, top-left, linking home), following Shell's clear-space and minimum-size rules.

### 3.11 Dark mode

Built on Shell's own dark palette (blue-tinted charcoal, dimmed yellow with dark text, yellow focus ring).
It follows the computer's setting first, then the person's choice in the account menu (D-038). Every pair
in §3.1 and §4 passes AA in both themes; the token contrast test (§11, rule 8) checks them.

### 3.12 Token rules

- No raw colour, font size, font weight, line height, radius, shadow, control height or spacing value in a
  component's CSS: only `var(--token)` (1–2 px borders and focus offsets, and the component sizes of §3.7,
  excepted).
- Media queries use only `max-width: 599px`, `min-width: 600px`, `max-width: 959px`, `min-width: 960px`,
  `min-width: 1280px`, `prefers-reduced-motion`, `prefers-color-scheme`, `forced-colors` and
  `hover`/`pointer`.
- A stylelint configuration enforces both (implementation plan, phase 1).

---

## 4. Status, level and money colours

### 4.1 Status families (status colour meanings)

Status is always **word + colour (+ icon where space allows)**; colour is never the only signal. **Each word
belongs to exactly one family, and the same word never appears in two colours** (§11, rule 6). One mapping
in code (`lib/statusTone.ts`) is the only place a word gets its family.

| Family (token name) | Meaning | Words | Dot (light / dark) | Chip text light (on its 12 % tint) | Chip text dark (on its 16 % tint) |
|---|---|---|---|---|---|
| **Neutral** (`--status-neutral`) | Not started, nothing to do yet, switched off | Draft, Created, Not started, No activities, Not requested, Inactive | `#6E6E6E` / `#8E99A4` | `#595959` (6.04) | `#B0B8C1` (6.10) |
| **Waiting** (`--status-review` = warning) | Someone must decide or check; also non-blocking warnings | Submitted, Waiting for approval, Partly approved (n of m approved), Waiting for activity approval | `#C77700` / `#E09A2A` (amber — never brand yellow) | `#8F5400` (5.36) | `#F0A640` (5.84) |
| **Positive** (`--status-positive` = success) | Agreed, published, complete and correct | Published, Approved, All approved, All filled in, Active (Settings records, people), Ready to send to GSAP | `#008557` / `#3DBB8C` | `#00704A` (5.22) | `#4CC79A` (5.67) |
| **Running** (`--status-active` = info) | Approved and under way in time | In progress (activity running), Running | `#336094` / `#6F9BD1` | `#2B5482` (6.56) | `#8FB4E3` (5.73) |
| **Done** (`--status-done`) | Finished | Complete, Finished | `#5B6B7A` / `#8696A6` | `#4D5C6A` (5.86) | `#A9B8C6` (6.11) |
| **Problem** (`--status-problem` = danger) | An error or something that blocks the next step | Declined, *n* fields needed, *n* items below need attention, over-allocated budget, negative balance, failed | `#DD1D21` / `#FF5C5F` | `#B8171B` (5.45) | `#FF7B7E` (5.14) |

All chip texts also pass on white and on grey (≥ 5.6 light) and on the dark page and surface (≥ 6.3).

- "In progress" is used **only** for a running activity. The approval roll-up of plans, campaigns and
  initiatives is written "Partly approved" or "2 of 4 approved" (functional review A-02; product-owner
  confirmation listed in the implementation plan).
- Amber is the attention colour; red is reserved for danger and errors (product-owner decision 5,
  D-314). Yellow never marks a status.
- A row shows **at most two chips**: its status and one problem chip; anything else goes into the problem
  chip's popover.

### 4.2 Level colours

A 4 px bar beside the name in trees and the level tag's border. The level is always written in the tag.

| Level | Light | On white · grey | Dark | On surface |
|---|---|---|---|---|
| Plan | `#003C88` Shell dark blue | 10.49 · 9.62 | `#6F9BD1` | 5.49 |
| Campaign | `#87217D` Shell purple | 8.26 · 7.57 | `#C96BBF` | 4.78 |
| Initiative | `#008192` Shell teal | 4.61 · 4.23 | `#3FB6C5` | 6.57 |
| Activity | `#A3195B` magenta | 7.38 · 6.76 | `#E06A9F` | 5.08 |
| Line item | `#6E6E6E` grey | 5.10 · 4.68 | `#8E99A4` | 5.46 |

Levels never use yellow (yellow means "act here" or "you are here"), nor the green of "positive" or the
amber of "waiting". Level marks are outlined tags and bars; status marks are filled tinted chips, so the two
never look alike.

### 4.3 Team colours

The decorative dot of a team tag (the name is always written, D-025). Token names are stored data and do
not change; the values are unchanged from version 1 and the same in both themes: `--team-blue` `#2A78D6`,
`--team-violet` `#6E56CF`, `--team-teal` `#12A594`, `--team-magenta` `#D6409F`, `--team-orange` `#E5701A`,
`--team-green` `#2F9E44`, `--team-slate` `#5B6B7A`, `--team-gold` `#C79400`. Because the dot is decoration
next to the written name, it has no contrast requirement.

### 4.4 Money and numbers

- **Right-aligned with tabular figures** in tables, totals and editable amount cells; the column header is
  right-aligned too, with the same padding, so values sit exactly under it (§11, rule 2).
- **One money format per screen.** By default the symbol form of the person's display currency
  ("$12,500.00"; compact "$1.25M" in summary cards with the full value on hover and focus). When one screen
  shows amounts in more than one currency, every amount on that screen uses the code form ("12,500.00
  USD"). This replaces mixed forms on one page (UI audit AC-04, functional review P-07); it refines
  decision 007-P4 and needs the product owner's confirmation (implementation plan).
- Negative values in `--danger-text` with a minus sign ("−$500.00"); never only red.
- **Nothing is "—"**: an empty amount, a missing value and "no activities yet" show "—", never 0 or
  "$0.00"; a short reason may follow in `--ink-2` ("— · Nothing planned yet").
- Percentages without a space: "38%" (UK style); shares in the split editor "33.33%".
- Whole numbers grouped: "1,250".

### 4.5 Dates and times

- **One date format: the person's choice** (default `DD MMM YYYY`, "12 Mar 2026"), in text **and in date
  inputs** (§5.5 `DateRangeField`). Ranges follow D-216 ("01 Mar – 31 May 2026"; "Ongoing from 01 Mar
  2026").
- Times are 24-hour: "14:05"; date and time: "12 Mar 2026, 14:05".
- All dates go through `lib/format.ts` (`useFormatters()`); no component formats a date itself.

### 4.6 Progress bars

8 px high, radius pill, track `--chip`, fill `--info`; fill `--danger` above 100 % (the fill stops at the
end, the real percentage is written and announced). The value is always written next to the bar
("38% used").

---

## 5. Components

Every shared component lives in `frontend/src/components/` and is imported from `@/components`. Features
reuse them instead of styling their own (FR-019, D-197). Sizes are in px; tokens are in §3 and §4. Unless a
row says otherwise, every interactive component shows the focus ring (2 px `--focus-ring`, 2 px offset),
has a 24 × 24 px minimum target and works with the keyboard alone. Components marked **new** are added
in version 2.

### 5.1 Actions

**`Button` and `LinkButton`**

| | Spec |
|---|---|
| Sizes | `sm` 32 high, padding 0 12, icon 16 · `md` 40, padding 0 16, icon 20 · `lg` 48, padding 0 20, icon 20. Text 14/20 Arial 700; icon 8 px before the label; icon-only buttons are square (32 / 40 / 48) and need `aria-label` plus a `Tooltip`. Radius 8. |
| `primary` | `--button-primary-bg` fill and border, `--button-primary-text`; hover `--button-primary-bg-hover`, pressed `--button-primary-bg-pressed`. **The page's or dialog's one main action** (§11, rule 1). |
| `secondary` | `--surface` fill, 1 px `--control-border`, `--ink` text; hover `--hover` fill and `--control-border-hover`; pressed `--chip`. Every other button, including Cancel. |
| `ghost` | Transparent, `--ink` text; hover `--hover`; pressed `--chip`. Icon buttons in headers and tables, "Collapse sidebar". |
| `danger` | `--danger-bg` fill, `--on-accent` text; hover `--danger-bg-hover`. **Only** as the confirm button of a destructive `ConfirmDialog`; it then is that dialog's single main action. Elsewhere destructive actions are menu items (§6.1). |
| Busy | A spinner replaces the icon, `aria-busy="true"`, the label stays, a second click is ignored. Never greyed out. |
| Unavailable | Avoid. The button stays enabled and explains on click (§6.6); if truly disabled: `aria-disabled`, `--gray-bg` fill, `--muted` text, `--hairline` border, a `LockedNote` beside it. |
| Words | A verb naming the action, sentence case: "Create campaign", "Submit for approval" — never "OK", "Yes" or "Submit". |

**`ActionsMenu`** (the ⋯ menu)

- Trigger: ghost icon button, `sm` in table rows and panel headers, `md` in the page header; named
  "Actions for <record>", with a tooltip.
- Menu: 220–288 wide, `--surface`, radius 8, padding 4, `--shadow-md`. Items 40 high, padding 0 12, text
  14, optional 16 icon, radius 4; hover and keyboard focus `--hover`; optional hint line 12 `--ink-2`.
- Order: View · Edit · Duplicate / Copy to… / Move to… · record-specific actions · View history · then a
  1 px `--hairline` separator and **Delete** last in `--danger-text`.
- Not modal; arrow keys move, Enter chooses, Escape closes and returns the focus to ⋯; the chosen action
  runs after the menu has closed. No tooltips inside a menu (research R14).

### 5.2 Navigation and filtering

**`PageHeader`** — the layout of §2.3. Props: `breadcrumb` (items), `title`, `meta` (level tag, chips, ID,
owner, dates), `description`, `actions` (`more`, `secondary`, `primary`); the `eyebrow` and `highlight` props
of version 1 are removed. It renders the page's only `<h1>`, reserves the breadcrumb row on every page and
accepts at most one primary action (a second one is a development-time error).

**`Breadcrumb`**

- 12/18, items 8 apart with a 12 px chevron (`--ink-2`, hidden from screen readers); links `--link`,
  underlined on hover and focus, each with a 24 px tall target; the current page `--ink-2`, not a link,
  `aria-current="page"`; `nav aria-label="Breadcrumb"`.
- Shown on every page below the top level; its row is reserved on every page (§2.3).
- Below 600 px only "‹ <parent>" (a 16 px chevron and the parent's name) — never wraps to two lines.
- More than 4 steps: middle steps collapse into "…" with a menu.

**`Tabs`** (new; replaces `PillTabs`, which stays as an alias until every use moves)

- Tab height 48, text 14, 24 between tabs, no fill; the list has a 1 px `--hairline` bottom border.
- **Selected tab: bold text and a 4 px `--brand-yellow` bar on its bottom edge** (Shell's "you are
  here"); `aria-selected="true"`. Hover: text underline.
- Counts after the label: "Activities (12)", or a `CountBadge` for things to act on.
- ARIA `tablist` / `tab` / `tabpanel`, one tab in the tab order, ← → Home End move and show.
- Too many tabs for the width: the list scrolls sideways with fade edges; tabs never wrap (UI audit AP-04).
- Page-section tabs keep the chosen tab in the address.

**`SegmentedFilter`** (status, level or layout of a list)

- Joined buttons 32 high, text 14, padding 0 12; one 1 px `--control-border` around the group, radius 8 on
  the outer corners, 1 px dividers between buttons; no pill shape.
- Chosen: `--ink` fill, `--surface` text (10.37 light, 12.00 dark), bold; `aria-pressed="true"`. Others:
  `--surface`, `--ink`; hover `--hover`.
- Counts in brackets: "Active (5)". Order and default for record states: **Active · Inactive · All**,
  default Active; for workflow states: **All · Draft · Published**, default All.
- Below 600 px the group scrolls sideways; it never wraps.

**`SearchField`**

- 40 high, radius 8 (never a pill), 1 px `--control-border`, `--surface`; 16 px magnifier at 12 from the
  start; text 14 (16 below 600 px) from 36; width 280–360 in toolbars, full width below 600 px.
- The placeholder says what it searches ("Search name or ID"), `--muted`; the accessible name is a
  hidden label ("Search plans").
- **Clear search**: 16 px ✕ in a 24 × 24 target at the end, once something is typed; focus returns to the
  input.
- Hover border `--control-border-hover`; focus ring. The list shows the result count next to it.

**`FilterChips`**

- A chip per filter in use: 32 high, padding 0 8 0 12, `--chip` fill, `--ink` 12/18 bold, radius 4,
  16 px ✕ in a 24 × 24 target; the whole chip is the button "Remove filter: Status: Draft".
- **Clear filters**: ghost `sm` button after the chips. Nothing shows while no filter is in use.

**List toolbar** (layout rule, new)

One 40 px row inside the table card: `SearchField` · `SegmentedFilter` · **Filters (n)** (secondary) ·
spacer · result count "4 of 6 plans" (14 `--ink-2`) · view switch · **Export** (secondary). Filter chips
on a second row. Every list writes its count the same way: "*n* of *m* <things>".

### 5.3 Containers, panels and dialogs

**`Card`**

- `--surface`, 1 px `--card-border`, radius 8, padding 24 (16 below 600 px), no shadow.
- Header: at least 40 high, items **centred vertically** (title and header buttons on one centre line);
  title `<h2>` Jost 600 16/24; optional subtitle 14 `--ink-2` under it; actions at the end (secondary or
  ghost `sm`/`md`). No divider line under the header; 16 px to the body.
- Cards in one row of the grid stretch to the same height. In a row, either every card has a subtitle or
  none does.
- Variants: **table card** (no body padding; the toolbar and table touch the card edges, the table's
  first and last cells keep 24 px); **link card** (the whole card is one link; hover `--shadow-lift`, no
  movement; a 16 px chevron at the top end).

**`SummaryCard` and `LinkSummaryCard`**

- `--surface`, 1 px `--card-border`, radius 8, padding 20, min height 104, no shadow.
- Three rows: label (14/20 `--ink-2`, ⓘ right after the label), value (Jost 700 28/36, tabular, `--ink`),
  detail (14 `--ink-2` or a `ProgressBar`).
- **A row of summary cards shares its row tracks** (CSS grid with subgrid), so values line up even when one
  label wraps (UI audit H-03, AW-02); labels are written to fit one line at the 4-up width.
- The value is coloured only when the colour means something: `--danger-text` for over-budget or negative,
  `--warning-text` for "waiting". Shortened values ("$1.25M") show the full value on hover and focus.
- `LinkSummaryCard`: the whole card is one link named "<label>: <value> — <where it goes>"; hover
  `--shadow-lift`; a 16 px chevron at the top end. **No tinted icon tiles** (version 1 style; hard to see
  in dark mode, UI audit D-01).
- Rows: 4 per row at ≥ 960 px; pages with 5 or 6 values use 3 per row; 2 per row below 960 px; an odd
  last card spans the full row below 600 px.

**`SidePanel`** (one panel layout)

- Width 480 (560 for panels with tables); full width below 600 px; full window height from the right
  edge; `--surface`, 1 px `--hairline` start border, `--shadow-lg`, `--overlay` behind it; slides in 250 ms.
- **Header** 64 high, padding 0 24: title Jost 600 18/24 (wraps; the header grows), optional description
  14 `--ink-2`; at the end the record's **⋯** (if it has actions) and **Close** (40 px icon button named
  "Close", no tooltip, R14).
- **Body** padding 24 (16 below 600 px), scrolls; details use `DescriptionList`; never a second breadcrumb.
- **Footer (the only panel footer layout)**: sticky, 72 high, padding 16 24, 1 px `--hairline` top
  border, buttons **right-aligned, 8 apart: Cancel or Close (secondary) · other secondary actions · the
  primary last**. Destructive and rare actions (Delete, Deactivate, Set inactive) are never in the footer:
  they live in the header's ⋯. Below 600 px the buttons stack full width in the same order (primary at
  the bottom, nearest the thumb).
- The component takes `primaryAction`, `secondaryActions`, `cancel` and `moreActions` props instead of
  free-form footer content, so every panel gets the same footer (§11, rule 4).
- Focus moves into the panel and stays there; Escape, Close and a click on the overlay close it (asking
  first when there are unsaved changes); the focus returns to what opened it.

**`FormPanel`** — a `SidePanel` with a form: footer **Cancel · Save and add another** (when creating
children) **· Save** (primary); the failure message and `FormErrorSummary` at the top of the body; "Discard
changes?" before typed values are lost. Save is never disabled.

**`ConfirmDialog`, `FormDialog` and `ClearValuesDialog`** (one dialog layout)

- Width 512 (max: window minus 32), radius 12, padding 24, `--shadow-lg`, `--overlay`; centred.
- Title Jost 600 18/24; a **Close** icon button (32 px, named "Close", no tooltip) at the top end, which
  acts as Cancel; text 14/20 `--ink`.
- Buttons 24 below the content, **right-aligned: Cancel (secondary) · the confirm button last**. Below
  600 px they stack full width in the same order.
- Destructive questions: confirm is `danger` and names the action ("Delete campaign"); **Cancel has the
  focus**. Other questions: confirm is `primary`. `ClearValuesDialog`: "Keep editing" (focused) and "Clear
  and save" (primary).
- `FormDialog`: the first field has the focus; Enter submits; a second submit while busy is ignored.
- `role="alertdialog"` for questions, `dialog` for forms; the focus returns where it was.

### 5.4 Data display

**Data table standard** (new: one shared table style, `components/DataTable/`)

| Part | Spec |
|---|---|
| Container | Inside a table card; a scroll box with `position: relative; overflow-x: auto` (UI audit X-02), focusable (`tabindex="0"`, named "<table name>, scrolls sideways") when it overflows, with a fade shadow on the edge that has more content (AW-05) |
| Header row | 40 high, 12/18 Arial 700 `--ink`, **sentence case**, `--gray-bg` fill, 1 px `--hairline` bottom; sticky while the table scrolls; sortable headers are buttons with an arrow and `aria-sort` |
| Rows | Min 40 high (compact 32), cells 8 × 12 (first and last 24 horizontal), 14/20 `--ink`, 1 px `--row-divider` between rows, no vertical lines; hover `--hover`; highlighted row `--brand-yellow-tint` with a 4 px yellow start bar |
| Name column | First; the record name as a link: `--link`, 14 regular, underline on hover and focus; min width 200; wraps at word boundaries to at most 2 lines, then ellipsis with the full name in a tooltip. Never breaks inside a word (no `overflow-wrap: anywhere` in tables, UI audit X-01). |
| Person and team columns | `PersonLabel` / team tags, **no wrapping**, min width 160; more than 2 teams show "+n" with a tooltip |
| Status column | At most two chips (§4.1), no wrapping |
| Number, money, percent | **Right-aligned header and values, tabular figures, no wrapping**, one money format (§4.4) |
| Dates | Left-aligned, no wrapping, the person's format |
| Actions | Last column, 48 wide, the row's ⋯ (`sm`); its header is a visually hidden "Actions" |
| Totals row | 700 weight, `--gray-bg`, 2 px `--ink` top border; numbers aligned as above |
| Never | Repeat the column's label inside the cell ("Owner: Alex Morgan", UI audit P-02); upper-case headers |
| Below 600 px | Scrolls inside its card; inbox-type lists (Approvals, Home lists) switch to stacked cards: name link, status, key facts as a 2-column `DescriptionList`, ⋯ |

**`TreeTable`** — the data table standard plus: a 4 px level bar at the start of the name cell; 24 px
indent per level; an expand button (24 × 24 target, 16 px chevron) before the name, named "Expand <name>" /
"Collapse <name>"; **Expand all · Collapse all** as secondary `sm` buttons in the card header; the plan row
shows its owner like every other row (avatar + name, UI audit PL-04); ARIA treegrid with one row in the tab
order (↑ ↓ → ← Home End Enter) as built; rows focus with an inset ring.

**`DescriptionList`** (new: the only key–value layout)

- A grid: label column **200 px** fixed at ≥ 600 px, then the value; 12 px between rows, 16 between
  columns; label 14 `--ink-2`, value 14 `--ink`; empty "—"; long values wrap.
- Below 600 px the label sits above its value (4 px).
- Used by details panels, record pages, Shell attribute values, finance coding and the stacked-card rows,
  so values start at the same x everywhere (UI audit X-08).

**`StatusChip`** — 24 high, padding 0 8, radius 4, 12/18 Arial 700, an 8 px dot in the family colour then
the word, `--status-<family>-text` on the family's tint; never wraps; the word is the meaning (§4.1).

**`CompletenessChip`** — a `problem` chip "3 fields needed" (or `positive` "All filled in"); a button that
opens a popover (radius 8, `--shadow-md`, padding 12 16, max 320 wide) listing the missing fields as links
to each field; "n items below need attention" is a second chip only where no status chip competes; in tree
rows it folds into the first chip's popover (at most two chips per row).

**`LevelTag`** — 20 high, padding 0 6, radius 4, 12/18 Arial 700 `--ink` on `--surface`, 1 px border in the
level colour; **sentence case** ("Campaign", "Line item"), never smaller than 12 px (UI audit AX-01).

**`CountBadge`** — min 20 × 20, padding 0 6, radius pill, the waiting family: `--status-review` at 16 %
with `--status-review-text`, 12/18 bold (5.11 light, 5.84 dark) — something waits for you; "99+" above 99;
a visually hidden label ("4 waiting for you"); nothing for 0.

**`ProgressBar`** — §4.6; `role="progressbar"` with the real percentage as `aria-valuetext`.

**`PersonLabel` and `Avatar`** — avatar 24 (`sm`, tables and pickers) or 32 (`md`, top bar and panels),
radius pill, `--brand-mark` fill with `--on-accent` initials 12/18 bold (14.74 light, 11.58 dark), hidden
from screen readers; then the full name 14 `--ink`, " (you)" and " (inactive)" written out; an optional
second line 12 `--ink-2`. In tables the name never wraps (ellipsis with a tooltip).

**`TeamPill` / `TeamPills`** — a team **tag** (no pill shape): 24 high, padding 0 8, radius 4, `--chip`
fill, an 8 px team-colour dot, the name 12/18 `--ink`, " (inactive)" written out; max width 200 with an
ellipsis and the full name in a tooltip; never breaks inside a word; as a link, underline on hover. Several
tags wrap onto more lines with 8 px gaps (outside tables).

**`ShellAttributeValues`** — the shown attributes in a `DescriptionList`, grouped by section with an `<h3>`
(16/24 Arial 700) — **but never a heading equal to its parent's** ("Shell attributes" inside the "Shell
attributes" card shows no inner heading; UI audit X-09); "—" for empty; Yes / No; "(inactive)" written
out; the notes of the form.

### 5.5 Form fields

**`FieldShell`** (the frame of every field; also `inputs.module.css`)

| Part | Spec |
|---|---|
| Label | Above the control, 14/20 Arial 700 `--ink`; then 8 px and the **marking** as a small tag ("Required", "Needed to publish", "Needed to submit": 20 high, padding 0 6, radius 4, `--chip`, 12/18 bold `--ink-2`); then an optional ⓘ |
| Hint | **Always under the label, above the control** (one rule, UI audit F-02), 14/20 `--ink-2` |
| Control | 40 high (48 below 600 px), padding 0 12, radius 8, 1 px `--control-border`, `--surface`, text 14 `--ink` (16 below 600 px), placeholder `--muted` |
| States | Hover: border `--control-border-hover`. Focus: the focus ring. Invalid: 1 px `--danger` border plus the message. Locked: `--gray-bg` fill, `--muted` text, `--hairline` border and a `LockedNote`. Read-only values are shown as text, not as inputs. |
| Note and error | Under the control, 4 px gap, 14/20: note `--ink-2`; error `--danger-text` with a 16 px alert icon. Screen readers hear marking, hint, note and error with the control (`aria-describedby`). |
| Width | Full column by default; short values use fixed widths: `xs` 120 (percent, year), `sm` 240 (money, codes, dates), `md` 360 |

**`TextField`, `PasswordField`, `TextAreaField`** — the control above; the password field has a **Show
password** toggle (32 px ghost icon button inside the end, `aria-pressed`); the text area is at least 96
high (4 rows), padding 8 12, resizes vertically, and counts down in its last 50 characters ("12 characters
left"; past the limit "3 characters too many"; typing is never cut off).

**`DecimalField` and `MoneyField`** — the control above with the value **right-aligned in tabular
figures**; the unit ("USD", "%") inside the end in 14 `--ink-2`; written out on leaving the field
("12,500.00"; "Rounded to 1,234.57" as a note); accepts "12500", "12,500.00" and "12.5k".

**Choices: one select style** (`ChoiceField`, `Combobox`; UI audit X-15)

| Situation | Control |
|---|---|
| One of up to 10 options | A native `<select>` drawn exactly like an input (same height, border, radius, padding), `appearance: none`, a 16 px chevron at the end. Used by forms, the Plans year filter and the Home currency picker alike. |
| One of more than 10 options | `Combobox` (searchable) |
| Several of up to 6 options | Checkboxes in columns (auto-fill, min 200 wide) — not a long vertical list (UI audit AF-01) |
| Several of more than 6 options | `Combobox` with `multiple`; chosen values as removable chips under the input |

**`Combobox`, `PersonPicker`, `TeamPicker`** — input as above; the list is a popover the width of the
input (min 280), `--surface`, radius 8, `--shadow-md`, max height 320 with scrolling; a count line at the top
(12 `--ink-2`: "300 people"); options 40 high (two-line options grow), hover and active `--hover`, chosen
options ticked (16 px) and `aria-selected`; "No one matches 'xyz'." when empty; chosen values as chips
(24 high, radius 4, `--chip`, ✕ in a 24 px target). People show avatar, name and job title; "(you)" first.
Inactive values that are kept carry an ⓘ explaining why.

**`CheckboxField`** — a 20 × 20 box, radius 4, 1.5 px `--control-border`; checked: `--blue` fill with a
white tick (`accent-color: var(--blue)`); the label (14 `--ink`, regular) is part of the click target; the
row is at least 24 high; ⓘ, hint, `LockedNote` and error as in `FieldShell`. A group of checkboxes is a
`fieldset` whose legend uses the label style. Switches (All work) are 40 × 24, radius pill, `--control-border`
when off and `--blue` when on, with the label on the right.

**`DateRangeField`** — two date inputs (start, end) side by side, each `sm` 240 wide, as text inputs that
**show and accept the person's date format** ("01 Mar 2026"; typing "1/3/26" or "2026-03-01" also works and is
rewritten on leaving the field), each with a 32 px **Choose date** button that opens a calendar popover
(month grid, ← → ↑ ↓ move by day, Page Up/Down by month, Enter picks, Escape closes). The allowed range
limits the choice and is written under both inputs ("Must be inside the campaign's dates (01 Feb – 30 Nov
2026)"); a non-blocking warning shows in `--warning-text` with a warning icon. This replaces the browser's
date input, which shows "12/31/2026" in US-English browsers (UI audit X-15, C-04; functional review P-03).

**`SplitEditor`** — the chosen values in a small table: value · share (a `DecimalField`, `xs`, right-aligned,
"%" unit); under it the running total in 14 bold: "100.00% — complete" (`--success-text`), "90.00% — 10.00%
left" (`--ink-2`), "110.00% — 10.00% too much" (`--danger-text`); **Split evenly** as a secondary `sm` button.

**`ShellAttributeFields`** — the attribute sections in the order of the form definition, each field by the
choice rules above; dependent fields stay visible but locked with a reason until their parent is set; the
Settings preview highlights one attribute with `--brand-yellow-tint` and a 4 px yellow start bar (was a
blue tint); never a heading equal to its parent's (X-09).

**`FormSection`** — one white card per section on form pages: title `<h2>` Jost 600 18/24, optional
description 14 `--ink-2`, fields 20 apart. Inside a side panel, sections are separated by 32 px and an
`<h3>` (16/24 Arial 700) instead of cards.

**`FormFooter`** — sticky at the bottom of the window across the content column, `--surface`, 1 px
`--hairline` top border, padding 16 24, 72 high; buttons right-aligned on the content's right edge:
**Cancel · Save and add another · Save** (primary last). Save is never disabled. Below 600 px the buttons
stack full width in the same order.

**`FormErrorSummary`** — an `InlineAlert` of the danger family at the top of the form: title `<h2>` 16/24
Arial 700 "2 problems need your attention", then each problem as an underlined `--link` link to its field;
it takes the focus when it appears, so it is read out (FR-020).

**`LockedNote`** — a 16 px lock icon and one sentence, 14/20 `--ink-2`, 4 px under the locked control.

**`InfoTip`** — a 16 px ⓘ (`--ink-2`) in a 24 × 24 target after a label, named "About <label>"; opens a
popover (`--surface`, 1 px `--hairline`, radius 8, `--shadow-md`, padding 12 16, max 320 wide, text 14
`--ink`) on hover, focus, click and tap; closes on Escape, when the pointer leaves (150 ms grace), on
focus loss and on a click outside (WCAG 1.4.13).

### 5.6 Feedback

**`Toast`** — width 360–480, padding 12 16, radius 8, `--surface`, `--shadow-md`, a 4 px start bar and a
20 px icon in the feedback colour (`--success` for done, `--danger` for failed, `--info` otherwise), the
message 14 `--ink`, one action as a bold underlined `--link` ("View", "Undo"), Close (✕, 24 px target).
**Placement: bottom-left of the content column at ≥ 600 px, full width (16 px margins) below 600 px —
always above a sticky form or panel footer, never covering it** (functional review P-09; WCAG 2.4.11). At
most 3 at once, newest at the bottom. 4 s, or 10 s with an action (FR-026); paused while hovered or
focused; F8 moves the focus to it; announced at once.

**`InlineAlert`** (new) — a message inside the page: padding 12 16, radius 8, the feedback colour at 8 %
(12 % in dark mode), a 4 px start bar and a 20 px icon in the feedback colour, optional title 14 bold, text
14 `--ink`, optional action link. Families: info, warning (waiting), success, danger. `role="status"`, or
`role="alert"` for failures. Used for: waiting and declined notices on record pages (with the decline
reason), failed saves, "view only" notices, "Shown because the plan's business unit includes Lubricants".

**`EmptyState`** — centred in its card, padding 48 24; a 48 px outline icon in `--ink-2`; title Jost 600
20/28; one sentence 14 `--ink-2` (max 480 wide); the action. **The action is a secondary button when the
page header already holds the same primary action** ("New plan"), and the primary only when the page has
no other (§11, rule 1).

**`ErrorState`** — the same layout with a 48 px alert icon in `--danger-text` (a cloud icon when offline),
the plain-language message as `role="alert"` and **Try again** (secondary, with a refresh icon). No
technical details on screen.

**`Skeleton`** — `--chip` shapes, radius 4, the height of the real content (table rows 40, text lines 20) so
nothing jumps when data arrives; a slow shimmer that stops with reduced motion; screen readers hear
"Loading…". Old data stays visible while refreshing.

**`Tooltip`** — `--tooltip-bg` / `--tooltip-text`, 12/18, padding 4 8, radius 4, max 280 wide,
`--shadow-md`, an arrow; opens after 300 ms on hover and on keyboard focus, stays while hovered, Escape
hides it. It only describes: the control has its own name. Never inside menus, never on the close button
of a dialog, panel or drawer (R14).

**`DemoBanner`** — §2.1 (navy, 32 high, one line, 12 px white).

### 5.7 Brand

**`Logo`** — §3.10: the 28 px navy tile with the white star (`md`; `lg` 32 on the sign-in page), optional
wordmark "MAPP" Jost 600 18. The tile is decoration; the link around it is named "MAPP home".

### 5.8 App shell parts (`frontend/src/app/`)

| Part | Spec |
|---|---|
| `TopBar` (new) | §2.1; replaces `MobileTopBar` at every width |
| `Sidebar` | §2.1: white, 240 / 64, items 40, the yellow bar with the pale-yellow fill, "More" group label, Collapse at the bottom. Account, bell and theme move to the top bar. |
| `NavDrawer` | §2.1: below 600 px, 280 wide, from the left |
| `AccountMenu` | Opened from the top bar's avatar button; a menu (§5.1 look) with name and role (not a link), My profile, Appearance (three choices, the current one ticked), a separator, Sign out |
| `NotificationBell` | 40 px ghost icon button with `CountBadge`; opens a 360 wide popover (radius 8, `--shadow-md`) with the notices, **Mark all as read** (ghost) and links |
| `ThemeToggle` | Becomes the Appearance choice inside `AccountMenu` |
| `PublicLayout` (sign-in) | `--page-bg` canvas, one centred white card (max 400 wide, radius 16, padding 32) with the logo, the title "Sign in to MAPP" (Jost 700 28), the fields (48 high) and **Sign in** (primary, `lg`, full width). The brand appears once (UI audit S-01); no gradient header (S-02). |

---

## 6. The standard CRUD pattern

Every user-managed record follows this pattern. Specs refer to it as "the standard CRUD pattern".

### 6.1 List

1. Page header with the plural name, a one-line description and **New <thing>** (primary, at the end of
   the actions).
2. The list toolbar of §5.2: a search box that says what it searches ("Search name or ID"), a segmented
   filter for status, a **Filters** button for the rest, removable filter chips, and a result count ("12 of
   40 plans").
3. Table (or cards for plans) by the data table standard (§5.4): name as a link, status chip, key facts,
   money with currency, a row **⋯ Actions** menu (View · Edit · Duplicate · record-specific actions ·
   Delete last, separated).
4. Selecting a row opens the record's **details view** — never straight into edit.
5. Empty state: "No plans yet — create your first plan" with the create button (secondary, because the
   header holds the primary); when filters hide everything: "Nothing matches these filters" with **Clear
   filters**.

### 6.2 Create

- Entry: the list's **New <thing>** button, or **Add <child>** on the parent (the parent is pre-filled).
- Short records (campaign, initiative, Settings items, person, team, approval workflow) open in a **side
  panel**; long records (plan, activity, line item, Shell attribute) open on a **full page** with sections
  (§2.5).
- One column; sections in a logical order; defaults pre-filled and labelled ("From the campaign");
  markings shown as words: **Required**, **Needed to publish**, **Needed to submit**.
- Footer: **Cancel** · **Save and add another** (for children) · **Save** (primary, last), right-aligned
  (§5.3, §5.5). Cancel asks before discarding typed values.
- After saving: toast "Initiative 'Digital & social' created — View"; the new row is highlighted.

### 6.3 View (details)

- Header (§2.3): breadcrumb (Plans › FY2026 Lubricants › Helix launch), record name, and the meta row with
  level tag, **status chip**, **completeness chip**, ID, owner, dates; money in summary cards; "Created by …
  on …, last changed by … on …" at the end of the Details tab or card.
- **Actions — one primary, chosen by the person's main job on this record:**
  - The owner or an editor: **Edit** (primary) · the next status action as a secondary button
    (**Publish**, **Submit for approval**) · **⋯ More** (Duplicate, Copy to…, Move to…, View history,
    Delete).
  - An approver of the current level, on a record waiting for them: **Approve** (primary) and **Decline**
    (secondary) side by side; Edit moves into ⋯.
  - The owner of a submitted activity is never offered Approve as a button; where the rules let them
    approve (D-020), it is in ⋯ with the warning (functional review A-01; client confirmation pending).
- Sections or tabs (§2.4 B): Overview (read-only fields; empty values show "—"), child records with counts,
  Approval (activities), History.

### 6.4 Edit

- **Edit** turns the details into a form with the same order of sections.
- **Save is never disabled.** On Save, every problem appears at once: a summary at the top with links,
  and a message under each field. Nothing is saved while problems remain.
- Consequences are confirmed before saving ("Changing planned cost sends this activity back for
  approval").
- Leaving with unsaved changes asks **Keep editing** (default) or **Discard changes**.

### 6.5 Delete

- A dialog: "Delete campaign 'Helix launch'?" — what goes with it ("3 initiatives, 12 activities and 20
  line items"), the effect ("Plan totals will update") and where it goes ("It moves to the recycle bin;
  you can restore it"). **Cancel** has focus; the red button says **Delete campaign**.
- After deleting: back to the parent; toast "Campaign 'Helix launch' moved to the recycle bin" with
  **Undo** (visible at least 10 seconds).
- The dialog and toast promise the recycle bin only where a person can open it (UI audit A-01, functional
  review P-02).

### 6.6 Locked and read-only

Nothing is greyed out without a reason next to it:

| Situation | Shown as |
|---|---|
| Calculated | Small "Calculated" label and ⓘ with the formula |
| From finance/GSAP later | "Filled by GSAP later" / "From finance (coming later)" |
| Fixed after save | Lock icon + "Year can't change once the plan is saved — create a new plan for another year." |
| Irreversible choice | Warning when ticked: "Ongoing can't be undone after saving." |
| Waiting for approval | `InlineAlert` (warning): "Submitted for approval — changing planned cost or dates will restart approval." |
| Action not yet possible | The button stays visible; clicking it explains: "Publish needs 4 fields — view list". |
| View-only section | `InlineAlert` (info) under the header: "You can view these settings. Ask an administrator to change them." |

### 6.7 Set inactive (Settings records)

One behaviour for "set inactive" and "deactivate" everywhere: when the record is in use, a dialog lists
where ("Used by 2 line items") before it applies; when it is not in use it applies at once with an **Undo**
toast. Reactivating needs no question. (Functional review S-01; the confirmation for records in use is a
proposal awaiting the product owner, see the implementation plan.)

---

## 7. Forms

- One column, max 720 px wide, left-aligned in the content column, section headings in the order users
  think: *Basics → Dates & cost → Shell attributes → Goals → More details*; each section a card on form
  pages (§5.5 `FormSection`) and the right rail at ≥ 1280 px (§2.4 C).
- Labels above fields; the hint between label and field; the note and the problem under it (§5.5).
- Dependent fields stay visible but locked with a hint until their parent is set ("Choose teams first —
  the owner must belong to one of the plan's teams"). When only one option fits, it is filled in and
  said so ("Region set to 'Retail – China', the only region for this cluster").
- Changing a parent value that invalidates child values lists what will be cleared before saving.
- Attribute groups that depend on the business unit say why they are shown ("Shown because the plan's
  business unit includes Lubricants").
- Validation messages say what is wrong and how to fix it: "End date must be on or after the start date
  (12 Mar 2026)." — never codes or technical text.
- Choices follow the one-select rule (§5.5); date fields show the person's date format.
- The form says how far it is: the rail (or the line under the header) counts the needed fields filled
  ("9 of 15 needed fields filled").

---

## 8. Words on screen

Shell business terms are kept: Plan, Campaign, Initiative, Activity, Line item, SWBS, GL code, BCPIR,
SP&A, AOP, SMP, INQ, MQL. Hive9 interface terms are replaced:

| Hive9 term | MAPP label |
|---|---|
| Inspection Window | Details (side panel or page) |
| Honeycomb | Select (checkboxes and a bulk-action bar) |
| HUD / heads-up display | Summary cards |
| Sys_Gen_Balance | Unassigned planned cost (Sys_Gen_Balance) |
| Create SWBS Code | Request SWBS code from GSAP |
| Override Model | Set later stages myself |
| Evergreen | Ongoing (no end date) |
| Waterfall Goal | Goal stage |
| C / E / F / L | Custom · Evenly · All in first month · All in last month |
| Show Hierarchies | Show levels |
| (Not Completed) | Not filled in |
| Plan Log / Finance Log | Activity log |

Buttons use verbs; messages are short, friendly and specific; **sentence case everywhere, no upper-case
labels**; page titles match their menu names ("All work", "My profile"); the same problem is worded the
same way on every page ("1 line item needs attention"). Dates follow the user's format (default
`DD MMM YYYY`, e.g. 12 Mar 2026). A date range shown as a value is written one way everywhere (D-216): in
`DD MMM YYYY` with the year once when both dates fall in the same year ("01 Mar – 31 May 2026"), otherwise
both dates in full ("01 Dec 2026 – 31 Jan 2027"); in the other formats both dates in full ("01/03/2026 –
31/05/2026"); an ongoing activity reads "Ongoing from 01 Mar 2026". Percentages are written "38%"; empty
values "—" (§4.4).

---

## 9. Accessibility (WCAG 2.2 AA)

The target is **WCAG 2.2 level A and AA**, as in Shell's public accessibility statement (product-owner
decision 4, D-313). Automated checks run axe with the tags `wcag2a`, `wcag2aa`, `wcag21a`, `wcag21aa` and `wcag22aa`.

- **Keyboard:** everything works with the keyboard; the order of focus follows the order on screen; menus,
  tabs, trees and comboboxes follow the WAI-ARIA patterns described in §5.
- **Focus visible (2.4.7) and not obscured (2.4.11):** a 2 px `--focus-ring` (navy in light mode, yellow in
  dark mode) with a 2 px offset on every control (inset inside tables, menus and trees), shown for keyboard
  focus (`:focus-visible`), not after a mouse click (UI audit AX-03). Sticky top bar, table headers, form
  footers and toasts never cover the focused element: `scroll-padding` equals their height.
- **Names:** every control has a visible label or an accessible name; icon-only buttons have a tooltip and a
  name. Exception: the close button inside a dialog, side panel or drawer has the name "Close" (or "Close
  menu") but no tooltip, because a tooltip there would swallow the first Escape (feature 001, research
  R14); inside the drawer, items show text labels instead of tooltips.
- **Contrast:** text ≥ 4.5:1 (all text tokens pass on white, grey and their tints, §3.1, §4.1); control
  borders, focus rings and meaningful icons ≥ 3:1 (1.4.11). Yellow is never text and never the only sign of
  a state.
- **Colour is never the only signal (1.4.1):** status as a word, the current place as bold text plus the
  bar plus `aria-current`, links in sentences underlined.
- **Target size (2.5.8):** every target at least 24 × 24 px (§3.6); controls are 32–48 px.
- **Dragging (2.5.7):** nothing needs dragging; any future drag (reordering, calendar bars) has a button or
  menu alternative.
- **Consistent help (3.2.6):** help, when it exists, is always in the top bar at the same place.
- **Redundant entry (3.3.7):** values already given are offered again instead of typed twice (defaults from
  the parent, "From the campaign").
- **Accessible authentication (3.3.8):** sign-in allows paste and password managers; no puzzles.
- **Errors:** announced and linked from the summary; the focus moves to the summary on a failed save.
- **Motion and zoom:** respect reduced motion; usable at 200 % zoom and reflowing without sideways scrolling
  at 320 CSS px (1.4.10; 400 % zoom on a 1280 px screen); text spacing overrides do not break layouts
  (1.4.12).
- **Forced colours:** cards, inputs and the current-place marker keep visible borders in Windows
  high-contrast mode.
- **Landmarks:** skip link, one `header` (top bar), one `nav` (sidebar), one `main`, one `footer`; one `h1`
  per page.
- **Screen sizes:** works from 390 px phones to 1920 px desktops; no page scrolls sideways at any width.

---

## 10. Usability targets (used in success criteria)

| Task | Hive9 (estimate) | MAPP target |
|---|---|---|
| Create a plan | ~14 steps | ≤ 6 steps |
| Add a campaign | ~8 | ≤ 5 |
| Add an initiative | ~7 | ≤ 3 |
| Add an activity with one line item | ~32 | ≤ 15 |
| Submit an activity for approval | 4 + retries (one error per save) | ≤ 2, all problems shown at once |
| Approve from the approvals inbox | ~8 | ≤ 3 |
| Find an activity by name or ID | 4–10 | ≤ 2 |
| Change the owner of 20 activities | ~27 | ≤ 6 |
| Restore a deleted item | 4–7 | 1 (Undo) or ≤ 3 |

A *step* is one click, selection or field entry. Generic success criteria for specs:

- At least 90 % of first-time users create, edit and delete a <record> without help, each task within
  2 minutes (5 for long forms).
- Every problem with a form is shown in a single save attempt.
- 100 % of read-only fields and unavailable actions explain why.
- Users can tell a record's status and what is missing without opening another screen.

---

## 11. Consistency rules (binding checklist)

Each rule has an automatic check where one is practical; reviewers check the rest on the screenshots
listed in the implementation plan.

| # | Rule | How it is checked |
|---|---|---|
| 1 | **At most one yellow primary button per page or dialog.** A page is the page header and its content; each open dialog or side panel is its own scope. Destructive confirm buttons are red, not yellow; an empty state's button is secondary when the header holds the primary. | e2e: count visible primary buttons per scope ≤ 1 on every captured state; unit tests on `PageHeader`, `SidePanel`, dialogs |
| 2 | **Numbers right-aligned with tabular figures** in tables, totals and amount inputs; headers aligned with their values; "38%" without a space. | e2e measurement: right edge of every numeric cell equals its header's (± 1 px); unit tests on number formatters |
| 3 | **One page-header layout:** every page uses `PageHeader` (breadcrumb row, title, meta, actions in fixed slots); no other component renders an `<h1>`. | Unit test scanning `src/features` for `<h1`; e2e: title top at the same y (± 1 px) on every page |
| 4 | **One panel footer layout:** right-aligned, Cancel or Close (secondary) · secondary actions · primary last; destructive actions in the header's ⋯. Dialogs and form footers follow the same order. | `SidePanel` takes action props, not free content; unit tests |
| 5 | **One date format:** the person's format in text and inputs, ranges per D-216, times 24-hour; all through `lib/format.ts`. | Lint: no `toLocaleDateString` / `Intl.DateTimeFormat` outside `lib/`; formatter unit tests |
| 6 | **Status colour meanings:** each status word has one family (§4.1) through one mapping; the same word never shows in two colours; red only for danger and errors; yellow only for the primary action, the current place and the highlighted row. | Unit test over the mapping; e2e colour sampling of chips on the calendar legend and the plan tree |
| 7 | **One height per control size** (32 / 40 / 48); controls side by side share a size. | stylelint: `height`/`min-height` of controls only from `--control-*` |
| 8 | **Tokens only** for colour, type, spacing, radius, shadow; only the breakpoints of §2.2; every token pair of §3.1 and §4 passes its contrast threshold in both themes. | stylelint; a Vitest test that parses `tokens.css` and checks every listed pair |
| 9 | **One key–value layout** (`DescriptionList`, 200 px labels) and **one table standard** (§5.4). | Code review; e2e: value columns start at the same x within a page |
| 10 | **Names never break inside a word** in tables, trees and tags (`overflow-wrap: anywhere` is not allowed). | stylelint rule; e2e: no table row taller than 64 px on the captured states |
| 11 | **One money format per screen** (§4.4); empty values "—". | Formatter unit tests; e2e text checks on the activity page and Home |
| 12 | **Sentence case**, no upper-case text, normal letter spacing, text ≥ 12 px. | stylelint (`text-transform: uppercase`, `letter-spacing`, font sizes); e2e: no rendered text under 12 px |
| 13 | **One select style** (§5.5) and one date input. | Code review; e2e screenshot review |
| 14 | **Lists say their count the same way** ("4 of 6 plans") and record-state filters read "Active (n) · Inactive (n) · All (n)". | e2e text checks on each list page |
| 15 | **Nothing covers the focus or the next action:** toasts sit above sticky footers; sticky bars set `scroll-padding`. | e2e: the Save button is not covered after a create; axe |
| 16 | **A heading never repeats its parent's heading.** | e2e: no two consecutive headings with the same text |
| 17 | **No page scrolls sideways** at 390, 600, 960, 1280 and 1440 px wide. | e2e: `scrollWidth ≤ innerWidth` on every captured state |

---

## 12. Waiting for official Shell assets

These items use provisional, open stand-ins until Shell supplies the real thing; each switch is one token
or one component (see the recommendations report for the questions to Shell).

| Item | Now | When Shell supplies it |
|---|---|---|
| Heading typeface (Shell Heavy / Bold / Book, URW, proprietary) | Jost 600/700 (OFL) in `--font-heading` | Change `--font-heading` (and possibly `--font`) and self-host Shell's files under its licence |
| Pecten and trade-mark consent | MAPP's navy tile and star | Replace `Logo` and the favicon, following Shell's placement and clear-space rules |
| Official screen colour values | Live shell.com values: yellow `#FFC800`, red `#DD1D21`, secondaries | Update §3.1 and §4 tokens and re-run the contrast test |
| Shell's internal design system ("Shell DS") | This document | If mandatory for internal tools, its tokens and components replace ours where they differ |
| Icon set | lucide outline icons | Swap the icon imports |
| Internal accessibility checklist | WCAG 2.2 AA | Add its extra checks to §9 and the test suite |

---

## Change history

| Version | Date | Change |
|---|---|---|
| 1 | 18 Oct 2026 | Apple-inspired design language (D-005), feature 001 tokens (research R17) |
| 2 | 20 Oct 2026 | Shell visual language per the product-owner decisions of 20 Oct 2026 (D-310–D-314): Shell tokens light and dark with contrast values, Jost + Arial type scale, spacing, radius, elevation, control heights, breakpoints and widths; top bar and white sidebar with the yellow bar; one page header and five page templates; specs for every shared component; WCAG 2.2 AA; consistency checklist. §6–§8 and §10 kept, with §6.3 (one primary by role), §6.7 (set inactive) and §7 (rail, progress) added. |
