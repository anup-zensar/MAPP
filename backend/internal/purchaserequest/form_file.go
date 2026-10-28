package purchaserequest

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/sheet"
)

const (
	FormatXLSX      = "xlsx"
	FormatCSV       = "csv"
	contentTypeXLSX = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	contentTypeCSV  = "text/csv"
	noDocuments     = "Routed by MAPP — no supporting documents are attached"
)

var formColumns = []string{ //nolint:gochecknoglobals // the PR form's columns, in the spec's order
	"PR ID", "PR name", "PR line ID", "HX key", "Short text", "Vendor", "Vendor code", "Cost centre", "SWBS code",
	"GL code", "Amount", "Currency", "Delivery date", labelMoA, "MoA approver e-mail", "Routing type", "Changed",
}

// FormRequest asks for the PR form of one routing of a purchase request, as xlsx or csv.
type FormRequest struct {
	PurchaseRequestID, RoutingID, Format string
}

// Form is the PR form of one routing of the purchase request, as routed, as a spreadsheet or a CSV file.
func (s *Service) Form(ctx context.Context, _ actor.Actor, req FormRequest) (File, error) {
	id, routingID, format := req.PurchaseRequestID, req.RoutingID, req.Format
	if format != FormatXLSX && format != FormatCSV {
		return File{}, errs.Validation(errs.FieldError{Field: "format", Message: msgFormFormat})
	}
	var file File
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		pr, err := s.visible(ctx, id)
		if err != nil {
			return err
		}
		routings, err := s.repo.Routings(ctx, pr.ID)
		if err != nil {
			return err
		}
		for _, r := range routings {
			if r.ID == routingID {
				file, err = formFile(pr, r, format)
				return err
			}
		}
		return errs.NotFound(msgRoutingNotFound)
	})
	return file, err
}

func formFile(pr PurchaseRequest, r Routing, format string) (File, error) {
	notes := [][2]string{
		{"Purchase request", pr.ID + " – " + pr.Name},
		{"Routing", routingWords[r.Type]},
		{"Routed at", r.RoutedAt.UTC().Format("02 Jan 2006 15:04") + " UTC"},
		{"Routed by", r.RoutedBy.Name},
		{"Recipients", strings.Join(r.Recipients, ", ")},
		{"Note", noDocuments},
	}
	table := formTable(r)
	file := File{Name: fmt.Sprintf("MAPP PR form – %s – %s – %s.%s", pr.ID, routingWords[r.Type],
		date.Format(date.Today(r.RoutedAt)), format)}
	var body bytes.Buffer
	if format == FormatCSV {
		file.ContentType = contentTypeCSV
		if err := sheet.WriteCSV(&body, notes, table); err != nil {
			return File{}, err
		}
		file.Body = body.Bytes()
		return file, nil
	}
	file.ContentType = contentTypeXLSX
	rows := make([][]sheet.Cell, 0, len(notes)+len(table)+1)
	for _, n := range notes {
		rows = append(rows, []sheet.Cell{sheet.Bold(n[0]), sheet.Text(n[1])})
	}
	rows = append(append(rows, []sheet.Cell{}), table...)
	if err := sheet.WriteXLSX(&body, sheet.Workbook{Sheets: []sheet.Sheet{{Name: "PR form", Rows: rows}}}); err != nil {
		return File{}, err
	}
	file.Body = body.Bytes()
	return file, nil
}

func formTable(r Routing) [][]sheet.Cell {
	heading := make([]sheet.Cell, 0, len(formColumns))
	for _, c := range formColumns {
		heading = append(heading, sheet.Bold(c))
	}
	table := [][]sheet.Cell{heading}
	for _, row := range r.Form {
		changed := "No"
		if row.Changed {
			changed = "Yes"
		}
		table = append(table, []sheet.Cell{
			sheet.Text(row.PurchaseRequestID), sheet.Text(row.PurchaseRequestName), sheet.Text(row.LineID),
			sheet.Text(row.HXKey), sheet.Text(row.ShortText), sheet.Text(row.Vendor), sheet.Text(row.VendorCode),
			sheet.Text(row.CostCentre), sheet.Text(row.SwbsCode), sheet.Text(row.GLCode), sheet.Money(row.Amount),
			sheet.Text(row.Currency), sheet.Date(row.DeliveryDate), sheet.Text(row.MoAApproverName),
			sheet.Text(row.MoAApproverEmail), sheet.Text(routingWords[r.Type]), sheet.Text(changed),
		})
	}
	return table
}
