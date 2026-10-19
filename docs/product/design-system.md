# Design System — NorthStar

NorthStar looks and behaves like **ZenReach**: a clean, white, Apple-inspired interface with a left
sidebar, compact page headers, summary cards, rounded white cards, a single blue primary action and plain
words. Shell's Hive9 is the *functional* reference only — its screens, toolbars and dense panels are never
copied (decision D-005).

Reference screenshots: [../reference/zenreach-screenshots/](../reference/zenreach-screenshots/).
Exact ZenReach tokens: ZenReach specification §12.5–12.6
([../reference/zenreach-feature-specification.md](../reference/zenreach-feature-specification.md)).

---

## 1. Principles

1. **A beginner understands every screen.** One purpose per page, plain words, nothing unexplained.
2. **One way to do each thing.** The same list, form, details, delete and status patterns everywhere.
3. **Status and next step are always visible.** Every record shows where it is (status chip) and what is
   missing (completeness), with a button for the next action.
4. **Show, don't hide.** Unavailable actions stay visible and explain why; read-only values say where they
   come from.
5. **Safe.** Confirm destructive actions, protect unsaved changes, offer Undo, keep a recycle bin.
6. **Calm.** White space, one accent colour, colour used for meaning (status, level, money warnings).

---

## 2. Layout (from ZenReach)

```
┌───────────────┬──────────────────────────────────────────────────────────────┐
│ ◆ NorthStar   │  PLANS                                     (eyebrow, blue)   │
│               │  Plan with confidence.        (title + grey highlight)       │
│ ⌂ Home        │  Every plan, campaign and activity in one place.  [+ New plan]│
│ ▤ Plans       │                                                              │
│ ✓ Approvals   │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐  summary   │
│ ▦ All work    │  │ Plans 6 │ │ Draft 2 │ │ Planned │ │ Budget  │  cards     │
│               │  └─────────┘ └─────────┘ └─────────┘ └─────────┘            │
│ MORE          │  [All] [Draft] [Published]   🔍 Search…   Filters (2)        │
│ ⚙ Settings    │  ┌──────────────────────────────────────────────────────────┐│
│ ↺ Recycle bin │  │ table / tree / cards in a rounded white card             ││
│ ☰ Activity log│  └──────────────────────────────────────────────────────────┘│
│               │                                                              │
│ ‹ Collapse    │                                                              │
│ (AB) Abhi  🔔 ☾│                                                     (?) help │
└───────────────┴──────────────────────────────────────────────────────────────┘
```

| Element | Rule |
|---|---|
| **Sidebar** | Light grey (`--gray-bg`), 244 px; collapses to a 72 px icon rail with tooltips (remembered per user). Main group at top, a **MORE** group label, then secondary items. Footer: collapse toggle, account button (avatar + first name), notification bell, light/dark toggle. On phones it becomes a slide-in drawer. |
| **Page header** | Small blue upper-case *eyebrow* (12 px); title 22–26 px bold with an optional grey "highlight" phrase; one-line subtitle (13.5 px, secondary text); page actions on the right — at most **one** blue primary button. |
| **Summary cards** | A row of 3–6 white cards: label (13 px, secondary, with an ⓘ tip) and a large value (24–30 px, bold, coloured only when the colour carries meaning). |
| **Tabs / segmented filters** | Pill-shaped toggle buttons (ZenReach style). |
| **Content cards** | White, radius 16 px, 1 px faint border, very soft shadow; header row with a 15 px semibold title and 12.5 px subtitle. |
| **Width** | Content max-width 1120 px with 20 px side padding; tables may scroll sideways inside their card. |
| **Footer** | Small grey text: "© NorthStar — demo data" during this phase. |
| **Environment banner** | A thin yellow banner "Demo environment — sign-in and data are for demonstration only" on every page in this phase. |

Primary navigation for this phase: **Home · Plans · Approvals · All work** and, under MORE,
**Settings · Recycle bin · Activity log**. (Finance and Measure appear later.)

---

## 3. Visual tokens

| Token | Light | Dark | Use |
|---|---|---|---|
| `--page-bg` | `#ffffff` | `#0b0b0d` | Page background |
| `--surface` | `#ffffff` | `#1c1c1e` | Cards, dialogs |
| `--gray-bg` | `#f5f5f7` | `#161618` | Sidebar, table header, subtle fills |
| `--ink` | `#1d1d1f` | `#f5f5f7` | Main text |
| `--ink-2` | `#6e6e73` | `#a1a1a6` | Secondary text |
| `--muted` | `#86868b` | `#8a8a8e` | Hints, placeholders |
| `--hairline` | `#d2d2d7` | `#38383a` | Dividers, input borders |
| `--card-border` | `rgba(0,0,0,.07)` | `rgba(255,255,255,.1)` | Card borders |
| `--hover` | `rgba(0,0,0,.05)` | `rgba(255,255,255,.06)` | Background of a hovered item |
| `--blue` | `#0071e3` | `#2a91ff` | Selection, focus ring, eyebrows, active icons |
| `--link` | `#0066cc` | `#59a8ff` | Link text (readable on white and on `--gray-bg`) |
| `--button-primary-bg` | `#0071e3` | `#0071e3` | Primary button fill (white text stays readable in dark mode) |
| `--on-accent` | `#ffffff` | same | Text and icons on blue, red and brand-gradient surfaces |
| `--brand-gradient` | `linear-gradient(135deg,#2a78d6,#9f4bd6)` | same | Logo, progress bars |
| `--brand-gradient-strong` | the brand gradient darkened by 12 % | same | Surfaces with white text: sign-in header, avatars |
| `--chip` | `rgba(0,0,0,.05)` | `rgba(255,255,255,.09)` | Neutral chips |
| `--overlay` | `rgba(0,0,0,.3)` | `rgba(0,0,0,.6)` | Dims the page behind dialogs, side panels and the menu drawer |
| `--tooltip-bg` / `--tooltip-text` | `#1d1d1f` / `#ffffff` | `#3a3a3c` / `#f5f5f7` | Tooltip bubble and its text |

- **Font:** token `--font`, the system font stack
  (`-apple-system, BlinkMacSystemFont, "Segoe UI", Inter, system-ui, sans-serif`); numbers use tabular
  figures. Sentence case everywhere.
- **Radius:** cards 16 px, inputs and buttons 10 px, chips 8 px, pills 999 px.
- **Spacing:** 4 px grid; cards 20–24 px padding.
- **Motion:** short fades (≤ 0.3 s); none when the user prefers reduced motion.
- **Dark mode:** supported via the tokens. On first use NorthStar follows the computer's setting, then
  remembers each person's choice (decision D-038).

---

## 4. Status, level and money colours

Status is always **word + colour (+ icon where space allows)**; colour is never the only signal.

| Family | Colour | Statuses |
|---|---|---|
| Neutral | grey `#86868b` | Draft, Created, Not started, No activities, Not requested, Inactive (Settings records, people) |
| In review | amber `#c77700` | Submitted, In progress (approval progress), Waiting for activity approval |
| Positive | green `#0ca30c` | Published, Approved, All approved, All filled in (completeness), Ready to send to GSAP, Active (Settings records, people) |
| Active | blue `#0071e3` | In progress (activity running), Running |
| Done | slate `#5b6b7a` | Complete (activity finished), Finished |
| Problem | red `#d03b3b` | Declined, Incomplete (n missing), over-allocated budget, negative balance |

Chips are 11–12 px semibold text on a 12 % tint of the colour (ZenReach `color + 1c` alpha). The chip
**text** uses a darker shade of the colour (lighter in dark mode) so it meets WCAG AA; the shades are the
`--status-<tone>-text` tokens (feature 001, research R17).

**Level colours** (a 4 px bar beside the name in trees and a small level tag): Plan blue `#2a78d6`,
Campaign violet `#4a3aa7`, Initiative aqua `#1baf7a`, Activity magenta `#e87ba4`, Line item grey `#86868b`.

**Money:** right-aligned, tabular figures, always with currency (`$12,500.00` or `12,500.00 USD`);
compact form in cards (`$1.25M`) with the full value on hover; negative values in red with a minus sign;
"—" for empty, never 0.

**Progress bars** (budget used, approval progress): 6–10 px, rounded, brand gradient fill on a `--chip`
track; red fill above 100 %.

---

## 5. Components

| Component | Behaviour |
|---|---|
| **Button** | Primary = blue filled pill; secondary = outlined; danger = red text/outline; labels are verbs ("Create campaign", "Submit for approval"), never "OK" or "Yes". |
| **Status chip** | §4. |
| **Completeness chip** | "All filled in" (green) or "3 fields needed" (red) — click opens the list of missing fields with links. |
| **InfoTip (ⓘ)** | Small muted icon next to labels; hover/focus/tap shows a one- or two-sentence plain-English definition from the glossary. |
| **Person picker** | Search by name, e-mail or job title; shows avatar (gradient initials), name and title. |
| **Choice / multi-choice** | Searchable when more than 10 options; selected values as chips. |
| **Split editor** | For "multiple choice with % split": table of value · share %, running total "100.00 % — complete" or "90.00 % — 10.00 % left", **Split evenly** button. |
| **Money input** | Currency shown inside the field; accepts `12500`, `12,500.00`, `12.5k`. |
| **Date range picker** | One control for start and end; shows the allowed range from the parent. |
| **Tree table** | Expand/collapse rows, level bar and tag, name as link, status chip, money columns right-aligned, totals row, **Expand all / Collapse all**. |
| **Side panel (drawer)** | 480–560 px from the right for viewing/editing short records; has its own header, scrollable body and sticky footer. |
| **Dialog** | For confirmations and short forms; Cancel focused for destructive actions; the danger button names the action. |
| **Toast** | Bottom-centre, 4 s, names the record ("Campaign 'Helix launch' created"), with **View** or **Undo** where useful. |
| **Empty state** | Icon, one sentence explaining the thing, and the create button ("Create your first plan"). |
| **Skeleton** | Grey placeholder shapes while loading; old data stays visible while refreshing. |
| **Avatar** | Circle with initials on the brand gradient. |
| **Floating help (later)** | ZenReach's assistant button position (bottom-right) is reserved; not built in this phase. |

---

## 6. The standard CRUD pattern

Every user-managed record follows this pattern. Specs refer to it as "the standard CRUD pattern".

### 6.1 List

1. Page header with the plural name, a one-line description and **New <thing>** (primary, top right).
2. Toolbar: search box that says what it searches ("Search name or ID"), pill filters for status, a
   **Filters** button for the rest, removable filter chips, and a result count ("12 of 40").
3. Table (or cards for plans): name as a link, status chip, key facts, money with currency, a row
   **⋯ Actions** menu (View · Edit · Duplicate · record-specific actions · Delete last, separated).
4. Selecting a row opens the record's **details view** — never straight into edit.
5. Empty state: "No plans yet — create your first plan" with the create button; when filters hide
   everything: "Nothing matches these filters" with **Clear filters**.

### 6.2 Create

- Entry: the list's **New <thing>** button, or **Add <child>** on the parent (the parent is pre-filled).
- Short records (campaign, initiative, settings items) open in a **side panel**; long records (plan,
  activity, line item) open on a **full page** with sections.
- One column; sections in a logical order; defaults pre-filled and labelled ("From the campaign");
  markings shown as words: **Required**, **Needed to publish**, **Needed to submit**.
- Sticky footer: **Save** (primary) · **Save and add another** (for children) · **Cancel** (asks before
  discarding typed values).
- After saving: toast "Initiative 'Digital & social' created — View"; the new row is highlighted.

### 6.3 View (details)

- Header: breadcrumb (Plans › FY2026 Lubricants › Helix launch), record name, level tag, **status chip**,
  **completeness chip**, ID, owner, dates, money; "Created by … on …, last changed by … on …".
- Actions: **Edit** (primary) · the next status action as a secondary button (**Publish**, **Submit for
  approval**, **Approve**) · **⋯ More** (Duplicate, Copy to…, Move to…, View history, Delete).
- Sections/tabs: Overview (read-only fields; empty values show "—"), child records with counts, Approval
  (activities), History.

### 6.4 Edit

- **Edit** turns the details into a form with the same layout.
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

### 6.6 Locked and read-only

Nothing is greyed out without a reason next to it:

| Situation | Shown as |
|---|---|
| Calculated | Small "Calculated" label and ⓘ with the formula |
| From finance/GSAP later | "Filled by GSAP later" / "From finance (coming later)" |
| Fixed after save | Lock icon + "Year can't change once the plan is saved — create a new plan for another year." |
| Irreversible choice | Warning when ticked: "Ongoing can't be undone after saving." |
| Waiting for approval | Banner: "Submitted for approval — changing planned cost or dates will restart approval." |
| Action not yet possible | The button stays visible; clicking it explains: "Publish needs 4 fields — view list". |

---

## 7. Forms

- One column, max 720 px wide, section headings in the order users think: *Basics → Dates & cost →
  Shell attributes → Goals → More details*.
- Dependent fields stay visible but disabled with a hint until their parent is set ("Choose teams first —
  the owner must belong to one of the plan's teams"). When only one option fits, it is filled in and
  said so ("Region set to 'Retail – China', the only region for this cluster").
- Changing a parent value that invalidates child values lists what will be cleared before saving.
- Attribute groups that depend on the business unit say why they are shown ("Shown because the plan's
  business unit includes Lubricants").
- Validation messages say what is wrong and how to fix it: "End date must be on or after the start date
  (12 Mar 2026)." — never codes or technical text.

---

## 8. Words on screen

Shell business terms are kept: Plan, Campaign, Initiative, Activity, Line item, SWBS, GL code, BCPIR,
SP&A, AOP, SMP, INQ, MQL. Hive9 interface terms are replaced:

| Hive9 term | NorthStar label |
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

Buttons use verbs; messages are short, friendly and specific; dates follow the user's format
(default `DD MMM YYYY`, e.g. 12 Mar 2026). A date range shown as a value is written one way everywhere
(D-216): in `DD MMM YYYY` with the year once when both dates fall in the same year ("01 Mar – 31 May
2026"), otherwise both dates in full ("01 Dec 2026 – 31 Jan 2027"); in the other formats both dates in
full ("01/03/2026 – 31/05/2026"); an ongoing activity reads "Ongoing from 01 Mar 2026".

---

## 9. Accessibility (WCAG 2.1 AA basics)

- Everything works with the keyboard; focus is visible (2 px blue ring with 2 px offset).
- Every control has a visible label or accessible name; icon-only buttons have a tooltip and a name.
  Exception: the close button inside a dialog, side panel or drawer has the name "Close" (or "Close
  menu") but no tooltip, because a tooltip there would swallow the first Escape (feature 001, research
  R14); inside the mobile drawer, footer controls show text labels instead of tooltips.
- Text contrast ≥ 4.5:1; status never relies on colour alone. Small text uses `--ink`, `--ink-2` or
  `--link`; `--muted` is only for icons, placeholders and decoration.
- Errors are announced and linked from the summary; focus moves to the summary on a failed save.
- Respect reduced motion; usable at 200 % zoom; usable down to 1280 × 720 and on tablets.

---

## 10. Usability targets (used in success criteria)

| Task | Hive9 (estimate) | NorthStar target |
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
