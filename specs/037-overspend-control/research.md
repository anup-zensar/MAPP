# Research: Overspend Control

Decisions taken while planning feature 037, with the alternatives considered. The product owner was not in
the loop; each decision is recorded in the decision log (D-346 – D-349) and the two that most need Shell's
confirmation are open questions (Q-024, Q-025).

## R1 — What "going over a budget" means

**Decision**: the **planned cost** below the budget (domain model §4.2): the sum of the activities' planned
cost of a plan, campaign or initiative, in the plan currency; for a global budget, the planned cost of each
plan it allocates to converted to USD at the current rate and added up — the figure the global budget page
already shows as Planned cost (014 FR-032, D-303).

**Rationale**: RFP §8 speaks of overspending against approved budgets; the setting's label says "planned
cost". Planned cost is the figure Shell plans spend with; it already turns red above the budget ("Budget
used", "Left to plan").

**Alternatives**: (a) also refuse over-allocation (children's budgets above the parent's) — rejected for now:
client 004 FR-049 requires over-allocation to be allowed with a warning and D-015/D-235 stay true for it;
asked as Q-024. (b) Count line items above the activity's planned cost as planned cost — rejected: it would
create a second "planned cost" that no screen shows (server is the source of truth for figures) and line items
are allocations of the activity's planned cost (§4.3); the commitment risk is closed at the purchase request
instead (R7).

## R2 — The refusal rule

**Decision**: after a save, for every budget set to Refuse whose planned cost is above its amount, the save is
refused when, compared with just before the save, the planned cost went up, the amount went down, the budget
did not exist (a new record, a copy, or a budget typed where there was none), or both. Saves that keep or
lower the cost and keep or raise the amount pass, even when the budget stays over.

**Rationale**: matches client 011 FR-024 ("below zero, or further below zero") and its edge case ("Hard Limit
switched on while rows are already negative: allowed … any further increase on them is refused") and FR-025
(lowering the budget below the planned cost is refused, stating the minimum). Choosing Refuse while over is a
change of the setting only: neither the planned cost nor the amount moves, so it passes.

**Alternatives**: refuse switching to Refuse while over — rejected (client 011 allows it, and it would stop an
owner from freezing a budget that is already over).

## R3 — Where the check runs: one guard around every planning write

**Decision**: package `internal/overspend` provides a **guarded transactor**: the transactor given to the
services that change planned cost or budgets (plans, campaigns and initiatives, activities, global budgets and
allocations, and history for restore, Undo and delete). Its outermost `Write` reads every budget with its
planned cost before the work, runs the work, reads them again and refuses (rolling the whole transaction
back) when R2's rule is broken. Nested `Write`s join the outer one and are not checked twice.

**Rationale**: about twenty write paths change planned cost (activity create, update, move, copy, copy-many,
split; initiative and campaign moves and copies; plan copy with contents; restore and Undo of every level;
plan, campaign, initiative and global budget edits; allocation create, update and remove), and features in
progress (012 bulk actions and import) will add more. One choke point covers them all, fails closed for
future paths, and keeps every service's code as it is. The check runs inside the same transaction as the
write, after it, so it sees exactly what would be committed, and the rule lives in one service.

**Alternatives**: a check call in each service method — rejected: twenty call sites, each needing a "before"
figure, easy to miss in a new path. A database trigger — rejected: business rules belong in services (IV, B2)
and the message needs names and money formatting.

**Cost**: two aggregate reads before and two after each planning write (plans, campaigns and initiatives with
their activities' planned cost in one query; global budgets with their allocated plans in another). Writes
are serialised already (`database.DB.Write` takes the write lock), so the check also covers two concurrent
saves that together go over. At MAPP's size (hundreds of plans, thousands of activities) the reads are
milliseconds; `apibench` shows the per-operation query count.

## R4 — The answer of a refused save

**Decision**: 409 with the new error code `over_budget`, a message naming every budget ("Not saved — the
planned cost of campaign 'Helix launch' would be 712,000.00 USD, over its budget of 700,000.00 USD by
12,000.00 USD. Its owner refuses overspending.") and a list `overBudget` of `{budget: RecordRef, amount,
plannedCost, overBy}` in the budget's currency. Money is formatted with two decimals and grouped thousands as
everywhere else.

**Rationale**: one error envelope (B6) extended with an optional property is backward compatible; the code
lets the web app present it as a refusal rather than a field problem; the list lets screens name each budget.

## R5 — Organisation default

**Decision**: a singleton record "Overspend control" (`overspend_configuration`, id `overspend`), read by
every signed-in person and changed with `settings.manage`, like Settings › Purchase requests (032). Without a
stored row it is Warn. New plans, campaigns, initiatives and global budgets start with it when their request
does not name a setting; copies keep their source's; changing it changes no existing budget (D-347).

**Alternatives**: a third value "Organisation default" inherited by budgets — rejected for now: one change
would silently turn many owners' budgets into hard stops (no surprises, IX) and every budget would need a
second "effective" value; asked as Q-025.

## R6 — Who may change a budget's setting

**Decision**: on plans, campaigns and initiatives: the record's owner or a person holding
`globalBudgets.manage` in the plan's scope (D-348); on global budgets and allocations: whoever may edit them
(they already need `globalBudgets.manage`). The rule is one function in `internal/overspend`
(`ControlOf`/`RequireChange`), giving the read-only reason the forms show and the 403 the API answers. No new
permission: a new one would need a migration of the role permission constraint that parallel features also
change.

## R7 — Purchase-request gate

**Decision**: `purchaserequest.LineItemFacts` gains `ActivityOverBy` (the activity's negative unassigned
planned cost as a positive amount in the plan currency, nil when not negative), filled by the wiring from the
line-item service's summaries (the same figure the activity's unassigned row shows, §4.3). `gates` adds one
problem per such activity, group `activity`. It applies to Submit, Re-submit, Save and re-submit and Submit
changes because they all run `gates` (032 FR-021), whatever the budgets' setting (D-349).

## R8 — Showing the refusal before saving

**Decision**: the activity form receives `overspendLimits` — for every budget above the activity set to Refuse,
its name and left to plan in the plan currency (a global budget's converted from USD at the current rate) —
and refuses on the form, with the API's wording, a planned cost whose increase is more than the smallest left
to plan. The budget forms and allocation dialogs compare the typed budget with the planned cost they already
show. The server still decides; its `over_budget` answer is shown in the form's error summary.
