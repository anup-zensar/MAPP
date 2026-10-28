# ADR-0017: Dashboard charts as plain SVG, PowerPoint and PDF files made by the API

**Status:** Accepted · **Date:** 2026-10-28

## Context
Feature 036 (constitution 4.6.0, D-333) adds executive and regional planning dashboards and the export of
plans and dashboards to PowerPoint (.pptx) and PDF files that download to the user (RFP §8). Two choices shape
the architecture and are hard to reverse:

1. **How charts are drawn on screen.** Constitution F1 asks for one charting approach chosen by ADR, with its
   accessibility, maintenance, licence and bundle size recorded. Until now MAPP had no charts beyond progress
   bars.
2. **Where the files are made.** In the browser (a JavaScript library builds the file from the API's figures)
   or in the API (Go writes the file from the same figures it answers).

The files must hold only what the person may see (constitution VII), every figure must be the server's
(constitution IV), nothing may leave the machine, and every new dependency needs a published version older
than 24 hours and a justification (B1, F1, VII).

## Decision
- **Charts on screen are plain SVG drawn by the web app** from the API's figures: grouped vertical bars only,
  with a legend in words, a text summary for assistive technology and a Table view that is a real table
  (F6). Four chart tokens outside the palette carry the series colours (design system §4.7). No chart
  library.
- **The API makes the files**, with Go's standard library only, in a new platform package
  `internal/platform/deck`: a format-free page model (title, cards, a table, a bar chart, notes) is laid out
  once into positioned boxes, texts and tables, then written either as an Office Open XML presentation
  (`archive/zip` + `encoding/xml`; ECMA-376, 16:9, Arial, real PowerPoint tables) or as a PDF 1.4 with the
  standard Helvetica fonts and WinAnsiEncoding. Text is fitted to its cell with Helvetica's metrics before
  either writer draws it, so both files hold the same pages.
- Files are made per request, returned as an attachment and never stored, logged, e-mailed or shared.

## Alternatives considered
- **A chart library** (Recharts, Chart.js, ECharts): 50–300 kB for one bar chart, a table equivalent still
  needed, and another component to keep accessible. Rejected for now; a later feature with many chart types
  may revisit it with a new ADR.
- **Files in the browser** (`pptxgenjs`, `jspdf`, ~750 kB together): the figures would be assembled on the
  client from several answers (constitution IV), the scope would be enforced by the API and trusted again in
  the browser, and the bundle of every dashboard user would grow. Rejected.
- **A headless browser or LibreOffice on the server** to print HTML to PDF and convert to PowerPoint: a third
  runtime part (constitution IV) and a large install on every laptop. Rejected.
- **Go libraries** (`unidoc/unioffice` — commercial licence; `jung-kurt/gofpdf` — archived; `go-pdf/fpdf` —
  maintained but covers PDF only): rejected; the standard library covers both formats in a few hundred lines
  (the precedent is D-120's hand-written .xlsx writer in `internal/platform/sheet`).

## Consequences
- No new dependency in either part; no change to the bundle of other routes; the dashboards' code loads with
  their own route (F8).
- PDF text is limited to Windows-1252 (Latin) characters; others are written "?" in the PDF only. Embedding a
  Unicode font (Shell's typeface, once supplied) is a later change inside `deck`.
- PowerPoint shapes are positioned absolutely; people can edit the text and the tables after export, while the
  bars are plain rectangles rather than native PowerPoint charts.
- A third format (Word) would be a third writer of the same laid-out pages.
