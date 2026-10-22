# UX/UI research findings and design recommendations

| | |
|---|---|
| **Deliverable** | UX/UI research findings and design recommendations |
| **For** | The Shell client team and the NorthStar product owner |
| **Date** | 20 October 2026 |
| **Status** | Recommendations agreed with the product owner on 20 October 2026 (decisions in §3, recorded as D-310–D-314); questions for Shell open (§8) |
| **Built on** | [Shell brand research](ux-research-shell-brand.md) · [Shell UI patterns research](ux-research-shell-ui-patterns.md) · [UI audit, page by page](ui-audit.md) (97 findings) · [Functional review](functional-review.md) (25 findings) |
| **Leads to** | [Design system, version 2](../product/design-system.md) (binding) · [UI/UX implementation plan](ui-ux-implementation-plan.md) |

---

## Summary

NorthStar's Plan module works from start to finish, but it does not yet look like a Shell tool or like a
finished product. We studied how Shell's own digital products look and behave, audited every NorthStar
screen at desktop and phone size in light and dark mode, and reviewed the main journeys of a planner, an
approver and an administrator. Our findings and recommendations:

1. **Shell's websites share one design system with concrete values.** Grey ink text (`#404040`/`#4A4A4A`),
   white and light-grey surfaces (`#F5F5F5`), **Shell yellow `#FFC800` only where you act and where you
   are**, Shell red almost never in the interface, a 4 px spacing grid, 12 columns, a 1300 px page, two
   breakpoints, three control heights and four corner radii. Business and corporate pages use less colour
   than consumer pages; an internal tool should follow the business look.
2. **NorthStar today looks like an Apple product, not a Shell one** (Apple blue, a blue-to-purple gradient,
   the Apple/Segoe font stack). Nothing on screen says Shell.
3. **The "unfinished" feel has measurable causes:** four layout bugs that break key screens (owner names one
   letter per line in the plan tree; nine pages wider than a phone), every feature building its own header,
   footer and table, and no spacing or type scale (15 font sizes, 7 control heights, 11 media queries at 8
   widths).
4. **The product owner decided on 20 October 2026** to give NorthStar the look of a Shell enterprise tool:
   yellow primary buttons with dark text, at most one per page; a light-grey canvas with white cards; Jost
   headings and Arial body text standing in for Shell's own font; no Shell logo until Shell consents; WCAG
   2.2 AA; red only for danger and errors (§3; decision log D-310–D-314).
5. **We recommend a small, strict system:** seven text sizes, one 4 px spacing scale, three control heights,
   four radii, one page header, one panel footer, one table standard, one date format and one meaning per
   status colour (§5). It is written down as [design system version 2](../product/design-system.md).
6. **The work is mostly foundation, then a page-by-page pass.** Tokens, fonts and a dozen shared
   components remove most findings at once; the [implementation plan](ui-ux-implementation-plan.md) then
   applies them area by area, with screenshots at 1440 and 390 px, light and dark, and automated WCAG 2.2 AA
   checks as the acceptance gate.
7. **Some things must wait for Shell:** the Shell typeface, the pecten, official colour values and, above
   all, whether Shell has an internal design system for staff tools that NorthStar must use (§7, §8).

---

## 1. What we studied

| Study | What it covered | How |
|---|---|---|
| [Shell brand research](ux-research-shell-brand.md) | Colour tokens, logo rules, fonts, imagery, tone, accessibility statement, consumer vs business pages; 17 Shell pages | Computed styles of every visible element at 1440 × 900 and 390 × 844, Shell's shared theme files, font metadata, Shell's trade-mark consent process |
| [Shell UI patterns research](ux-research-shell-ui-patterns.md) | Type scale, grid, spacing, components, navigation, the Fleet Hub B2B portal, the Shell Fleet App; 12 products | Shell's component source, computed styles, keyboard focus walk, font comparison |
| [UI audit](ui-audit.md) | 61 NorthStar pages and states as four demo people | 310 screenshots, layout measurements, axe on 121 page-and-theme checks |
| [Functional review](functional-review.md) | The planner, approver and administrator journeys of features 001–011 | Scripted browser journeys, API calls, a trial merge of feature 011 |

The Shell sites were captured on 20 October 2026; values can change when Shell redeploys. Shell's own
brand book is behind a sign-in, so official values still need Shell's confirmation (§8).

---

## 2. Findings

### 2.1 What Shell's digital products look like

| Aspect | What Shell does | Evidence |
|---|---|---|
| Colour | One grey ink for all text, links and icons (8.9:1 on white); white and `#F5F5F5` surfaces; 8–10 colours per page. Yellow fills the main button (dark text, never white) and marks the current menu item and tab with a 4 px bar. Red appears in no interface element on 15 of 17 pages measured. | Brand research §3.1, §3.4 |
| Type | Headings in Shell's proprietary "Shell Heavy" (URW, all rights reserved); everything else Arial 14 px. Six styles cover almost all text: Arial 12 / 14 / 14 bold and headings at 20, 28 and 32 px. | Patterns research §3; brand §5 |
| Grid and sizes | 4 px spacing unit, 12 columns with 20 px gaps, 1300 px page, breakpoints at 600 and 960 px, controls 32 / 40 / 48 px, corners 4 / 8 / 12 / 16 px, no pill shapes, no shadows on cards. | Patterns §4, §5 |
| Wayfinding | The same signature everywhere: a 4 px yellow bar under the current section, audience and tab, plus `aria-current`; breadcrumbs that collapse to the parent on phones. | Patterns §6 |
| Business tools | Shell Fleet Hub, the closest Shell product to NorthStar, has a left navigation, page actions, data tables, activity logs and confirmation dialogs, but an older visual style (square corners, dark-blue buttons, upper-case labels). | Patterns §6.2 |
| Tone | Short, plain, sentence-case headlines; verb-first buttons; calm corporate copy. | Brand §2 |
| Accessibility | Shell publicly commits to WCAG 2.2 A and AA, 400 % zoom and dark mode that follows the system setting. Its own sites rely on the browser's thin default focus outline and do not handle reduced motion. | Brand §9; patterns §7 |
| Logo | The pecten is a registered trade mark; partners need yearly consent through Brand Central and artwork approval from `Askbrand@shell.com`. | Brand §4 |

### 2.2 Where NorthStar stands

**What already works and must be kept** (UI audit "What already works"):

- One token file and a dark mode that passes automated checks on all 60 dark pages.
- The CRUD language: "Required / Needed to publish / Needed to submit", error summaries that link to the
  fields, delete dialogs that say what goes with the record.
- Empty states with a clear next action; the All work grid (right-aligned tabular numbers, totals, level
  bars); a visible focus ring and reduced-motion support — both better than Shell's own sites.
- Every business rule the functional review tried held; forms show all errors at once.

**What makes it look unfinished:**

| Root cause | Effect on screen | Findings |
|---|---|---|
| Four layout bugs | Owner names one letter per line in the plan tree (a plan with 13 records is 3,000 px tall); team pills split per letter in the Plans table; 9 pages wider than a phone (up to 1,099 px on a 390 px screen); side panels opening off-screen with Close unreachable; Home amounts floating mid-card | UI audit X-01, X-02, H-01, C-01; functional review P-01, X-01 |
| Each feature built its own parts | 5 page-header layouts (titles at 4 different heights), 5 side-panel footers, 3 key–value layouts, 3 select styles, 2 table-header styles, numbers aligned differently from table to table, two blue primary buttons on the activity page | UI audit X-03 to X-08, X-15, X-18 |
| No spacing or type scale | 15 font sizes (10.5–28 px), 15 corner radii, 7 control heights side by side (26–38 px), off-grid spacing, 11 media queries at 8 widths | UI audit X-10; patterns §8 |
| Words and numbers disagree | "In progress" means two things; "$500.00" and "500.00 USD" on one page; "Nothing" shown three ways; the same status word in two colours on the calendar | Functional review A-02, P-07, X-08; UI audit X-14, AC-04 |
| Dead ends | Recycle bin, Activity log, View history and Recent changes lead to "Coming soon" while the delete dialog promises a restore | UI audit A-01; functional review P-02, X-03 |
| Not Shell | Apple blue `#0071e3`, a blue-to-purple gradient, the Apple/Segoe font stack, input borders at 1.5:1 contrast (WCAG asks 3:1) | Brand §10; patterns §7, §8 |

---

## 3. Product-owner decisions (20 October 2026)

| # | Decision | Evidence | What it means in the design system |
|---|---|---|---|
| PO-1 (D-310) | **Primary action buttons in Shell yellow `#FFC800` with dark text** (`#404040` or `#4A4A4A`), as on shell.com — **at most one yellow primary button per page or dialog** | Yellow is Shell's "act here" colour on every current Shell site; ink on yellow is 6.67:1; Fleet Hub's dark blue belongs to an older generation (patterns §2, §10 Q2) | Tokens `--button-primary-*`; rule 1 of §11; one primary chosen by the person's role (§6.3) |
| PO-2 (D-311) | **Light-grey page background `#F5F5F5` with white cards and panels** | Shell's paper1/paper2 pair; white cards on grey bands (patterns §4); answers patterns §10 Q3 | `--page-bg` grey; cards, panels, dialogs, top bar and sidebar white; cards have no shadow |
| PO-3 (D-312) | **Jost 600/700 for headings** (open licence, `@fontsource/jost` 5.3.0, OFL-1.1) and **Arial, Roboto, Helvetica, sans-serif for body text**, standing in for Shell's proprietary font; **no Shell pecten or logo** (needs Shell's consent) — NorthStar keeps its own mark | Jost matches Shell Heavy's shape most closely (x/cap 0.66 vs 0.65; patterns §3.3, image 08); Arial is Shell's own body stack; pecten consent process (brand §4) | `--font-heading`, `--font`, self-hosted fonts; Logo redrawn without gradient; §12 lists the switch-over |
| PO-4 (D-313) | **Accessibility target WCAG 2.2 AA** | Shell's public statement covers internal sites (brand §9) | §9 of the design system; axe with `wcag22aa` |
| PO-5 (D-314) | **Red only for danger and errors** | Red appears in no interface element on 15 of 17 Shell pages (brand §3.4) | Red for destructive confirm buttons, errors and the problem status family; amber for "waiting"; never decoration |

These decisions, recorded as D-310–D-314 in the decision log, replace the **visual** part of decision
D-005 (the version 1 Apple-inspired look). The functional part stays: NorthStar's layout and interaction
patterns (design system §2, §6–§8) are kept and Hive9's screens are not copied. Constitution 1.1.0
amends principle III to match (one yellow primary action, WCAG 2.2 AA), replacing "one blue primary
action" and "WCAG 2.1 AA".

---

## 4. Design principles for NorthStar as a Shell enterprise tool

| Principle | What it means on screen |
|---|---|
| **Shell corporate, not Shell consumer** | White and grey, grey ink, yellow only where you act and where you are. No promotional panels, photography, sparkles or yellow areas in working screens. |
| **Calm** | Colour carries meaning or it is not used. One accent. No gradients, no shadows on cards, no movement on hover. |
| **Structured** | Every page is one of five templates (list, record, form, settings section, dashboard) with one page header. Cards group content; tables follow one standard. |
| **Symmetric** | One left edge and one right edge on every page; titles at the same height on every page; equal-height cards in a row; numbers exactly under their headers; controls side by side at the same height. |
| **One of everything** | One ink, one accent, one height per control size, one radius per object type, one spacing scale, one date format, one money format per screen, one meaning per status colour. |
| **Obvious and safe** (kept) | One purpose per screen, plain words, status and next step visible, nothing greyed out without a reason, confirm before destroying, Undo and a recycle bin. |
| **Accessible beyond Shell's own sites** | A strong custom focus ring, reduced motion, error summaries with links, status never by colour alone, 24 px minimum targets. |
| **Ready for Shell's assets** | Every provisional choice (font, logo, colour values) is one token or one component, so Shell's official assets drop in without a redesign. |

---

## 5. Recommendations

Each recommendation names its evidence and, where one applies, the product-owner decision (PO-1–PO-5
are D-310–D-314, §3). The exact values are in [design system version 2](../product/design-system.md).

### 5.1 Colour

| Recommendation | Evidence | Decision |
|---|---|---|
| Replace Apple blue and the gradient with Shell's system: ink `#404040` text (10.4:1 on white), secondary text `#595959` (7.0:1), muted `#6E6E6E` (5.1:1, now safe for small text). | Brand §10.2; patterns §9.2 | PO-1, PO-2 |
| Yellow `#FFC800` as a fill only: the one primary button (dark text), the 4 px current-place bar, the skip link, the highlighted-row tint. Never as text, a focus ring on white, or a warning. | Yellow on white is 1.55:1 (brand §9) | PO-1 |
| Red `#DD1D21` only for the destructive confirm button, errors and the "problem" status family; amber for "waiting"; Shell's secondary colours (blue, green, purple, teal, magenta) only for feedback, statuses, planning levels and team dots, always next to a word. Planning levels avoid the green of "approved" and the amber of "waiting". | Brand §3.4, §10.4 | PO-5 |
| Input, select and checkbox borders `#8C8C8C` (3.4:1) — today's `#D2D2D7` is 1.5:1, a WCAG 1.4.11 failure. | Patterns §7 | PO-4 |
| Focus ring in Shell's system navy `#092949` in light mode (14.7:1 on white; 9.5:1 next to a yellow button) and Shell yellow in dark mode; links in Shell dark blue `#003C88` (10.5:1), underlined inside sentences, so links stand out in a working tool. | Brand §10.2; patterns §9.2, §9.9 | PO-4 |
| Environment banner in Shell's navy system-message colour `#092949` with white text instead of pale yellow, which would clash with the brand yellow (and turns olive in dark mode). | Patterns §5 "System message"; UI audit D-02 | – |
| Dark mode built on Shell's own dark palette: charcoal `#14191F`/`#1D232A`, dimmed yellow `#D6A100` with dark text (7.5:1), a yellow focus ring, red `#C42B2F` for danger (white text 5.6:1); every pair re-checked (≥ 4.5:1 for text). | Brand §3.2, §10.3 | PO-4 |
| One meaning per status colour: neutral grey, waiting amber, positive green, running blue, done slate, problem red; "In progress" only for a running activity, the approval roll-up renamed "Partly approved" or "2 of 4 approved". | UI audit X-14; functional review A-02 | PO-5 (rename needs confirmation, §8.2) |

### 5.2 Typography

| Recommendation | Evidence | Decision |
|---|---|---|
| Jost 600/700 for headings and summary values, self-hosted from `@fontsource/jost` 5.3.0 (Latin subset, no request to the internet); Arial for all other text. | Patterns §3.3, image 08 | PO-3 |
| Seven sizes replace fifteen: 12/18, **14/20 (default)**, 16/24, 18/24, 20/28, 28/36 and 32/40 (the Home greeting only). Page title 28 px; card titles 16 px; dialog, panel and form-section titles 18 px. | Shell's scale and six core styles (patterns §3.2); UI audit X-10 | PO-3 |
| Arial 400/700 and Jost 600/700 only; sentence case; no upper-case eyebrows or tags; normal letter spacing; nothing under 12 px (level tags and the sidebar label are 10.5 px today). | Patterns §8; UI audit AX-01, X-18 | PO-4 |
| Tabular figures for all numbers in tables and totals. | Arial digits are equal width (brand §5) | – |

### 5.3 Spacing and sizing

| Recommendation | Evidence |
|---|---|
| Spacing only in 4 px steps (`--space-1` 4 px to `--space-16` 64 px) with fixed uses: 8 label to field, 20 between fields, 24 card padding, 20 between cards, 24 below the page header. | Shell's spacing table (patterns §4, §9.4); UI audit X-10 |
| Three control heights: 32 (in tables and card headers), **40 (default)**, 48 (sign-in, phone forms, tabs); every control in a toolbar the same height. | Patterns §8 (seven heights today) |
| Four radii from Shell's scale: 4 chips and tags; 8 controls, menus and cards (flat, with a 1 px border on the grey page — calmer than Shell's 16 px marketing cards); 12 dialogs; 16 the sign-in card only. No pill-shaped buttons, tabs or search fields. | Patterns §4, §8 |
| Minimum target 24 × 24 px (ⓘ, chip ✕ and checkboxes are 18–20 px today). | WCAG 2.5.8; UI audit PH-05 |
| A lint rule that rejects raw pixel values, colours and other breakpoints, so the scale holds. | UI audit fix 7 |

### 5.4 Layout and grid

| Recommendation | Evidence |
|---|---|
| Content column fluid up to **1300 px** (today 1120 px fixed), centred, 24 px side padding (16 on phones); 12 columns with 20 px gaps. The plan tree, Plans table and All work grid gain room. | Shell `--page-width`; UI audit X-13 |
| Forms in a 720 px column aligned to the left edge, with a right rail at ≥ 1280 px for section links and progress ("9 of 15 needed fields filled") instead of 40 % empty screen. | UI audit X-13, F-01, AF-01; functional review P-04 |
| Three breakpoints (600, 960, plus 1280 for side rails) instead of 11 media queries at 8 widths. | Patterns §9.5 |
| Five page templates: list, record, form, settings section, dashboard. | UI audit X-03, X-12 |
| Equal-height cards in a row; lists in side-by-side cards capped at 5 rows with "View all (n)". | UI audit X-11, X-12, H-02, PR-01 |

### 5.5 Components

| Recommendation | Evidence |
|---|---|
| **One page header** for every page: a breadcrumb row always reserved (so titles sit at the same height everywhere), a 28 px title, a meta row on record pages, actions at the right with the primary last. No eyebrow, no grey "highlight" phrase. | UI audit X-03, X-19, F-04, ST-03, LI-02 |
| **One side-panel footer**: right-aligned Cancel · secondary · primary; destructive actions in the header's ⋯. The same order in dialogs and full-page form footers. | UI audit X-05, X-06 |
| **One table standard**: sentence-case bold headers on grey, 40 px rows, numbers right-aligned under right-aligned headers, names never broken inside a word, a scroll cue when wider than the card, no labels repeated in cells. | UI audit X-01, X-07, X-18, AW-05, P-02 |
| **One key–value layout** with 200 px labels. | UI audit X-08 |
| **Underline tabs** with the yellow bar instead of pill tabs; segmented filters only for filters, with counts "Active (5)". | Patterns §5 "Tabs"; UI audit AP-04, X-16 |
| **One select style** and **one date input** that shows the person's date format (today the browser shows "12/31/2026" next to "31 Dec 2026"). | UI audit X-15, C-04; functional review P-03 |
| Summary cards without tinted icon tiles, values aligned across a row even when a label wraps. | UI audit H-03, AW-02, D-01 |
| Status chips and level tags at 12 px, sentence case; at most two chips per tree row. | UI audit AX-01, PL-02, X-14 |
| Toasts that never cover a sticky Save bar. | Functional review P-09 |
| An inline alert under the header for waiting and declined records (the decline reason is today 2,190 px down the page). | UI audit AC-02 |

### 5.6 Navigation and user flows

| Recommendation | Evidence |
|---|---|
| **App shell**: a 56 px white top bar (logo, reserved search slot, notifications, account menu with appearance and sign-out) and a 240 px white sidebar with 40 px items and the **4 px yellow "you are here" bar**, collapsing to a 64 px rail and to a drawer on phones. | Patterns §9.7; Fleet Hub's left navigation (patterns §6.2) |
| **One primary per page, chosen by the person's job**: Edit for the owner; Approve and Decline side by side for the approver; the owner of a submitted activity is not offered Approve as a button. | UI audit X-04, AC-03; functional review A-01 |
| **One rule for side panel vs page**: short records without child tables in a panel; plan, activity, line item and Shell attribute on a page. Written down so users can predict it. | Functional review S-01 |
| **Record pages with tabs** when they have three or more areas (plan: Plan tree · Details · History; activity: Overview · Details · Approval · History), shortening 3,000–4,000 px pages. | UI audit AC-02, AF-01 |
| **No dead ends in a client build**: merge the Activity log and Recycle bin (feature 011) before the client demo, or hide the menu items and stop promising a restore until then. | UI audit A-01, H-05; functional review P-02, X-03 |
| A global search in the top bar that opens All work with the query is a cheap way to meet the "find an activity in ≤ 2 steps" target. Optional; needs the product owner's agreement. | Design system §10; patterns §6.1 item 7 |

### 5.7 Accessibility

| Recommendation | Evidence |
|---|---|
| Move from WCAG 2.1 AA to **WCAG 2.2 AA**: focus not obscured by sticky bars and toasts, 24 px targets, no dragging without an alternative, help in a consistent place, no redundant entry, accessible sign-in. | Shell statement (brand §9); PO-4 |
| Keep NorthStar's stronger-than-Shell basics: 2 px focus ring with offset (keyboard only, not after a mouse click), reduced motion, error summary with links, status as words. | Patterns §7; UI audit AX-03 |
| Fix the one real contrast failure (input borders) and the "Calculated" chip contrast. | Patterns §7; UI audit F-05 |
| Run axe with the `wcag22aa` tag on every page in both themes, plus a keyboard walk and a screen-reader smoke test of the CRUD flow. | Implementation plan §4 |

### 5.8 Responsiveness

| Recommendation | Evidence |
|---|---|
| No page may scroll sideways at any width: fix the scroll boxes (`position: relative`), full-width side panels below 600 px. | UI audit X-02, PH-01, PH-02; functional review X-01 |
| Phones: summary cards 2 × 2; inbox lists (Approvals, Home) as stacked cards; filters in a drawer on All work; breadcrumb collapsed to "‹ Parent"; banner one 32 px line; tabs and segmented filters scroll instead of wrapping; inputs 16 px text (no zoom) and 48 px high. | UI audit H-06, AW-03, PH-03, PH-04, PH-06, A-03; patterns §7 |

### 5.9 Words, numbers and dates

| Recommendation | Evidence |
|---|---|
| One date format everywhere, including date inputs; times 24-hour. | UI audit X-15; functional review P-03 |
| One money format per screen (symbol form, or the code form for every amount when a screen shows several currencies); "38%" without a space; "—" for nothing. | UI audit AC-04, X-17; functional review P-07, X-08 |
| Page titles equal to menu names; one wording per problem ("1 line item needs attention"); no repeated headings ("Shell attributes" › "Shell attributes"). | UI audit AW-01, PR-02, X-09; functional review X-07, P-08 |

---

## 6. What users will notice

| Today | After the revamp |
|---|---|
| White page, grey sidebar, blue buttons, purple gradient logo | Light-grey canvas with white cards, white sidebar with a yellow bar on the current section, one yellow button per page, a navy NorthStar mark |
| Five header layouts; titles jump between pages | One header; titles at the same place on every page |
| Plan tree rows 200 px tall with one letter per line | Compact 40 px rows, names on at most two lines |
| Phone pages wider than the screen, panels off-screen | No sideways scrolling; panels fill the phone screen |
| Pill tabs, three select styles, browser date fields in US format | Underline tabs, one select style, dates in the person's format everywhere |
| Numbers left- or right-aligned depending on the table | Numbers always right-aligned under their headers |
| "In progress" in two colours with two meanings | One word, one colour, one meaning |

---

## 7. What waits for official Shell assets

| Item | Why it waits | Stand-in until then | Who to ask |
|---|---|---|---|
| Shell typeface (Shell Heavy, Bold, Book; URW) and a licence for an internal app | Proprietary font, "all rights reserved"; must not be copied from shell.com | Jost 600/700 (OFL) for headings, Arial for text | Shell brand team via Brand Central |
| Pecten files and consent to show them in NorthStar | Registered trade mark; yearly consent and artwork approval | NorthStar's own ink star mark; no "Shell" wordmark, no red-and-yellow shapes | Shell program manager → Brand Central consent; artwork to `Askbrand@shell.com` |
| Official screen colour values (yellow `#FFC800` vs `#FBCE07`; red `#DD1D21`) and the secondary palette | Only the live website CSS is public | The live shell.com values | Shell brand team |
| Shell's internal design system ("Shell DS", seen only as a sign-in page) | It may be mandatory for internal tools and would outrank this work | Design system version 2 | Shell digital / IT contact for internal applications |
| Icon library, illustration, photography | Shell images are copyright | Open outline icons (lucide); no photography | Shell brand team |
| Internal tool naming and co-branding rules | Not public | "NorthStar" | Shell program manager |
| Internal accessibility checklist for employee applications | The public statement says WCAG 2.2 AA | WCAG 2.2 AA | Shell digital accessibility contact |

Every stand-in is one token or one component, so each asset can be switched in without redesigning screens.

---

## 8. Questions

### 8.1 For Shell

1. **Font licence.** May NorthStar, as an internal Shell tool, use the Shell typeface (Shell Book, Bold and
   Heavy, by URW), and will Shell supply the files and the licence terms? Body text too, or headings only?
2. **Pecten.** Will Shell grant trade-mark consent and artwork approval for the pecten in NorthStar's top
   bar and favicon, and supply the clear-space and minimum-size rules?
3. **Shell DS.** Does Shell have an internal design system for staff and B2B applications ("Shell DS":
   tokens, React components, icons)? Is it mandatory for NorthStar? If so, can the team get access?
4. **Colour values.** Which screen values are official: yellow `#FFC800` (today's websites) or `#FBCE07`
   (Fleet Hub), red `#DD1D21` or `#DD1D20`, and the secondary palette?
5. **Primary colour for staff tools.** The product owner chose yellow with dark text, as on shell.com.
   Does Shell's digital brand team prefer another colour (Fleet Hub uses dark blue) for internal tools?
6. **Accessibility.** Is there an internal accessibility standard or checklist beyond WCAG 2.2 AA, and which
   assistive technologies, browsers, screen sizes and Windows display scaling do Shell's marketing and
   finance staff use?
7. **Naming.** Is "NorthStar" acceptable as the tool's name, and is any vendor credit allowed or required?
8. **Icons and tone.** Is there a Shell icon set or tone-of-voice guide for internal tools?

### 8.2 For the product owner (decisions the design depends on)

| # | Question | Recommendation | Source |
|---|---|---|---|
| 1 | Rename the approval roll-up "In progress" to "Partly approved" / "2 of 4 approved"? | Yes — "In progress" then means only a running activity | Functional review A-02; UI audit X-14 |
| 2 | Hide Approve as a header button from the owner of a submitted activity (keep it in ⋯ with the warning) until the client rules on self-approval? | Yes | Functional review A-01 |
| 3 | One money format per screen (code form when a screen shows several currencies), refining 007-P4? | Yes | Functional review P-07; UI audit AC-04 |
| 4 | Settings "Set inactive": confirm when the record is in use (listing where), apply with Undo otherwise? | Yes | Functional review S-01 |
| 5 | Record pages with tabs (plan: Plan tree · Details · History; activity: Overview · Details · Approval · History)? | Yes | UI audit AC-02, AF-01 |
| 6 | Before feature 011 is merged: hide Recycle bin and Activity log from the menu and stop promising a restore, or merge 011 first? | Merge 011 first; hide only if the demo comes earlier | UI audit A-01; functional review P-02, X-03 |
| 7 | Global search in the top bar that opens All work with the query? | Optional, low cost | Usability target "find an activity ≤ 2 steps" |
| 8 | Extend the end-to-end checks (D-040 chose smoke tests only) with a screenshot and accessibility suite for the revamp? | Yes — it is the acceptance gate | Implementation plan §4 |
| 9 | Behaviour items raised by the functional review that are not design: remember the plan tree's expand state (P-05), confirm before publishing an empty plan (P-06), copy Shell attributes down from the parent (P-04) | Decide separately; the design supports each | Functional review |

---

## 9. Next steps

1. **Record the decisions** of §3 in the decision log and amend constitution principle III (one yellow
   primary action, WCAG 2.2 AA). Done: D-310–D-314 and constitution 1.1.0.
2. **Build the foundation** (phase 1 of the [implementation plan](ui-ux-implementation-plan.md)): tokens,
   fonts, shared components, app shell and the global bug fixes.
3. **Apply it page by page** (phase 2), each area accepted on screenshots at 1440 and 390 px in light and
   dark mode, axe WCAG 2.2 AA with no violations, and no sideways scrolling.
4. **Send the questions of §8.1 to Shell** through the client team; switch in official assets as they arrive.

---

## Sources

- [Shell brand and design language research](ux-research-shell-brand.md) (20 Oct 2026) — Shell's live tokens,
  dark palette, colour usage counts, pecten consent process, fonts, accessibility statement.
- [Shell UI patterns research](ux-research-shell-ui-patterns.md) (20 Oct 2026) — type scale, grid, components,
  navigation, Fleet Hub, font comparison (evidence images 01–09 in
  [`ux-research-shell-ui-patterns/`](ux-research-shell-ui-patterns/)).
- [NorthStar UI audit](ui-audit.md) (20 Oct 2026) — 97 findings with files, 12 key screenshots in
  [`ui-audit/`](ui-audit/).
- [NorthStar functional review](functional-review.md) (20 Oct 2026) — 25 findings, evidence in
  [`functional-review-evidence/`](functional-review-evidence/).
- Shell's public pages cited in the two research reports, in particular [shell.com](https://www.shell.com/),
  the [accessibility statement](https://www.shell.com/accessibility.html) and the
  [terms of use](https://www.shell.com/terms-of-use.html).
- [Jost](https://fonts.google.com/specimen/Jost), SIL Open Font License 1.1, via `@fontsource/jost` 5.3.0.
