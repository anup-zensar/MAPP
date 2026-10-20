# Design System — NorthStar

NorthStar **behaves** like ZenReach — a left sidebar, compact page headers, summary cards, the standard
CRUD pattern and plain words — and **looks** like a Shell enterprise tool: calm, structured and
symmetric, white cards on a light grey page, Shell yellow for the one main action and for "you are
here", red only for danger (product owner decisions of 20 October 2026, §2–§4). Shell's Hive9 is the
*functional* reference only — its screens, toolbars and dense panels are never copied (decision D-005).

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
6. **Calm.** White space, one accent colour (Shell yellow, for acting and for "you are here"), colour used
   for meaning (status, level, money warnings).

---

## 2. Layout (Shell enterprise look)

The visual language follows Shell's own web design system (product owner decisions of 20 October 2026,
which replace the visual part of D-005 and D-022; research in
[../reports/ux-research-shell-brand.md](../reports/ux-research-shell-brand.md) and
[../reports/ux-research-shell-ui-patterns.md](../reports/ux-research-shell-ui-patterns.md)). ZenReach's
*behaviour* (§6–§8) stays.

```
┌──────────────────────────────────────────────────────────────────────────────────────┐
│ [★] NorthStar                                                     ☾   🔔3   (PR) Priya ▾│ top bar 56 px, white
├────────────────┬─────────────────────────────────────────────────────────────────────┤
│▌⌂ Home         │ Demo environment — sign-in and data are for demonstration only (navy) │
│  ▤ Plans       │ Plans › FY2026 Lubricants                          (breadcrumb, 12 px) │
│  ✓ Approvals 3 │ PLAN                                                                  │
│  ▦ All work    │ FY2026 Lubricants Europe          [More ▾] [Unpublish] [Edit]          │ title 28 px Jost
│                │ Published · All filled in · P-1001 · Owner                             │
│  More          │ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐                    │
│  ⚙ Settings    │ │ Budget   │ │ Planned  │ │ Remaining│ │ Used     │  white cards on     │
│  ↺ Recycle bin │ └──────────┘ └──────────┘ └──────────┘ └──────────┘  the #F5F5F5 page  │
│  ☰ Activity log│ Overview   Activities (12)   History        (underline tabs, yellow bar)│
│                │ ┌──────────────────────────────────────────────────────────────────┐ │
│  « Collapse    │ │ table: 40 px rows, grey header band, money right-aligned          │ │
└────────────────┴─────────────────────────────────────────────────────────────────────┘
```

| Element | Rule |
|---|---|
| **Top bar** | White, 56 px, across the whole width, stays at the top. Left: the NorthStar mark (a white star on a navy tile; **never the Shell pecten**, which needs Shell's consent) and the name in Jost. Right: the light/dark switch, the notification bell and the account menu (avatar, first name; name, e-mail, job title, My profile, Sign out). Below 960 px, **Open menu** sits left of the mark. No control that does nothing (no help link until there is help to link to). |
| **Sidebar** | White, 240 px, under the top bar; collapses to a 64 px icon rail with tooltips (remembered per person). Items 40 px, 20 px icons, 14 px ink text. **You are here:** a 4 px yellow bar at the start, bold text and a pale-yellow fill, plus `aria-current="page"`. A "More" group label (12 px bold, sentence case). Collapse at the bottom. Below 960 px it opens as a drawer from Open menu. |
| **Page** | Light grey `#F5F5F5` canvas; content centred, at most 1300 px wide, 24 px gutters (16 px below 600 px). Breakpoints: 600 and 960 px only (1280 px for a details page's side column). |
| **Page header** (`PageHeader`, one layout for list, details and form pages) | Breadcrumb (12 px) on every page below the top level; then the level tag or a small sentence-case eyebrow (never both, and no eyebrow under a breadcrumb or equal to the title); the title (Jost 700, 28 px) with the actions on its right: **More** menu, secondary buttons, then the **one** yellow primary button rightmost; facts and chips under the title; a one-line description. |
| **Summary cards** | A row of white cards (8 px corners, 20 px padding): 12 px bold label with an ⓘ, a 28 px Jost value with tabular figures, coloured only when the colour means something. Labels get room for two lines, so values line up. |
| **Tabs** | Underline tabs, 48 px high: the chosen tab is bold with a 4 px yellow bar. Segmented filters (32 px, joined, chosen = ink fill) are for filtering a list only. |
| **Content cards** | White, 8 px corners, a hairline edge, no shadow; header (min 64 px) with an 18 px Jost title on the left centred against its actions on the right; 24 px padding (16 px on phones). |
| **Footer** | One 12 px line: "© NorthStar — demo data". |
| **Environment banner** | Shell's navy system message (white bold 12 px, 32 px) at the top of the content column on every page in this phase. |

Primary navigation for this phase: **Home · Plans · Approvals · All work** and, under More,
**Settings · Recycle bin · Activity log**. (Finance and Measure appear later.)

---

## 3. Visual tokens

Every value lives in `frontend/src/styles/tokens.css`; components use `var(--token)` only. Dark mode
uses Shell's own dark palette, follows the computer's setting first, then the person's switch (D-038).
Every text token is 4.5:1 or more on `--page-bg`, `--surface` and `--gray-bg` in both themes; control
borders, the focus ring and status dots are 3:1 or more (WCAG 2.2 AA).

| Token | Light | Dark | Use |
|---|---|---|---|
| `--page-bg` | `#F5F5F5` | `#14191F` | The page canvas |
| `--surface` | `#FFFFFF` | `#1D232A` | Cards, panels, dialogs, menus, top bar, sidebar |
| `--gray-bg` | `#F5F5F5` | `#252C34` | Table header band, subtle fills inside white surfaces |
| `--ink` / `--ink-2` / `--muted` | `#404040` / `#595959` / `#6E6E6E` | `#E0E0E0` / `#B0B8C1` / `#949FAB` | Text, secondary text, hints (all readable as small text) |
| `--hairline` / `--row-divider` / `--card-border` | `#D9D9D9` / `#E8E8E8` / `#E0E0E0` | `#38414B` / `#2C343C` / `#2F3741` | Dividers, table rows, card edges (decoration) |
| `--control-border` | `#8C8C8C` (3.4:1) | `#76818D` | Inputs, selects, check boxes, outlined buttons |
| `--brand-yellow` / `--on-brand-yellow` | `#FFC800` / `#404040` | `#D6A100` / `#14191F` | The primary button and the "you are here" bar; **a fill only, never text** |
| `--brand-yellow-tint` | `#FFF4CC` | `#3B3723` | Current sidebar item, a row just created or moved |
| `--focus-ring` | `#092949` (navy) | `#FFC800` | The 2 px focus ring with a 2 px gap on every control |
| `--blue` / `--link` | `#003C88` | `#8CB4E6` | Links, checked controls, selection tints, progress bars |
| `--danger-bg` | `#DD1D21` (Shell red) | `#C42B2F` | Destructive confirm buttons only (white text) |
| `--info` / `--success` / `--warning` / `--danger` (+ `-text`) | `#336094` / `#008557` / `#C77700` / `#DD1D21` | lighter shades | Feedback colours, one meaning each |
| `--banner-bg` | `#092949` | `#092949` | Environment banner |
| `--overlay` | black 45 % | black 60 % | Behind dialogs, side panels and the drawer |

- **Type:** Jost 600/700 (open licence, standing in for Shell's own typeface) for headings and big
  numbers (`--font-heading`); Shell's body stack `Arial, Roboto, Helvetica, sans-serif` for everything
  else (`--font`). Seven sizes: 12 (minimum), **14 (base)**, 16, 18, 20, 28 and 32 px (`--text-xs` …
  `--text-3xl`). Weights 400 and 700 (Jost headings 600). Sentence case, no letter spacing, no upper-case
  labels. Numbers use tabular figures.
- **Spacing:** the 4 px grid, `--space-1` (4 px) … `--space-16` (64 px); 1–2 px only for borders.
  Label → control 8, between fields 20, card padding 24, between cards 20.
- **Control heights:** 32 px (toolbars, filters, chips, table buttons), **40 px** (default buttons,
  inputs, selects, search), 48 px (sign-in, touch). One height per size everywhere; inputs use 16 px
  text below 960 px.
- **Radius:** 4 px (chips, tags, tooltips, menu items), 8 px (buttons, inputs, menus, cards), 12 px
  (dialogs), 16 px; pill shapes only for avatars, count badges and switches.
- **Elevation:** cards are flat (a hairline on the grey page); menus and toasts `--shadow-md`, dialogs
  and panels `--shadow-lg`, a clickable card lifts with `--shadow-lift`.
- **Motion:** 0.25 s; none when the person prefers reduced motion.

---

## 4. Status, level and money colours

Status is always **word + colour (+ icon where space allows)**; colour is never the only signal. One
meaning per colour:

| Family | Colour | Meaning | Statuses |
|---|---|---|---|
| Neutral | grey | Not started or not in use | Draft, Created, Not started, No activities, Not requested, Inactive |
| Review | amber (`--warning`) | Waits for a decision | Submitted, Waiting for activity approval, approval In progress |
| Positive | green (`--success`) | Approved or ready | Published, Approved, All approved, All filled in, Active |
| Active | blue (`--info`) | Running | In progress (activity running), Running |
| Done | slate | Finished | Complete, Finished |
| Problem | red (`--danger`) | Something is wrong | Declined, Incomplete (n missing), over-allocated, negative balance |

Chips (`StatusChip`, `CompletenessChip`) are 24 px high, 12 px bold, 4 px corners: the word in the
family's `-text` shade on a 12 % tint of its colour (4.5:1 or more in both themes). Yellow is never a
status colour: it means "act here" or "you are here".

**Level colours** (a 4 px bar beside the name in trees, and the level tag): Plan dark blue `#003C88`,
Campaign purple `#87217D`, Initiative teal `#008192`, Activity magenta `#A3195B`, Line item grey
`#6E6E6E` (lighter shades in dark mode).

**Money:** right-aligned, tabular figures, never broken over two lines, always with currency
(`$12,500.00` or `12,500.00 USD`); compact form in cards (`$1.25M`) with the full value on hover;
negative values in red with a minus sign; "—" for empty, never 0.

**Progress bars** (budget used, approval progress): 8 px, 4 px corners, dark blue on a grey track; red
above 100 %.

---

## 5. Components

| Component | Behaviour |
|---|---|
| **Button** | `primary` = Shell yellow with dark text, **at most one per page, panel or dialog**; `secondary` = white, outlined; `tertiary` = no frame (icon buttons); `danger` = Shell red with white text, for the confirm button of a destructive dialog only. Sizes 32 / 40 / 48 px. Labels are verbs ("Create campaign", "Submit for approval"), never "OK" or "Yes". |
| **Status chip** | §4. |
| **Completeness chip** | "All filled in" (green) or "3 fields needed" (red) — click opens the list of missing fields with links. |
| **InfoTip (ⓘ)** | A 24 px target next to labels; hover/focus/tap shows a one- or two-sentence plain-English definition from the glossary. |
| **Fields** | Label (14 px bold) above, the marking as a word, hint and problem under the control (12 px). Inputs, selects, comboboxes, money, decimal and search fields are 40 px, 8 px corners, a 3:1 border; no pill shapes. |
| **Person picker** | Search by name, e-mail or job title; shows avatar (initials on dark blue), name and title. |
| **Choice / multi-choice** | Searchable when more than 10 options; selected values as chips. |
| **Split editor** | For "multiple choice with % split": table of value · share %, running total "100.00 % — complete" or "90.00 % — 10.00 % left", **Split evenly** button. |
| **Money input** | Currency shown inside the field; accepts `12500`, `12,500.00`, `12.5k`. |
| **Date range picker** | One control for start and end; shows the allowed range from the parent. |
| **Tables and tree table** | A 40 px grey header band with 12 px bold headings; 40 px rows, 8 × 12 px cells, hairlines between rows; money right-aligned and never wrapped; names wrap between words only, never letter by letter; columns can set a minimum width; wide tables scroll sideways inside their box (with an edge shadow), never the page. Tree: expand/collapse, level bar and tag, name as link, ⋯ menu per row, a bold totals row under a 2 px ink line, **Expand all / Collapse all**. |
| **Side panel (drawer)** | 480–560 px from the right for viewing/editing short records; header (64 px, 18 px Jost title, Close), scrollable body, sticky footer. **One footer layout:** the primary rightmost, Cancel next to it, destructive actions in the More menu. |
| **Dialog** | 512 px, 12 px corners, for confirmations and short forms; Cancel then the action on the right; Cancel focused for destructive actions; the danger button names the action. |
| **Menu** | ⋯ on rows and cards ("Actions for <name>"), **More ▾** in page headers; items 40 px; Delete last, after a line, in red words. |
| **Toast** | Bottom-centre, white with a 4 px ink bar, 4 s, names the record ("Campaign 'Helix launch' created"), with **View** or **Undo** where useful. |
| **Empty state** | Icon in a grey circle, a 20 px Jost title, one sentence and the create button ("Create your first plan"). |
| **Skeleton** | Grey placeholder shapes while loading; old data stays visible while refreshing. |
| **Avatar** | Initials (12 px bold, white) in a dark-blue circle. |
| **Floating help (later)** | Not built in this phase. |

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

## 9. Accessibility (WCAG 2.2 AA)

- Everything works with the keyboard; focus is visible (2 px navy ring with a 2 px gap; yellow in dark
  mode). Targets are at least 24 × 24 px (WCAG 2.2, 2.5.8).
- Every control has a visible label or accessible name; icon-only buttons have a tooltip and a name.
  Exception: the close button inside a dialog, side panel or drawer has the name "Close" (or "Close
  menu") but no tooltip, because a tooltip there would swallow the first Escape (feature 001, research
  R14); inside the mobile drawer, footer controls show text labels instead of tooltips.
- Text contrast ≥ 4.5:1 and control borders ≥ 3:1 in both themes; status never relies on colour alone;
  yellow is never text or a thin line on a light background.
- Errors are announced and linked from the summary; focus moves to the summary on a failed save.
- Respect reduced motion; usable at 200 % zoom; no page wider than the screen at 390 px; usable from
  phones (390 px) to 1440 px and wider.

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
