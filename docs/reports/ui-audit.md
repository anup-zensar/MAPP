# MAPP UI audit, page by page

**Date:** 20 October 2026 · **Build audited:** branch `plan-module` (features 001–010; 011 not merged) · **Method:** the real app with the demo data, signed in as four demo people, 310 screenshots, measured layouts, automated accessibility checks.

---

## Summary

The product owner is right: MAPP does not yet look production-ready. The cause is **not** the basic parts: the colour tokens, chips, cards, error summaries, empty states and dark mode are sound, and the automated WCAG check finds almost nothing. Three things make it look unfinished:

1. **Four layout bugs break the most important screens.** The plan tree, the module's main screen, shows each owner's name **one letter per line**. That makes every row about 200 px tall: a plan with 13 records takes about 3,000 px of scrolling. The Plans table does the same to team names. On a phone, 9 pages are wider than the screen (up to 1,099 px on a 390 px phone). The campaign panel then opens off-screen, so its labels show without values and its Close button cannot be reached. Each bug traces to one or two lines of CSS (see X-01, X-02, X-07 and H-01).
2. **Each feature was assembled its own way.** The pages use 5 different header layouts, the side panels 5 different footers, and the details screens 3 key–value layouts with different label widths. Selects come in 3 styles and table headers in 2. Numbers are right-aligned in some tables and left-aligned in others. Next to each other, these differences read as "misaligned" and "no symmetry".
3. **The design tokens have no spacing or type scale.** `tokens.css` defines colours, radii and shadows only. The 155 CSS files therefore use 15 font sizes (10.5–28 px), 15 corner radii and about 40 padding values. Without a scale, drift was bound to happen.

A fourth point is outside this audit but matters for the revamp. The visual language is the version 1 Apple-inspired look: Apple blue `#0071e3`, a blue-to-purple gradient and the `-apple-system`/Segoe UI font stack. Nothing on screen says "Shell". The brand research should define the replacement tokens.

**Count:** the tables below hold 97 entries: **9 high**, **52 medium**, **36 low**. Many page entries repeat a cross-page problem on one specific page, and the 9 high entries come down to **3 root causes** (X-01, X-02, H-01). The 10 fixes below remove most of the "unpolished" impression. Fixes 1, 2 and 10 are small CSS changes.

### The 10 fixes with the most effect

| # | Fix | Removes | Effort |
|---|---|---|---|
| 1 | Stop names, people and team pills wrapping per letter in tables (`overflow-wrap: anywhere` → `break-word`, `white-space: nowrap` on pills/people in table cells, minimum widths for Owner/Teams columns) | X-01: plan tree 3,000 px tall; Plans table | ½ day |
| 2 | Add `position: relative` to every sideways-scrolling table box | X-02: all 9 phone pages wider than the screen; off-screen side panels and toasts | 1 hour |
| 3 | One page-header component for every page (list, detail, form) with fixed slots | X-03: 5 header layouts, titles at y = 72 / 85 / 113 / 116 px | 2 days |
| 4 | One side-panel footer and one form-footer rule (primary first, right-aligned in panels and dialogs; destructive actions in ⋯) | X-05, X-06 | 1 day |
| 5 | A table standard: numbers right-aligned and tabular, one header style, names never wrap per letter, scroll cue when wider than the card | X-07, X-17, X-18, AW-05 | 2 days |
| 6 | One key–value component ("description list") with one label width | X-08: 24 px value offsets, 3 layouts | 1 day |
| 7 | Spacing and type scales as tokens, plus a lint rule that rejects raw px values | X-10: the cause of most drift | 2 days + clean-up |
| 8 | Fluid content width for data pages (tree, All work, tables); a right-hand summary rail for 720 px forms | X-13: squeezed tables, empty right halves | 1–2 days |
| 9 | One colour per status family, never the same word in two colours | X-14: two "In progress" colours on one calendar | 1 day |
| 10 | Fix the Home "My activities" amount position and the card-header alignment | H-01, X-11 | 2 hours |

---

## How the audit was done

- **App:** a separate copy of the demo data (`go run ./cmd/seed -reset`) and the API on `127.0.0.1:18101`. The web app ran on `127.0.0.1:14101`. The servers on 8080 and 5173 were not touched. Both audit servers are stopped.
- **People:** Mei Lin (marketer), Priya Raman (marketer with the richest data), Daniel Okoro (team leader and approver) and Alex Morgan (administrator).
- **Screens:** 61 pages and states: sign-in (empty and with errors), Home (two people), Plans (cards, table, empty search, actions menu, copy and delete dialogs), the plan page (collapsed, tree expanded, draft AOP plan), new and edit plan (with errors), campaign and initiative panels (details, edit, new with errors), the activity page (draft, approved, declined, approver view), activity forms, the delete dialog, line item page and forms, Approvals (waiting, empty, approve and decline), All work (grid, calendar, budget, expanded, empty search), Settings and every section with details panels, People, Teams, My profile, Coming soon, Not found, account menu, notifications and the collapsed sidebar.
- **Sizes and themes:** each page at **1440 × 900** and **390 × 844** (phone), **light and dark**, as full-page screenshots (242), plus 68 viewport screenshots of what a person actually sees: side panels, sticky footers, menus.
- **Measurements:** a script recorded where each page's title sits, how numbers in tables are aligned, which elements stick out past the screen, tap-target sizes and text under 12 px. **axe-core** ran on every desktop page in both themes (WCAG 2.1 A/AA).
- **Evidence:** the 12 most telling screenshots are in [`docs/reports/ui-audit/`](ui-audit/). The other screenshots are named `<nn>-<page>__<d|m>-<light|dark>.png` (`d` = desktop, `m` = phone; `v…` = viewport shot). They are kept outside the repository, in the session's scratch folder `…\scratchpad\uiaudit\shots` and `…\vshots`, and are not committed. Positions are in CSS pixels at 1440 px wide unless a phone is named.

**Severity:** **High** = a key screen looks broken or can't be used. **Medium** = a visible inconsistency or misalignment that erodes trust. **Low** = polish.

---

## The 12 key screenshots

| File | What it shows |
|---|---|
| [01-plan-tree-owner-column-collapses.png](ui-audit/01-plan-tree-owner-column-collapses.png) | Plan tree: "P r i y a R a m a n" one letter per line; every row about 200 px tall |
| [02-plans-table-text-wraps-per-letter.png](ui-audit/02-plans-table-text-wraps-per-letter.png) | Plans table: team pills per letter, "38 / %" split across lines, plan names on 3 lines |
| [03-five-different-page-headers.png](ui-audit/03-five-different-page-headers.png) | Plans, plan, activity, line item and edit plan headers side by side: five layouts |
| [04-five-different-panel-footers.png](ui-audit/04-five-different-panel-footers.png) | Five side panels, five footers: left- or right-aligned, Delete as text, as an outlined button or in ⋯ |
| [05-home-uneven-columns-and-amounts.png](ui-audit/05-home-uneven-columns-and-amounts.png) | Home: unequal columns, card dividers 21 px apart, an amount floating mid-card |
| [06-line-items-numbers-not-under-headers.png](ui-audit/06-line-items-numbers-not-under-headers.png) | Activity line items: right-aligned headers above left-aligned numbers (23 px and 43 px off) |
| [07-plan-details-value-column-offset.png](ui-audit/07-plan-details-value-column-offset.png) | Plan details: the Shell attribute values start 24 px further right than every other value |
| [08-calendar-legend-colour-clashes.png](ui-audit/08-calendar-legend-colour-clashes.png) | All work calendar: "In progress" in amber **and** blue; "Submitted" in the same amber |
| [09-budget-view-numbers-clipped.png](ui-audit/09-budget-view-numbers-clipped.png) | All work budget view: values cut mid-number ("$24,000.") at the card edge, with no scroll cue |
| [10-phone-panel-values-off-screen.png](ui-audit/10-phone-panel-values-off-screen.png) | Phone: the campaign panel shows labels with no values; Close and Edit are off-screen |
| [11-phone-page-wider-than-screen.png](ui-audit/11-phone-page-wider-than-screen.png) | Phone: the Models page renders 654 px wide on a 390 px screen |
| [12-activity-form-long-and-narrow.png](ui-audit/12-activity-form-long-and-narrow.png) | New activity: 4,092 px long in one 720 px column, with the right ~40 % of the screen empty |

---

## Part 1: Problems that repeat across pages

These come first because fixing each once fixes many pages.

| ID | Problem | Evidence | Likely cause (file) | Severity |
|---|---|---|---|---|
| **X-01** | **Names and pills break one letter per line inside tables.** The plan tree's Owner column shrinks to about 10 px, so "Priya Raman" becomes 11 lines and each child row is about 200 px tall. The plan page is 4,063 px long, 3,000 px of it the tree. The Plans table does the same to team pills ("Glo / bal / Mar / keti / ng / Ops"). | `01-…`, `02-…`; `11-plan-tree-expanded__d-light.png`, `12-plan-draft-aop__d-light.png`, `05-plans-table__d-light.png` | `overflow-wrap: anywhere` in `components/PersonLabel/PersonLabel.module.css:17,32`, `components/TeamPill/TeamPill.module.css:13` and `components/TreeTable/TreeTable.module.css:124`. In an auto-width table this value lets the browser shrink a column to one character. The rule appears **63 times in 46 CSS files**. | **High** |
| **X-02** | **On a phone, 9 pages are wider than the screen**: plan page 1,099 px, Plans in table layout 1,059, a draft plan 996, activity page 839, Shell attributes 756, Models 654, Currencies 646, Teams 615, People 430. The browser widens the whole page, so fixed elements are placed for 1,099 px. The campaign panel's Close button lands at x = 1,047 and its Save button at x = 758, both outside the 390 px screen. Toasts centre at x ≈ 550. | `11-…`, `10-…`; `39-settings-models__m-light.png`, `10-plan-page__m-light.png`, `v02-campaign-panel__m-light.png` | The tables do scroll inside their cards. However, the hidden "Actions" column label (`.visually-hidden`, `position: absolute`, `styles/global.css:62`) escapes the scroll box because the box is not `position: relative`. It then sits at x = 1,099 and stretches the page. Fix: add `position: relative` to `.scroll` in `TreeTable.module.css:2`, `PlanTable.module.css:2`, `LineItemsSection.module.css`, `settings/shared/SettingsTable.module.css` (`.tableScroll`) and `people/lists.module.css`. | **High** |
| **X-03** | **Five different page-header layouts** (screenshot 03). The title's top edge sits at y = 85 on list pages, 72 on the plan page, 113 on the activity page and 116 on line items, forms and Settings sub-pages. The level tag sits left of the title (plan), above it (activity), or is repeated as both eyebrow and tag (line item: "LINE ITEM" twice). The title is 26 px everywhere except the activity page (22 px). The overflow menu is "⋯" on plan and activity but a "More ▾" button on line items. Edit has a pencil icon on activity and line item but not on plan. Edit pages stack breadcrumb + eyebrow + title + subtitle, four lines before any content. | `03-…`; metrics of `10-plan-page`, `21-activity-page-draft`, `26-line-item-page`, `15-plan-edit` | `components/PageHeader/PageHeader.tsx`, `features/plans/PlanHeader.tsx`, `features/activities/ActivityHeader.tsx` (+ `.module.css`), `features/line-items/LineItemPage.tsx`. Each feature built its own header. | **Medium** (the most visible inconsistency) |
| **X-04** | **Two blue primary buttons in one header.** The activity page shows **Edit** and **Submit for approval** both as filled blue buttons. An approver sees **Edit** and **Approve**, while Decline is hidden in ⋯. This breaks the design system's "at most one blue primary button" (§2) and "Edit primary · next status action secondary" (§6.3). | `21-activity-page-draft__d-light.png`, `32-approval-decision-dialog__d-light.png` | `features/activities/ActivityHeader.tsx`, `useActivityActions.tsx` | **Medium** |
| **X-05** | **Five different side-panel footers** (screenshot 04). Campaign: Edit + ⋯, right. Approval workflow: Edit + Set inactive **left**, Delete as red text at the far right. Person: Edit + Deactivate, right, with a note below. Team: Edit + Set team inactive + an **outlined** red Delete, right. New campaign: Save + Save and add another + Cancel, right. | `04-…`; `48-…`, `50-…`, `52-…` (`__d-light`) | `features/campaigns/PanelFrame.tsx`, `features/approval-workflows/WorkflowDetailsPanel.module.css`, `features/people/panels.module.css`, `components/SidePanel/SidePanel.tsx` | **Medium** |
| **X-06** | **Form buttons move around.** Full-page forms (plan, activity, line item) put Save/Cancel at the **left** of a 720 px column. Panel forms and dialogs put them at the **right**. | `v05-plan-new-form__d-light.png` vs `v04-campaign-new-errors__d-light.png` | `components/FormFooter/FormFooter.module.css` vs `SidePanel` | **Medium** |
| **X-07** | **Numbers are aligned differently from table to table.** All work and Home's "Plans at a glance" right-align them; Plans (year, %), the plan Goals table, Settings › Models ($10,000.00, counts) and Activity types (default cost) left-align them. The activity's line items table right-aligns its headers but left-aligns the values under them: 23 px off under "Planned cost" and 43 px under "Planned cost (USD)". The measurement script found left-aligned numbers in tables on 7 distinct pages (21 of the 61 states). | `06-…`; `21-activity-page-draft__d-light.png`, `39-settings-models__d-light.png`, `05-plans-table__d-light.png` | Line items: `features/line-items/LineItemsSection.module.css:24` (`.table td {text-align:left}`) outranks `.number` at `:37`, a CSS specificity bug. Settings: `settings/shared/SettingsTable.module.css` has no number class on money columns. Plans: `features/plans/PlanTable.module.css:23`. | **Medium** |
| **X-08** | **Three key–value layouts with different label widths**, sometimes in the same card. In plan details the label column is 200 px but the Shell attributes block uses 220 px, so its values start **24 px further right** (screenshot 07). Activity details use a 4-column "label above value" grid right above a "label left, value right" list. The line item's Finance coding card mixes both. Side panels use a 38 % label column; person and team panels stack labels above values. | `07-…`; `12-plan-draft-aop__d-light.png`, `21-activity-page-draft__d-light.png`, `26-line-item-page__d-light.png` | `features/plans/PlanPage.module.css:152` (200 px) vs `components/ShellAttributeFields/ShellAttributeFields.module.css:63` (`minmax(120px, 220px)`); `features/activities/ActivitySections.module.css:3`; `features/campaigns/PanelSection.module.css:25` | **Medium** |
| **X-09** | **Duplicate headings.** "Shell attributes" › "Shell attributes" appears on the plan form, plan edit, activity form and activity page; "Finance coding" › "Finance coding" on the line item page and form. | `14-plan-new-errors`, `21-activity-page-draft`, `26-line-item-page`, `27-line-item-edit` (`__d-light`) | `components/ShellAttributeFields/ShellAttributeFields.tsx` and `ShellAttributeValues.tsx` print the attribute section's name inside a section that already carries that name. | **Medium** |
| **X-10** | **No spacing or type scale.** `styles/tokens.css` defines colours, radii, shadows and one font stack, but no spacing or text sizes. Across 155 CSS files there are **15 font sizes** (10.5, 11, 11.5, 12, 12.5, 13, 13.5, 14, 15, 16, 17, 22, 26, 27, 28 px), **15 different corner-radius values** (the four radius tokens plus raw 3, 4, 6, 8, 9, 12 and 20 px, 50 % and others) and more than 40 padding combinations. Every inconsistency above grows from this. | grep of `frontend/src/**/*.css` | `frontend/src/styles/tokens.css` | **Medium** (root cause) |
| **X-11** | **Card titles don't line up with their buttons, and card dividers don't line up across a row.** Card headers align to the top, so a title sits 4–9 px above the centre of its header button ("Line items" vs "Add line item": 9 px). When one card in a row has a subtitle or button and its neighbour doesn't, their header dividers miss each other: Home 21 px (y = 692 vs 671), Profile 11 px (y = 220 vs 209). | `06-…`, `05-…`; `55-profile__d-light.png` | `components/Card/Card.module.css:11` (`align-items: flex-start`); no fixed header height | **Medium** |
| **X-12** | **Ragged two- and three-column layouts with large empty areas.** On Home the left column ends at y = 1,190 and the right at 855, leaving 335 px of blank space. On the plan page the Goals card ends 274 px above the Details card. The three Profile cards are 449, 425 and 302 px tall. | `05-…`; `11-plan-tree-expanded__d-light.png`, `55-profile__d-light.png` | `features/home/HomePage.module.css`, `features/plans/PlanPage.module.css:120`, `features/people/ProfilePage.module.css` | **Medium** |
| **X-13** | **The content width is fixed at 1,120 px.** Data-heavy pages get squeezed even on a 1,440 px screen: the plan tree's 8 columns, the 9-column Plans table, the All work grid. Collapsing the sidebar frees 172 px but the content stays 1,120 px. Forms and the line item page sit in a 720 px column, leaving about 400 px empty at the right. | `60-sidebar-collapsed__d-light.png`, `26-line-item-page__d-light.png`, `12-…` | `--content-max` in `styles/tokens.css`; `app/AppLayout.module.css` `.content`; `features/line-items/LineItemPage.module.css:31` and the form modules (`max-width: 720px`) | **Medium** |
| **X-14** | **Status colours clash.** On the calendar legend "In progress" appears twice, in amber (campaign/initiative) and in blue (activity). "Submitted" uses the same amber, "Complete" and "Created" are both grey, and "All approved" and "Approved" are both green. A plan-tree row can stack up to 4 chips. | `08-…`; `34-all-work-calendar__d-light.png`, `11-plan-tree-expanded__d-light.png` | `components/StatusChip/StatusChip.tsx`, `features/campaigns/StatusChips.tsx`, `features/activities/activityStatus.ts`, `features/all-work/CalendarView.tsx` | **Medium** |
| **X-15** | **Three select styles and two date formats.** Forms and the Plans year filter use the native Windows dropdown; Home's currency picker is a custom select with a triangle; people and teams use a custom combobox. Date inputs show the browser's `01/01/2026` (US order on this machine) while every other date reads `01 Jan 2026`. | `13-plan-new-form`, `03-home`, `17-campaign-edit-panel` (`__d-light`) | `components/FieldShell/inputs.module.css`, `features/home/CurrencyPicker.module.css`, `components/Combobox`, `components/DateRangeField` (known item 003 FR-024) | **Medium** |
| **X-16** | **Toolbars differ on every list page.** The result count sits below the toolbar on Plans ("3 of 3") and inside it elsewhere ("2 of 2", "6 people", "Showing 10 of 10 activities"). Filters default to *Active* on Models, Currencies and Shell attributes but to *All* on People and Approval workflows. Filter counts are plain text ("Active 2") in Settings but badges on Approvals. On Plans the toolbar stops at x = 1,290 while the content edge is 1,382 (92 px short). Tabs shrink to fit on Plans but fill a full-width grey bar on Shell attributes and Finance budget items. | `04-plans-cards`, `29-approvals-waiting`, `39-settings-models`, `41-settings-shell-attributes`, `49-settings-people` (`__d-light`) | `features/plans/PlansPage.module.css`, `features/settings/shared/*`, `features/people/lists.module.css`, `components/PillTabs` | **Low** |
| **X-17** | **Percentages break across lines.** "38 %" is written with a space, so narrow cells show "38" on one line and "%" on the next (Plans table, plan tree, plan cards: "38 % / used"). UK style is "38%". | `02-…`, `01-…`, `04-plans-cards__d-light.png` | `lib/` formatting helpers (percent format); `PlanCard.module.css` | **Low** |
| **X-18** | **Two table-header styles.** Home uses UPPERCASE 12 px on white; Plans, Settings and line items use Title case on a grey band; the plan tree uses Title case without a band. | `03-home`, `05-plans-table`, `11-plan-tree-expanded` (`__d-light`) | `features/home/PlansAtAGlanceCard.module.css:41`, `PlanTable.module.css`, `TreeTable.module.css` | **Low** |
| **X-19** | **Breadcrumbs press against the demo banner.** Detail pages start their breadcrumb about 10 px under the banner (y = 41), while list pages start their eyebrow at y = 72. | `03-…` | `components/Breadcrumb/Breadcrumb.module.css`, page modules | **Low** |

---

## Part 2: Page by page

### Sign-in (`01-sign-in`, `02-sign-in-errors`)

Clean and readable in both themes; the error summary and field messages are exemplary.

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| S-01 | The brand appears twice: the logo top-left of the page **and** in the card's gradient header. The card floats on a grey page with nothing else on it. | `01-sign-in__d-light.png` | `features/sign-in/SignInPage.module.css`, `app/PublicLayout.module.css` | Low |
| S-02 | The blue-to-purple gradient header is the only strongly branded element in the product, and it isn't Shell's. | `01-sign-in__d-light.png` | `--brand-gradient` in `styles/tokens.css` | Low (brand) |

### App shell: sidebar, banner, top bar, footer

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| A-01 | Two of seven sidebar items (Recycle bin, Activity log) lead to "Coming soon" pages. The delete dialog still promises "It moves to the recycle bin; you can restore it". | `56-coming-soon__d-light.png`, `09-plan-delete-dialog__d-light.png` | `app/navigation.ts` | Medium (trust) |
| A-02 | With the sidebar collapsed, content stays 1,120 px wide; the freed space is unused (see X-13). | `60-sidebar-collapsed__d-light.png` | `app/AppLayout.module.css` | Low |
| A-03 | On phones, the banner (2 lines, 44 px) plus the top bar (56 px) take 100 px before any content. | `03-home__m-light.png` | `components/DemoBanner`, `app/MobileTopBar.module.css` | Low |

### Home (`03-home`, `03b-home-admin`, `59-notifications`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| H-01 | **Amounts in "My activities" jump around.** When a row has completeness chips, the amount drops under the text and floats mid-card ($40,000.00 at x ≈ 562–625). Rows without chips keep it at the right edge ($150,000.00 ends at x = 811). The same happens on phones and in Priya's view ($120,000.00). | `05-…`; `59-notifications__d-light.png` | `features/home/MyActivitiesCard.module.css:73` (`.row` wraps); `.side` needs `margin-left: auto`, or use a 2-column grid | **High** (looks broken) |
| H-02 | Uneven columns: "Needs your attention" runs 335 px longer than "Waiting for my approval", and the header dividers are 21 px apart (X-11, X-12). | `05-…` | `features/home/HomePage.module.css` | Medium |
| H-03 | Summary cards: the 4th label wraps ("Planned cost vs plan / budget"), so its value sits 4 px lower than the others. Its ⓘ is pinned right while cards 1–3 have it after the label. These cards (icon tile + subtitle) also look different from the Plans, Approvals and All work summary cards. | `03-home__d-light.png` | `features/home/SummaryCards.module.css` vs `components/SummaryCard` | Medium |
| H-04 | The Currency picker is centred on the whole header block (y = 123), aligned with neither the title (y = 102) nor the subtitle (y = 132). | `03-home__d-light.png` | `features/home/HomePage.module.css`, `CurrencyPicker.module.css` | Low |
| H-05 | "Recent changes: Coming soon" takes a full card in a prime spot (waits for 011). | `03-home__d-light.png` | `features/home/RecentChangesCard.tsx` | Low |
| H-06 | Phone: the four summary cards stack full-width, about 600 px for four numbers. Use a 2 × 2 grid. | `03-home__m-light.png` | `features/home/SummaryCards.module.css` | Low |

### Plans list (`04-plans-cards`, `05-plans-table`, `06-plans-empty-search`, `07`–`09` menu and dialogs)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| P-01 | **Table layout breaks:** team pills wrap per letter, "38 / %" splits, and plan names run to 3 lines while the Completeness column has spare room (X-01, X-17). | `02-…` | `features/plans/PlanTable.module.css`, `TeamPill.module.css` | **High** |
| P-02 | Every table cell repeats its column's label: "Owner: Alex Morgan", "Business unit: Mobility, Lubricants". | `02-…` | `features/plans/PlanTable.tsx` | Medium |
| P-03 | Cards: chips wrap differently per card, so the rows inside the cards don't line up. The owner row sits at y = 483 in card 1 and y = 511 in cards 2–3 (28 px), progress bars 573 vs 601, footers 626 vs 673. | `04-plans-cards__d-light.png` | `features/plans/PlanCard.module.css` | Medium |
| P-04 | The toolbar mixes three control styles (pill segments, a native select, a pill search) and stops 92 px short of the right edge (X-16). | `04-plans-cards__d-light.png` | `features/plans/PlansPage.module.css` | Low |
| P-05 | The Copy dialog (520 px) and Delete dialog (440 px) differ in width; neither has a close ✕. Delete promises a restore that doesn't exist yet (A-01). | `08-plan-copy-dialog`, `09-plan-delete-dialog` (`__d-light`) | `components/FormDialog`, `components/ConfirmDialog` | Low |
| P-06 | Phone: 5 summary cards in 2 columns leave an orphan card, and 4 toolbar rows come before the first plan. | `05-plans-table__m-light.png` | `features/plans/PlansPage.module.css` | Low |

The empty search state is good: a clear message, **Clear filters**, and the removable chip (`06-plans-empty-search__d-light.png`).

### Plan page and plan tree (`10-plan-page`, `11-plan-tree-expanded`, `12-plan-draft-aop`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| PL-01 | **The Owner column collapses** (X-01). 13 rows take about 3,000 px; the tree is hard to read and hard to use. | `01-…` | `PersonLabel.module.css`, `TreeTable.module.css` | **High** |
| PL-02 | Up to four chips stacked in the Status column ("In progress · 2 of 4 activities approved", "Running", "All filled in", "2 items below need attention"): the eye can't scan it. | `01-…` | `features/plans/tree/PlanTree.tsx`, `features/campaigns/StatusChips.tsx` | Medium |
| PL-03 | The same idea is worded and laid out two ways: "Remaining $50,000.00" on the plan row, "$150,000.00 left" on campaign rows (and "left" wraps onto its own line). | `01-…` | `features/plans/tree/PlanTree.tsx` | Low |
| PL-04 | The plan row shows its owner as plain text; every other row shows an avatar + name. | `01-…` | `features/plans/tree/PlanTree.tsx:104` | Low |
| PL-05 | The Total row wraps "Total · 2 campaigns · 4 initiatives · 7 activities" and "Allocated to campaigns" onto 2–3 lines. | `11-plan-tree-expanded__d-light.png` | `PlanTree.module.css` | Low |
| PL-06 | Details card: the Shell attribute values are 24 px out of line (X-08). Goals table: numbers left-aligned, and the "Calculated" chip wraps under the value for Revenue only. | `07-…`; `11-plan-tree-expanded__d-light.png` | `PlanPage.module.css:152`, `ShellAttributeFields.module.css:63`, `features/plans/PlanGoals.tsx` | Medium |
| PL-07 | Goals ends 274 px above Details, leaving a blank block (X-12). | `11-plan-tree-expanded__d-light.png` | `PlanPage.module.css:120` | Low |

### Campaign and initiative panels (`16`–`19`, `v02`–`v04`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| C-01 | **Phone: unusable.** The panel opens beyond the screen edge (X-02): labels show without values, and Close (x = 1,047) and Save (x = 758) are off-screen on a 390 px phone. | `10-…` | root cause X-02; `components/SidePanel/SidePanel.module.css:18` | **High** |
| C-02 | A second breadcrumb inside the panel, right under the page's own breadcrumb. | `v02-campaign-panel__d-light.png` | `features/campaigns/RecordDetails.tsx` | Low |
| C-03 | The "Goals" heading is grey while "Budget and cost" and "Description" are dark: it looks disabled. | `19-initiative-panel__d-light.png`, `16-campaign-panel__d-light.png` | `features/campaigns/PanelSection.module.css` | Low |
| C-04 | Date fields show `01/01/2026` and `12/31/2026` while the rest of the panel shows `01 Jan – 31 Dec 2026` (X-15). | `17-campaign-edit-panel__d-light.png` | `components/DateRangeField` | Medium |

### Plan form, new and edit (`13`–`15`, `v05`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| F-01 | The single 720 px column leaves the right ~40 % of a 1,440 px screen empty, with no summary or section navigation (X-13). | `v05-plan-new-form__d-light.png` | `features/plans/PlanForm.module.css:6` | Medium |
| F-02 | Help text sits **above** some inputs ("Default: the current year") and **below** others (the model's rates). | `14-plan-new-errors__d-light.png` | `components/FieldShell` | Low |
| F-03 | "Shell attributes" › "Shell attributes" (X-09). | `14-plan-new-errors__d-light.png` | `ShellAttributeFields.tsx` | Medium |
| F-04 | Edit plan stacks breadcrumb + "PLANS" eyebrow + "Edit plan" + the plan name, four lines; the title sits 31 px lower than on New plan (X-03). | `15-plan-edit__d-light.png` | `features/plans/PlanEditPage.tsx` | Medium |
| F-05 | The "Calculated" chips in the goals panel fail WCAG colour contrast (the only axe finding in light mode). | `15-plan-edit__d-light.png` | `features/plans/PlanForm.module.css:94–100` | Medium |

### Activity page (`20`–`22`, `25`, `32`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| AC-01 | Header: the tag sits above a 22 px title (other details pages use 26 px with the tag left of the title), and two blue primary buttons (X-03, X-04). | `03-…` | `features/activities/ActivityHeader.tsx` (+ `.module.css`) | Medium |
| AC-02 | **The decline reason is buried.** A declined activity shows only a small "Declined" chip at the top; the reason ("The planned cost is above this quarter's budget…") appears at y ≈ 2,190 of a 2,867 px page. | `22-activity-page-declined__d-light.png` | `features/approvals/ApprovalSection.tsx`, `ActivityPage.tsx` | Medium |
| AC-03 | An approver opening a submitted activity sees a "Ready to submit" card, which is meaningless to them, and Decline is not next to Approve. | `32-approval-decision-dialog__d-light.png` | `features/activities/ReadyToSubmitPanel.tsx`, `useActivityActions.tsx` | Medium |
| AC-04 | Money in three formats on one page: "$120,000.00" (header), "50,000.00 USD" (line items table) and "20,000.00 USD not yet assigned". | `21-activity-page-draft__d-light.png` | `features/line-items/lineItemText.ts`, `features/activities/activityText.ts` | Medium |
| AC-05 | Line items table: numbers not under their headers (X-07); 23 px and 43 px off. | `06-…` | `LineItemsSection.module.css:24` | Medium |
| AC-06 | Three key–value layouts on one page: the facts row, the 4-column Details grid and the label-left Shell attributes list (X-08). | `21-activity-page-draft__d-light.png` | `ActivitySections.module.css` | Medium |
| AC-07 | "From finance (coming later)" uses the browser's ▶ triangle; everything else uses chevron icons. | `21-activity-page-draft__d-light.png` | `features/line-items/FromFinanceSection.tsx` | Low |

### Activity form (`23`, `24`, `v06`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| AF-01 | 4,092 px long in one 720 px column, with no section navigation or progress. The multi-select Shell attributes are long vertical checkbox lists (Country/OU 6, Zero-based cost 4, Brands 4, Sector 4…), mixed with native selects for single choices. | `12-…` | `features/activities/ActivityForm.module.css:6`, `ShellAttributeFields.tsx` | Medium |
| AF-02 | "Shell attributes" › "Shell attributes" (X-09). | `12-…` | `ShellAttributeFields.tsx` | Medium |

### Line item page and forms (`26`–`28`, `v07`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| LI-01 | The page is 720 px wide while its parent (activity) page uses 1,120 px, so moving between them shifts the whole layout. | `26-line-item-page__d-light.png` | `features/line-items/LineItemPage.module.css:31` | Medium |
| LI-02 | "LINE ITEM" appears as both the eyebrow and the tag; "More ▾" replaces the ⋯ used elsewhere (X-03). | `03-…` | `LineItemPage.tsx`, `features/line-items/MoreMenu.tsx` | Medium |
| LI-03 | The Finance coding card has a duplicate heading and switches from label-left to label-above for "Contracted amount" within the same card. | `26-line-item-page__d-light.png` | `LineItemPage.tsx`, `LineItemPage.module.css` | Medium |
| LI-04 | The edit form is clear; the summary strip ("Activity planned cost · Unassigned now · After saving") is a pattern worth reusing. *Note:* a New line item form is fully pre-filled (name, owner, the whole unassigned amount), so one click on Save creates a record. | `27-line-item-edit`, `28-line-item-new-errors` (`__d-light`) | `features/line-items/LineItemForm.tsx` | Low |

### Approvals (`29`–`31`, `v11`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| AP-01 | Owner ("Priya / Raman") and Level ("Level 1 of 1 · Daniel / Okoro") wrap, while the Plan › campaign column also wraps. A 7-column table in 1,080 px. | `29-approvals-waiting__d-light.png` | `features/approvals/ApprovalsTable.module.css` | Low |
| AP-02 | The toolbar has an unexplained gap between the search box (ends x = 642) and Filters (starts x = 695). | `29-approvals-waiting__d-light.png` | `ApprovalsFilters.module.css` | Low |
| AP-03 | A marketer lands on "Waiting for me", which is always 0 for them, instead of the approvals they sent. | `31-approvals-empty__d-light.png` | `ApprovalsPage.tsx` | Low |
| AP-04 | Phone: the three pill tabs wrap inside the rounded pill; "Decided by me" falls to a second row inside the grey shape. | `v11-approvals__m-light.png` | `components/PillTabs/PillTabs.module.css` | Medium |

### All work (`33`–`37`, `v12`)

The best-built page: numbers right-aligned in tabular figures, clear level bars, a total row, good empty state.

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| AW-01 | The title "See all planned work across plans" differs from the menu name "All work"; every other page's title matches its menu name. | `33-all-work-grid__d-light.png` | `features/all-work/AllWorkPage.tsx` | Low |
| AW-02 | The 6th summary card's label wraps ("Unassigned planned / cost"), so its value sits 19 px lower than the other five (y = 240 vs 221). | `33-all-work-grid__d-light.png` | `features/all-work/SummaryCards.module.css` | Medium |
| AW-03 | The order is search, 10 filter chips on 2 rows, a lone result count, then the Grid/Calendar/Budget tabs. The main view switch is buried below the filters. On a phone the chips take 6 rows and the data starts at y ≈ 1,140. | `33-all-work-grid__d-light.png`, `33-all-work-grid__m-light.png` | `AllWorkPage.module.css`, `FilterBar.module.css` | Medium |
| AW-04 | Activity names wrap under their "ACTIVITY" tag, so row heights alternate between 40 and 64 px. | `33-all-work-grid__d-light.png` | `features/all-work/RowName.module.css` | Low |
| AW-05 | **Grid and Budget views are cut at the card edge** mid-value ("$24,000.", a header "U…") with no shadow or scrollbar to show there is more. | `09-…` | `features/all-work/AllWorkGrid.module.css` | Medium |
| AW-06 | Calendar: colour clashes (X-14). Bar labels truncate to "Billboa…" and "Ch…" while there is room in the row. A status chip under each name doubles row height. | `08-…` | `features/all-work/CalendarView.module.css` | Medium |

### Settings hub and sections (`38`–`54`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| ST-01 | Hub: 8 cards in 3 columns leave a hole in the last row. | `38-settings__d-light.png` | `features/settings/SettingsPage.module.css` | Low |
| ST-02 | A marketer sees the same administrator hub with no sign of what they may change (RBAC is deferred; at least show "view only"). | `54-settings-as-marketer__d-light.png` | `SettingsPage.tsx` | Medium (when RBAC lands) |
| ST-03 | Sub-pages repeat the breadcrumb (Settings › Models) **and** a "SETTINGS" eyebrow; the title moves 31 px lower than on top-level pages (X-03). | `39-settings-models__d-light.png` | `components/PageHeader` | Medium |
| ST-04 | Finance budget items: the two stacked tables size their columns separately. Code starts at x = 603 vs 579 (24 px) and Status at 1,156 vs 1,150. | `46-settings-finance-budget-items__d-light.png` | `features/settings/finance-budget-items/*.module.css` (needs fixed column widths) | Medium |
| ST-05 | Money and counts are left-aligned (Models, Activity types) (X-07). | `39-settings-models`, `40-settings-activity-types` (`__d-light`) | `settings/shared/SettingsTable.module.css` | Medium |
| ST-06 | Panels for workflow, person and team each have their own footer and body layout (X-05, X-08). | `04-…` | see X-05 | Medium |
| ST-07 | Phone: Models, Activity types, Shell attributes, Currencies, People and Teams all render wider than the screen (X-02). | `11-…` | X-02 | **High** (part of X-02) |

### People, Teams and My profile (`49`–`52`, `55`)

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| PE-01 | The People toolbar puts the filter on the right with "6 people"; the other Settings lists put filters on the left with "2 of 2" (X-16). | `49-settings-people__d-light.png` | `features/people/lists.module.css` | Low |
| PR-01 | Profile: three cards of different heights (449 / 425 / 302 px) and header dividers 11 px apart (X-11, X-12). | `55-profile__d-light.png` | `features/people/ProfilePage.module.css` | Medium |
| PR-02 | The title "Your profile" differs from the menu and eyebrow "My profile". The "Manage teams in Settings › Teams" link is underlined; links elsewhere are not. | `55-profile__d-light.png` | `ProfilePage.tsx` | Low |

### Menus, notifications, empty and error pages

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| M-01 | Account menu, notifications popover and nav drawer: consistent and tidy; no issues beyond A-03. | `v09-account-menu__d-light.png`, `59-notifications__d-light.png`, `v09-account-menu__m-light.png` | — | — |
| M-02 | Not found and Coming soon: good, centred, with a way back. Coming soon wraps its message in a card; Not found has none. | `56-coming-soon`, `57-not-found` (`__d-light`) | `features/placeholders`, `app/NotFoundRoute.tsx` | Low |

---

## Part 3: Phone (390 × 844)

| ID | Problem | Evidence | Sev. |
|---|---|---|---|
| PH-01 | **9 pages wider than the screen**, so the browser lays out the whole page at up to 1,099 px (X-02). | `11-…`, `10-…`; a script measured `innerWidth = 1,099` on the plan page | **High** |
| PH-02 | **Side panels open off-screen**; Close and Save are unreachable (C-01). | `10-…`; Playwright could not click them | **High** |
| PH-03 | Pill tabs wrap inside their pill (Approvals) (AP-04). | `v11-approvals__m-light.png` | Medium |
| PH-04 | Too much before the content: 4–6 stacked summary cards and, on All work, 6 rows of filter chips (H-06, AW-03). | `03-home__m-light.png`, `33-all-work-grid__m-light.png` | Medium |
| PH-05 | Small tap targets: ⓘ buttons and chip-remove ✕ are 20 × 20 px, checkboxes 18 × 18 px, breadcrumb links 20 px tall. WCAG 2.2 asks for at least 24 × 24 px; Apple and Google recommend 44 and 48 px. | metrics of `03-home`, `13-plan-new-form`, `22-activity-page-declined` (`__m-light`) | Medium |
| PH-06 | Breadcrumbs wrap to two lines on deep pages (activity, line item). | `21-activity-page-draft__m-light.png` | Low |

Tables themselves scroll inside their cards correctly. Once X-02 is fixed, the remaining phone work is mostly density.

---

## Part 4: Dark mode

Dark mode is the most consistent part of the product: every colour comes from tokens, and **axe found 0 violations on all 60 desktop dark pages**.

| ID | Problem | Screenshot | File | Sev. |
|---|---|---|---|---|
| D-01 | The icon tiles on Home's summary cards are hard to see: a purple icon on a dark purple tile, a green icon on a dark green tile. | `03-home__d-dark.png` | `features/home/SummaryCards.module.css` | Low |
| D-02 | The demo banner turns heavy olive (`#3a2f00`) and draws the eye first. | `03-home__d-dark.png` | `--banner-bg` in `styles/tokens.css` | Low |
| D-03 | Every light-mode layout problem above appears identically in dark mode (e.g. the plan tree's owner column). | `11-plan-tree-expanded__d-dark.png` | — | (counted above) |

---

## Part 5: Accessibility

**Good:** a visible 2 px focus ring everywhere; a skip link; every field labelled; error summaries that link to the fields; status always written as words, never colour alone; level bars backed by a text tag; dark mode passes. Automated result: **1 page with a violation** in 121 desktop page-and-theme checks (F-05).

| ID | Problem | Evidence | File | Sev. |
|---|---|---|---|---|
| AX-01 | Text under 12 px in 16 CSS rules: level tags (10.5 px), the sidebar group label (10.5 px), All work cells and calendar (11 px), goal notes (11 px). The measuring script found up to 30 such text runs on one page (expanded plan tree). | `LevelTag.module.css:6`, `Sidebar.module.css:107`, `all-work/*.module.css` | Medium |
| AX-02 | Small tap targets on phones (PH-05). | — | Medium |
| AX-03 | A side panel opens with a heavy focus ring on its Close button. That is correct for keyboard users but distracting with a mouse: show the ring for keyboard focus only (`:focus-visible`). | `v02-campaign-panel__d-light.png` | `components/SidePanel` | Low |

---

## What already works and should be kept

- **The colour tokens and dark mode.** One file, no stray colours (no hard-coded colours outside `tokens.css`).
- **The CRUD language.** "Required / Needed to publish / Needed to submit", error summaries with links, and delete dialogs that say what goes with the record.
- **Empty states** with a clear next action: Plans, Approvals and All work.
- **The All work grid:** right-aligned tabular numbers, totals and level bars. It should become the table standard.
- **The summary strip** on the line item form and the "Calculated" ⓘ pattern.

---

## Recommended order for the UI revamp

1. **Day 1, bug fixes (no redesign needed):** X-01, X-02, X-07 (specificity), H-01, X-08 (one label width), X-17. On their own these make the plan page, the Plans table and phones usable.
2. **Foundations:** add spacing, type and size tokens and a stylelint rule that rejects raw px values. Replace the brand tokens with Shell's (from the brand research).
3. **Shared page parts:** one `PageHeader` (list / detail / form variants, at most one primary action), one `SidePanel` footer, one `FormFooter` rule, one `DescriptionList`, one `DataTable` standard, one `SummaryCard`.
4. **Layout:** fluid width for data pages; forms with a right-hand summary or section rail; equal-height card rows.
5. **Status system:** one colour per family; at most two chips per tree row (status + one problem chip), with the rest in a tooltip.
6. **Phone pass:** a 2 × 2 summary grid, a filter drawer, 24 px minimum tap targets.

---

## Appendix: how to reproduce

The scripts live outside the repository (`…\scratchpad\uiaudit\`):

- `shoot.cjs <baseURL> <outDir> <d-light|m-light|d-dark|m-dark>`: 61 full-page states with layout metrics and axe.
- `shoot2.cjs`: viewport shots of panels, forms and menus.
- `hscroll.cjs`, `chain.cjs` and `diag.cjs`: the phone-width checks behind X-02.

They need the API on a seeded temporary data folder and Vite pointed at it with `MAPP_API_URL`, as described under "How the audit was done". Note: the desktop run pressed Save on an untouched New line item form, which created line item L-1009 in the **temporary** data. That is why later screenshots show "2 line items need attention" on *Launch film*.
