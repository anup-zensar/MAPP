# Success criteria check: Planning and Currency Extensions (016)

**Checked**: 2026-10-29, on the worktree branch after T074 – T088 (T095). Evidence is a passing automated test
unless the row says otherwise.

| SC | What it asks | Result | Evidence |
|---|---|---|---|
| SC-001 | Converted amounts of the worked examples match to the cent | Met | `platform/money` `TestConvert_Table_FR015`, `TestConvertParts_AddsUp_FR015`; seed facts in the `plan` and `lineitem` handler tests |
| SC-002 | A 1,000-row GSAP rate file is reviewed in under 10 seconds, in at most 3 steps from Settings | Met — the review takes about 2 ms in the service; Settings › Currencies › Import GSAP rates › choose the file (3 steps) | `settings` `TestCheckRateImport_ThousandRows_016_SC002` (timing test, without `-short` and `-race`) |
| SC-003 | 90 % of first-time planners create a GBP plan with a budget, a campaign budget and an activity planned cost, and find their USD values, within 4 minutes | Not measured — needs a usability session with Shell planners; the journey is covered end to end by the end-to-end lane (T089) | — |
| SC-004 | Budget by month's planned cost per month equals All work Budget & cost | Met | `allwork` `TestBudgetByMonth_EqualsBudgetAndCost_SC004` |
| SC-005 | After a split, the two activities add up to the original | Met | `activity` `TestSplit_SumsUnchanged_SC005` |
| SC-006 | A 5-year range with 25,000 activities shows its first page within the second 010 allows for one year | Met — about 0.3 s | `allwork` `TestQuery_StressFiveYears_016_SC006` (timing test) |
| SC-007 | The Breakdown totals row equals the Planned cost card | Met | `allwork` `TestBreakdown_TotalsEqualCard_SC007`; web `AllWorkPage.breakdown.test.tsx` shows the API's totals row unchanged |
| SC-008 | Every built-in field of the five planning forms shows an ⓘ; an edited text shows on the next page | Met | `PlanFieldHelp.test.tsx`, `RecordFieldHelp.test.tsx`, `ActivityFieldHelp.test.tsx`, `LineItemFieldHelp.test.tsx` (one ⓘ per labelled field, none for an emptied text); Field help is read with every page load (`useFieldHelp`) |
| SC-009 | Every new screen and dialog passes axe and works with the keyboard | Met in component tests (axe in `AllWorkPage.breakdown.test.tsx`, `PlannedCostByCard.test.tsx` and the earlier 016 screens' tests); the end-to-end axe pass is T089 | `src/testing/axe.ts` checks |
