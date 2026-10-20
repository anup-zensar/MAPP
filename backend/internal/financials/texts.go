package financials

import (
	"fmt"
	"strconv"
)

// The texts of feature 015 that the API answers: plan "User-visible messages", "Simulated GSAP extract",
// "API operations" and data model §4.2. Every text a person reads is here, so the wording stays one.

// The not-linked reasons (data model §4.2): the reason of the first key a transaction carries.
const (
	reasonLineItemID    = "No line item %s — check the line item ID in GSAP."
	reasonTrackingKey   = "No line item for HX key %s — check the tracking key in GSAP."
	reasonPONotLinked   = "PO %s isn't linked to a line item yet."
	reasonPOSeveral     = "PO %s has lines on several line items — the row needs its PO line."
	reasonSwbsNone      = "SWBS code %s isn't on any linked transaction yet."
	reasonSwbsSeveral   = "SWBS code %s is on transactions of several line items."
	reasonNoReference   = "No line item reference (line item ID, HX key, PO number or SWBS code)."
	reasonLineItemInBin = "Line item %s is in the recycle bin — its amounts come back when it's restored."
)

// The column rules of the simulated GSAP extract (plan, "Simulated GSAP extract").
const (
	msgType          = "Type must be PO, INVOICE, PAYMENT or ACCRUAL."
	msgDocumentEmpty = "Document number is missing."
	msgDocumentForm  = `Use 20 letters, digits, "-" or "/" at most.`
	msgLine          = "Line must be a number of up to 6 digits."
	msgPONumber      = `PO number: use 20 letters, digits, "-" or "/" at most.`
	msgPOLine        = "PO line must be a number of up to 6 digits."
	msgPostingDate   = "Posting date must be a date like 2026-03-31."
	msgAccrualMonth  = "Accrual month must be a month like 2026-09."
	msgAmount        = "Amount must be a number like 1234.50 (a minus sign for credits)."
	msgCurrencyEmpty = "Currency is missing."
	msgCurrency      = "Currency '%s' isn't set up in Settings — an administrator adds it under Currencies."
	msgVendorCode    = "Use 20 characters or fewer."
	msgVendorName    = "Use 120 characters or fewer."
	msgDescription   = "Use 200 characters or fewer."
	msgSwbsCode      = "SWBS code: use 40 characters or fewer, without spaces."
	msgTrackingKey   = "Tracking key must look like HX1003."
	msgLineItemID    = "Line item ID must look like L-1003."
	msgRepeatedKey   = "Row %d repeats %s from row %d — keep one."
)

// The whole-file refusals of the simulated GSAP extract.
const (
	msgFileName       = "Choose a .csv file — GSAP extracts are imported as CSV."
	msgFileUnreadable = "The file couldn't be read — choose it again."
	msgFileTooLarge   = "The file is larger than 700 KB — split it into smaller files."
	msgFileNotUTF8    = `The file isn't UTF-8 text — save it as "CSV UTF-8" and try again.`
	msgFileNoRows     = "The file has no rows."
	msgFileTooMany    = "The file has more than 5,000 rows — split it into smaller files."
	msgFileColumns    = "The first row must name the columns. Missing: %s."
)

// The refusals of the operations (plan, "API operations"; 014 messages 1–4 for access).
const (
	msgRecordGone       = "This %s no longer exists — someone may have deleted it."
	msgTransactionGone  = "This GSAP transaction doesn't exist."
	msgNoLineItem       = "No line item %s."
	msgLineItemInBin    = "Line item %s is in the recycle bin — restore it first."
	msgLineItemIDForm   = "Enter a line item ID like L-1007."
	msgLinkedByGSAP     = "Linked by GSAP data (%s) — correct it in GSAP."
	msgLinkedByPerson   = "Linked by %s on %s — unlink it first."
	msgOnlyManualUnlink = "Only links made by a person can be undone here."
	msgEnterVersion     = "Send the version you started from."
	msgNoAccessRecord   = "You don't have access to this %s. Ask an administrator if you need it."
	msgChooseScope      = "Choose My teams or Everything."
	msgChooseListed     = "Choose one of the listed values."
	msgSearchTooLong    = "Use 200 characters or fewer."
	msgChooseYear       = "Choose a year from 2000 to 2100."
	msgChoosePlan       = "Choose a plan of the list."
	msgChooseRecord     = "Choose a plan, campaign, initiative, activity or line item."
	msgChooseSort       = "Choose a column and a direction, such as postingDate:desc."
	msgChoosePage       = "Choose a page from 1."
	msgChoosePageSize   = "Choose 50, 100 or 200 rows per page."
	transactionNoun     = "GSAP transaction"
)

// The words of how a transaction was matched (plan, "Matched by").
const (
	matchedLineItemID  = "line item ID %s"
	matchedTrackingKey = "HX key %s"
	matchedPONumber    = "PO number %s"
	matchedSwbsCode    = "SWBS code %s"
)

// The activity-log texts (FR-027, data model §7).
const (
	importKindSingular      = "GSAP extract"
	importKindPlural        = "GSAP extracts"
	transactionKindSingular = "GSAP transaction"
	transactionKindPlural   = "GSAP transactions"
	importedVerb            = "imported"
	linkedVerb              = "linked"
	unlinkedVerb            = "unlinked"
	lineItemLabel           = "Line item"
	notLinkedText           = "Not linked"
	relinkedText            = "%s earlier %s now linked"
)

// The feed names (FR-008).
const (
	csvFeedName  = "Simulated GSAP extract"
	demoFeedName = "Demo data (simulated GSAP)"
)

// countsText is "<rows> rows: <new> new, <updated> updated, <unchanged> unchanged, <refused> refused ·
// <notLinked> not linked", with " · <n> earlier transactions are now linked" when an import linked some
// (data model §7, US4 scenarios 2, 7 and 8).
func countsText(c ImportCounts) string {
	text := fmt.Sprintf("%s: %d new, %d updated, %d unchanged, %d refused · %d not linked",
		plural(c.Rows, "row", "rows"), c.New, c.Updated, c.Unchanged, c.Refused, c.NotLinked)
	if c.Relinked > 0 {
		text += " · " + relinkedSentence(c.Relinked)
	}
	return text
}

// relinkedSentence is "2 earlier transactions are now linked" (US4 scenario 8).
func relinkedSentence(n int) string {
	if n == 1 {
		return "1 earlier transaction is now linked"
	}
	return fmt.Sprintf(relinkedText, strconv.Itoa(n), "transactions are")
}

// plural writes n with the singular or plural noun: "1 row", "4 rows".
func plural(n int, singular, pluralNoun string) string {
	if n == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(n) + " " + pluralNoun
}

// matchedWords names how a transaction was linked by the rules, for "Linked by GSAP data (HX key HX1006)".
func matchedWords(t Transaction) string {
	switch t.MatchedBy {
	case MatchedByLineItemID:
		return fmt.Sprintf(matchedLineItemID, t.SentLineItemID)
	case MatchedByTrackingKey:
		return fmt.Sprintf(matchedTrackingKey, t.TrackingKey)
	case MatchedByPONumber:
		return fmt.Sprintf(matchedPONumber, t.PONumber)
	case MatchedBySwbsCode:
		return fmt.Sprintf(matchedSwbsCode, t.SwbsCode)
	}
	return ""
}
