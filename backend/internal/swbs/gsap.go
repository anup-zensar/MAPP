package swbs

import "context"

type Request struct {
	LineItemID   string
	LineItemName string
	CompanyCode  string
}

type Answer struct {
	LineItemID string
	SwbsCode   string
}

// GSAP answers SWBS requests, one code each; a live FSSI 1523 writer and reader implement it later (ADR-0016).
type GSAP interface {
	RequestCodes(ctx context.Context, requests []Request) ([]Answer, error)
}
