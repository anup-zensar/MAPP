package swbs

import (
	"fmt"
	"strings"
)

const (
	runKind         = "swbsRun"
	runKindSingular = "SWBS run"
	runKindPlural   = "SWBS runs"
	ranVerb         = "ran"

	sourceSimulatedText = "Simulated GSAP"
	sourceDemoText      = "Demo data (simulated GSAP)"

	nothingSentText  = "Nothing was ready to send."
	returnedOneText  = "Simulated GSAP returned 1 SWBS code: %s."
	returnedManyText = "Simulated GSAP returned %d SWBS codes: %s."
	codeText         = "%s (%s) %s"
	lineItemText     = "%s (%s)"

	msgEmptyList = "Name at least one line item, or leave the list out to send every ready line item."
	msgTooMany   = "Send at most 200 line items at a time."
	msgBadID     = "Use a line-item ID such as L-1006."
	msgLimit     = "Ask for 1 to 100 runs."
	msgNotFound  = "This line item doesn't exist or you can't see it."
	msgNotReady  = "Not sent — these line items aren't ready to send to the simulated GSAP: %s. Reload to see their SWBS status."
)

func (s Source) text() string {
	if s == SourceDemo {
		return sourceDemoText
	}
	return sourceSimulatedText
}

func detailsOf(codes []Code) string {
	if len(codes) == 0 {
		return nothingSentText
	}
	listed := make([]string, 0, len(codes))
	for _, c := range codes {
		listed = append(listed, fmt.Sprintf(codeText, c.LineItemName, c.LineItemID, c.SwbsCode))
	}
	if len(codes) == 1 {
		return fmt.Sprintf(returnedOneText, listed[0])
	}
	return fmt.Sprintf(returnedManyText, len(codes), strings.Join(listed, "; "))
}
