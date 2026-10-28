# Research: Bulk Actions and Excel Import of Plans (012)

## R1. Re-checking the baseline against Shell's spec 008 and RFP §8

Shell's spec 008 asks for a bulk selection in the Grid with a sticky bar, per-item results, an audit record per
change naming the run, at least 1,000 items per action (FR-001–FR-010); an import of activities from a
plan-specific template with a validation report listing every error by row and column, partial success and
undo (FR-044–FR-051); and exports. RFP §8 goes further: "import the skeleton of plan and upload the updated plan
using excels" and "excel integration or bulk plan uploading". The baseline (activities only, add only, no line
items) covered Shell's 008 but not the RFP. Decision: one workbook layout for the whole plan below the plan
record, used for the template, the export and the import, where a row with an ID updates (D-339, D-340).

Items of Shell's 008 not taken (scope of other features or out of scope): Export All / Selected Columns from a
selection (010 exports the grid), Create Purchase Request (032), Create Template, Link To, Package, snapshots,
calendar feed and PDF.

## R2. Reading .xlsx without a dependency

`platform/sheet` already writes .xlsx with `archive/zip` and `encoding/xml` (D-120); `settings/importfile.go`
reads the first two columns of an option list the same way. The reader added here is general: it reads the
workbook's sheet list (`xl/workbook.xml` + `xl/_rels/workbook.xml.rels`), shared strings, inline strings, numbers
and booleans, and turns date-formatted numbers (built-in formats 14–22, 45–47 and custom formats with d, m or y
outside quotes) into `YYYY-MM-DD`. Limits: 5 MB file, 50 MB per uncompressed part, 5,000 rows and 200 columns
per sheet. Excel, LibreOffice and Google Sheets write these parts; no formula is evaluated (the cached value is
read). Fuzz tests (`FuzzReadXLSX`, `FuzzReadCSV`) prove it never panics on any input (constitution B9).

## R3. One transaction per run, preview discarded

`database.DB.Write` joins an outer `Write`, so a bulk run or an import is one transaction in which each record
goes through its owning service (with its checks, log entries and hooks). The preview and the check run the
same code and end the `Write` with a sentinel error, as `financials` does for its check (D-126, D-177).
Accepted risk: a service that wrote part of a record and then returned a business error would leave that part
in the run; the owning services check everything before they write (verified for the operations used), so a
business refusal leaves the record untouched (FR-022). A storage error aborts the whole run.

## R4. Cell formats (D-138 kept, extended)

Dates `YYYY-MM-DD`, `12 Mar 2026`, `12/03/2026` is refused as ambiguous; Yes/No also true/false, y/n, 1/0;
amounts `12500`, `12500.5`, `12,500.00` (two decimals at most); people by e-mail (an active person); references by
name or ID; multiple choices `A; B`; splits `China 60; Hong Kong 40` or `China; Hong Kong` (even split, R10); the
export writes dates `YYYY-MM-DD`, amounts with two decimals and no grouping, Yes/No, so it reads back
unchanged (SC-004).

## R5. Updating through the single edit

The edit operations take the whole form plus the version (D-203). The app adapters build the form from the
stored record, replace the fields the file or the bulk action sets, and send the stored version — so the edit's
rules, its activity-log diff and its hooks apply unchanged. Whether a workbook row changes anything is decided
before the call by comparing the normalised values of the present columns with the stored record's (the same
functions the export uses), so an unchanged row is never sent.

## R6. Where selection lives

The All work grid already pages, filters and searches every plan of a year and shows one plan's tree when
filtered to it; the plan page's tree is a separate component. Putting checkboxes in both doubles the work for
the same records; the plan tree is left for later (Q-021).

## Decisions for the log

D-338 (bulk actions), D-339 (the plan workbook), D-340 (import semantics), D-341 (storage, access and limits);
questions Q-020 (prerequisite 2 and updates by import) and Q-021 (checkboxes in the plan tree).
