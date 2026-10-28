package swbs

import (
	"cmp"
	"context"
	"fmt"
	"strings"
	"unicode"
)

const (
	noCompanyCode = "0000"
	maxSegment    = 10
)

// Issued counts the codes the runs the store keeps already gave that start with prefix.
type Issued interface {
	CountCodes(ctx context.Context, prefix string) (int, error)
}

// Simulated is ADR-0016's simulated GSAP: SIM-<company code>-<nnnn>, numbered per company code (FA §D7).
type Simulated struct {
	issued Issued
}

func NewSimulated(issued Issued) Simulated {
	return Simulated{issued: issued}
}

func (g Simulated) RequestCodes(ctx context.Context, requests []Request) ([]Answer, error) {
	last := map[string]int{}
	answers := make([]Answer, 0, len(requests))
	for _, r := range requests {
		prefix := "SIM-" + cmp.Or(segment(r.CompanyCode), noCompanyCode) + "-"
		n, counted := last[prefix]
		if !counted {
			count, err := g.issued.CountCodes(ctx, prefix)
			if err != nil {
				return nil, err
			}
			n = count
		}
		n++
		last[prefix] = n
		answers = append(answers, Answer{LineItemID: r.LineItemID, SwbsCode: fmt.Sprintf("%s%04d", prefix, n)})
	}
	return answers, nil
}

// segment keeps the code's letters and digits, upper case and at most ten, as the code's column allows.
func segment(companyCode string) string {
	kept := strings.Map(func(r rune) rune {
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return unicode.ToUpper(r)
		}
		return -1
	}, companyCode)
	return kept[:min(len(kept), maxSegment)]
}
