package dashboard

import (
	"bytes"
	"context"
	"strconv"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/deck"
	"mapp/internal/platform/errs"
)

const (
	contentTypePPTX = "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	contentTypePDF  = "application/pdf"
	nameSeparator   = " – "
	footer          = product + " — demo data"
	recordPlan      = "plan"
)

type File struct {
	Name        string
	ContentType string
	Body        []byte
}

// PlanFile is the plan to export, the currency to show amounts in ("" for the person's own) and the format.
type PlanFile struct {
	PlanID, Currency, Format string
}

// PlanExport writes a plan the person may see as a file (FR-023); another plan is not found, as a missing one.
func (s *Service) PlanExport(ctx context.Context, a actor.Actor, f PlanFile) (File, error) {
	var file File
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		year, found, err := s.deps.Plans.YearOf(ctx, f.PlanID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(authz.NotFoundMessage(recordPlan))
		}
		in, err := s.read(ctx, a, request{kind: KindExecutive, query: Query{Year: year, Currency: f.Currency}, plan: f.PlanID})
		if err != nil {
			return err
		}
		if len(in.board.plans) == 0 {
			return errs.NotFound(authz.NotFoundMessage(recordPlan))
		}
		d := in.planDeck()
		file, err = render(d, f.Format, fileName(in.board.plans[0].Name, dayText(in.today)))
		return err
	})
	return file, err
}

// DashboardExport writes the dashboard of r as a file with the same figures as the screen (FR-024).
func (s *Service) DashboardExport(ctx context.Context, a actor.Actor, r request, format string) (File, error) {
	var file File
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		in, err := s.read(ctx, a, r)
		if err != nil {
			return err
		}
		v := in.view()
		title := in.dashboardTitle()
		file, err = render(in.dashboardDeck(v, title), format, fileName(title, yearText(v.Filters.Year), dayText(in.today)))
		return err
	})
	return file, err
}

func render(d deck.Deck, format, name string) (File, error) {
	var buf bytes.Buffer
	if format == FormatPDF {
		err := deck.WritePDF(&buf, d)
		return File{Name: name + ".pdf", ContentType: contentTypePDF, Body: buf.Bytes()}, err
	}
	err := deck.WritePPTX(&buf, d)
	return File{Name: name + ".pptx", ContentType: contentTypePPTX, Body: buf.Bytes()}, err
}

// fileName is "SP&A One – <parts joined by dashes>", without the characters file systems refuse.
func fileName(parts ...string) string {
	name := strings.Join(append([]string{product}, parts...), nameSeparator)
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`/\:*?"<>|`, r) {
			return '-'
		}
		return r
	}, name)
}

func (in *input) newDeck(title string) deck.Deck {
	return deck.Deck{Title: title, Author: product, Created: in.now, Footer: footer}
}

func yearText(year int) string {
	return strconv.Itoa(year)
}
