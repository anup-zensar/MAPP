# Decision Log

Product and scope decisions, numbered `D-###`, and open questions, numbered `Q-###`. Technical decisions
that are hard to reverse are ADRs in `docs/architecture/adr/`. Add new entries at the end; never renumber.

## Decisions

| ID | Date | Decision | Why | Source |
|---|---|---|---|---|
| D-001 | 2026-10-18 | The application is called **NorthStar**. This phase covers only the **Plan module** and what it needs (sign-in, shell, settings, people, approvals, views, history). | Product owner instruction | Product owner |
| D-002 | 2026-10-18 | Frontend React + TypeScript; backend Go; REST + JSON with an OpenAPI contract. | Product owner instruction; Shell target architecture | ADR-0002, ADR-0003 |
| D-003 | 2026-10-18 | Data is kept in a local JSON file store behind repository interfaces; PostgreSQL is added when Azure is available. | No Azure yet | ADR-0004 |
| D-004 | 2026-10-18 | Demo sign-in only: one sign-in page in ZenReach style, seeded demo users with a shared password shown on the page. No roles: every signed-in user can do everything. The actor is recorded on every change. | Product owner: authentication and RBAC later | ADR-0005 |
| D-005 | 2026-10-18 | NorthStar uses ZenReach's design language and screen patterns; Hive9 is the functional reference only and its screens are not copied. | Product owner: Hive9 too complex; use ZenReach design | ADR-0006 |
| D-006 | 2026-10-18 | Shell's vocabulary on screen: Plan → Campaign → Initiative → Activity → Line item. ZenReach's Program/Tactic and Hive9 interface terms are replaced (design system §8). | Users know Shell's words | Plan Module doc |
| D-007 | 2026-10-18 | When sources disagree: Shell's Plan Module doc > Hive9 functional analysis (live observations) > GSAP workshop > Hive9 vendor guide > ZenReach spec. ZenReach decides layout and interaction. | Shell's own process is authoritative | docs/reference/README.md |
| D-008 | 2026-10-18 | GSAP, Finance and Measure are out of scope. Line items record "Request SWBS code" and show its status; finance/GSAP values are shown as "coming later". | Phase scope; GSAP frozen by Nucleus | Workshop, project plan |
| D-009 | 2026-10-18 | Two levels of mandatory fields: **Required** (blocks saving, very few) and **Needed to publish / Needed to submit** (blocks publishing a plan or submitting an activity). Shell's "mandatory" fields become "needed" fields. | Lets beginners save drafts; Hive9 guide says mandatory fields gate submission; avoids Hive9's one-error-per-save | Domain model §3 |
| D-010 | 2026-10-18 | Base currency USD; line items may use a local currency converted with a rate "units per 1 USD" kept in Settings; users choose a display currency. | Simple, matches Shell's "preferred currency" display | Plan Module doc step 5 |
| D-011 | 2026-10-18 | Fiscal year = calendar year (Jan–Dec). | Shell plans are named FY2026 with Jan–Dec dates | Assumption, to confirm (Q-004) |
| D-012 | 2026-10-18 | Campaigns and initiatives have no approval workflow of their own; they show derived **Approval progress** and **Timing** statuses. | Hive9 group approvals are tenant-specific and add complexity | Domain model §2.2 |
| D-013 | 2026-10-18 | Changing an approved/submitted activity's planned cost, dates or line-item amounts/currencies sends it back to Created after confirmation; other edits do not. | ZenReach rule; keeps approval meaningful without blocking small edits | Domain model §2.5 |
| D-014 | 2026-10-18 | Shell attributes are configuration (Settings), seeded with Shell's as-is attribute set and fictitious sample options until Shell supplies real lists. | Business configuration is data | Constitution II |
| D-015 | 2026-10-18 | A child budget may exceed its parent's remaining budget; NorthStar shows "Over-allocated by X" in red but does not block. | Shell as-is: remaining budget "takes negative value too" | Plan Module doc step 3 |
| D-016 | 2026-10-18 | A negative unassigned balance (line items exceed the activity's planned cost) warns while editing and blocks submitting. | Keeps approved activities consistent | Domain model §4.3 |
| D-017 | 2026-10-18 | Multi-year activities (linking an activity to initiatives in several fiscal years) are out of this phase; a campaign crossing the plan's year end shows a warning. | Rare and complex in Hive9 | FA B6.4 |
| D-018 | 2026-10-18 | Shell's "ShellAllPlans" field is dropped (greyed out and unused in Shell's form). | No observable purpose | Plan Module doc step 1 |
| D-019 | 2026-10-18 | Records have readable, permanent IDs: P-, C-, I-, A-, L- followed by a number. | Users quote IDs; GSAP's HX key will use the line-item number | Domain model §1 |
| D-020 | 2026-10-18 | In this phase any signed-in user can approve or decline; the screen shows the designated approver and records who acted. | No roles yet (D-004) | Domain model §8 |
| D-021 | 2026-10-18 | Baseline specs for all 13 features are written on `main` first; each feature gets its own branch when planning starts. | Lets Shell review the whole module at once | ADR-0001 |
| D-022 | 2026-10-18 | **Re-submit** sends a Declined activity straight to Submitted (new round at level 1); the owner can **Recall** a Submitted activity back to Created. | Fewer steps; matches Hive9's Re-Submit button | Domain model §2.3, feature 008 |
| D-023 | 2026-10-18 | Moving an activity keeps its status unless its dates must change (then the re-approval rule applies). | Moving is housekeeping, not a change of what was approved | Domain model §2.5 |
| D-024 | 2026-10-18 | Line-item spread amounts are in the local currency; monthly USD values are converted per month with the rounding difference in the last month. | Months always add up to the USD planned cost | Domain model §4.4 |
| D-025 | 2026-10-18 | Teams get a colour (auto-assigned, changeable) used for team pills, as in ZenReach. | Home and lists show teams as coloured pills | Domain model §6 |
| D-026 | 2026-10-18 | Plans show the same derived Approval progress as campaigns and initiatives (all activities in the plan). | One consistent progress indicator on every container | Domain model §2.1 |
| D-027 | 2026-10-18 | One current exchange rate per currency (no yearly or monthly rate tables in this phase); conversions always use it; a rate change never triggers re-approval. | Simplicity; Hive9's monthly FX actuals belong to the Finance module | Domain model §2.5, §4.3; feature 002 |
| D-028 | 2026-10-18 | People are never deleted, only deactivated (they own and approved records). | Keeps history truthful | Feature 003 |
| D-029 | 2026-10-18 | "Multiple choice with % split" attributes are used only at activity and line-item level. | Shell's form has splits only on activity attributes (Country/OU, Brands) | Feature 002 |
| D-030 | 2026-10-18 | Saving an out-of-date copy of a record is refused with who changed it and when (optimistic concurrency), for every record type. | Never silently overwrite a colleague's change | Feature 001 FR-029 |
| D-031 | 2026-10-18 | Undo restores exactly (statuses and open approvals included); a later restore from the recycle bin page resets plans to Draft and activities to Created. | Undo fixes a mistake; a later restore may come back into a changed plan and must be re-approved | Domain model R14, features 001, 011 |
| D-032 | 2026-10-18 | Line items of an activity without dates are not spread (shown in a "No dates" column); negative unassigned balances are never spread. | Dates are only needed to submit | Domain model §4.4 |
| D-033 | 2026-10-18 | Move rules, single and bulk: a campaign moves to another plan of the same year; an initiative to another campaign of the same plan; an activity to another initiative of the same plan. Moves never change dates (a record that would fall outside its new parent's dates is refused/skipped). Copies may target any plan. | Simple, predictable; avoids silent date or approval changes | Features 005, 006, 012 |
| D-034 | 2026-10-18 | A dependent Shell attribute whose parent value has no options set up is not counted as missing. | Avoids Hive9's unfillable required field (FA B10 #5) | Domain model §3 |
| D-035 | 2026-10-18 | Plans list: "My teams" switch is off by default; a plan with no campaigns can be published; a plan copy keeps the source's owner and teams. | Fewest surprises for beginners; defaults to confirm with Shell | Feature 004 |
| D-036 | 2026-10-18 | A developer-only **Sample records** page (sample folders and notes) and a **Shared building blocks** page exist in local and development environments only, outside the navigation. | US5 needs a demonstration record type before plans exist; the gallery lets designers review every shared component | Feature 001 plan, research R23 |
| D-037 | 2026-10-18 | **Undo** after a delete is offered for 10 seconds on screen, one at a time (a new delete replaces the message; the earlier delete stays restorable from the recycle bin), and accepted by the server for 2 minutes after the delete. Undo restores exactly; if someone has meanwhile taken the name, Undo is refused and the item is restored from the recycle bin instead (it then gets the suffix " (restored)"). Later restores follow the status rules of R14. | Undo must stay an exact "oops" fix; the margin covers a paused message and slow networks | Feature 001 FR-026, research R8 |
| D-038 | 2026-10-18 | Light/dark mode and the collapsed sidebar are saved **per person** on the server (default: follow the computer, sidebar expanded). They are display preferences, not business records, so their changes are not written to the activity log and have no version. | Demo people often share one browser during demonstrations; FR-022 and feature 011 scope the log to business records | Feature 001 research R16 |
| D-039 | 2026-10-18 | The activity log stores, for every entry, the record's path at the time, readable labels and values and any comment, and is kept in an append-only file; the recycle bin keeps resolved entries (who undid or restored them, and when). Recorded in ADR-0007. | Entries written by features 002–010 must already carry what feature 011 shows | Feature 001 research R8, ADR-0007 |

## Open questions

| ID | Question | Affects | Default until answered |
|---|---|---|---|
| Q-001 | What exactly are "BCPIR" and "SP&A" plans in Shell's process: a flag on a plan (Shell's form) or a pair of plan views per campaign (ZenReach)? Do they need their own sign-off? Evidence: Hive9 has Measure dashboards "SP&A Lubes" and "SP&A Mobility" (FA §F4.4) and activity categories BCPIR_ and PIR_ (FA §E3.1.2). | 013, 004 | BCPIR is a yes/no on the plan; feature 013 waits |
| Q-002 | What are the real approval workflows (levels, approvers, thresholds) for activities? Do levels depend on planned cost (e.g. level 2 only above 50,000 USD)? | 008 | Ordered levels, any one approver per level approves, no cost thresholds |
| Q-007 | Must a plan be Published before its activities can be submitted for approval? | 008, 004 | No — activities of Draft plans can be submitted |
| Q-008 | Must an activity's planned cost be fully assigned to line items (unassigned balance exactly 0.00) before submitting, or is a positive balance only advice? | 006, 007, 008 | Advice only; a negative balance blocks (D-016) |
| Q-009 | Should the All work filter bar also filter by Shell attributes (e.g. Cluster, Country / OU, GL code), including "Not filled in", as Hive9's filter panel does? | 010 | Only the standard filters |
| Q-003 | Real option lists for every Shell attribute (Cluster/Region, Country/OU, Company code, Cost centre, Spend type → GL code, …) from a production extract | 002 | Fictitious samples |
| Q-004 | Is Shell's fiscal year always January–December? | 004, 010 | Yes (D-011) |
| Q-005 | Which fields of the Shell forms must be "needed to publish/submit" versus optional — confirm the marking in the field catalogue | 004–007 | As in the field catalogue |
| Q-006 | Should plans be hidden from people outside the plan's teams once roles exist (Shell's current rule)? | later | "My teams / All plans" switch, not enforced |
