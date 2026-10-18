# GSAP–Hive9 Workshop (summary)

Workshop of 10 Sep 2026 between Shell and Zensar. Only the points that matter for the Plan module and its
future integration are kept; participant names are omitted.

## The business flow (four stages)

1. **Intake and approval (in Hive9).** The marketer creates a **line item** under an activity and ticks
   *Create SWBS code*. After the **activity is approved**, a nightly batch sends the request to GSAP; GSAP
   returns an **SWBS code** for each line item within 24–48 hours. The marketer then raises a Hive9 **PR
   form** (short text, WBS, cost centre, amount, GL code, MoA approver), approved by the MoA (Manual of
   Authority) approver or self-approved.
2. **Requisition to PO (in GSAP).** A requisitioner retypes the PR in GSAP, entering the tracking key
   **"HX" + the Hive9 PR line-item ID** (e.g. HX64307). With an outline agreement the PO is created
   automatically; otherwise a supply-chain analyst builds it; POs of 50,000 USD or more need a CP lead's
   release (the threshold is set in GSAP, not Hive9).
3. **PO to invoice.** PR number, PO number, PO date and amount and a PO link flow back to Hive9 nightly;
   the vendor invoices; invoice details flow back too.
4. **Approve to pay.** The marketer approves the invoice in Hive9; the requisitioner releases the service
   entry in GSAP; payment is made; **actuals** are the payments released.

## Facts for NorthStar's line items

- Inputs to the SWBS request: name, currency, planned amount, contracted amount, contracted spend
  (long term / short term), spend category, **spend type** and **GL code** (13 marketing GL codes; one
  spend type can map to several GL codes), and the **finance budget link**. Only the SWBS code comes back.
- SWBS codes belong to one **company code**; GSAP refuses cross-company charging.
- Greyed fields on the line item are calculated from GSAP data: total PR amount, PO amount, invoiced,
  actuals (paid).
- Dropdown values (GL codes, spend types, …) were configured by the Hive9 vendor; Shell wants its own
  administrator to maintain them in the new application.
- A PR's amount is **blocked** against the budget from the moment the PR is created until it is cancelled.
- SLAs: 2 working days per PR/PO step; 10 working days for invoices.

## Integration constraints

- Interfaces are flat files over SFTP, run as nightly batches: **FSSI 1523** (SWBS request out, response
  in) and **PGSI 941** (PR/PO/invoice in), moved by SAP Cloud Integration (CPI).
- GSAP is frozen by the **Nucleus** S/4HANA programme (cut-over Jan 2027, freeze until about March 2027).
  A temporary manual route (file exports and imports) is needed until then. Not part of NorthStar's
  current phase.
