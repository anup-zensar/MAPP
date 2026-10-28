package purchaserequest

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"mapp/internal/platform/date"
	"mapp/internal/platform/errs"
)

// parseListQuery reads the Purchase requests page's query (contract listPurchaseRequests): every wrong value
// is reported at once; left out, sort is id, page 1 and the page size 50.
func parseListQuery(v url.Values) (ListQuery, error) {
	var problems errs.Problems
	q := ListQuery{Search: v.Get("q"), TypeOptionIDs: v["type"], Vendors: v["vendor"], OwnerIDs: v["owner"],
		Sort: v.Get("sort"), Page: 1, PageSize: pageSizes[0]}
	for _, status := range v["status"] {
		q.Statuses = append(q.Statuses, Status(status))
	}
	if q.Sort == "" {
		q.Sort = "id"
	}
	q.Page = number(&problems, v, "page", q.Page)
	q.PageSize = number(&problems, v, "pageSize", q.PageSize)
	q.TypeNotCompleted = flag(&problems, v, "typeNotCompleted")
	q.VendorNotCompleted = flag(&problems, v, "vendorNotCompleted")
	q.WaitingForGSAP = flag(&problems, v, "waitingForGsap")
	q.CreatedFrom = day(&problems, v, "createdFrom")
	q.CreatedTo = day(&problems, v, "createdTo")
	if len(v.Get("q")) > maxSearch {
		problems.Add("q", msgNameTooLong)
	}
	return q, problems.Err()
}

func number(problems *errs.Problems, v url.Values, name string, fallback int) int {
	text := v.Get(name)
	if text == "" {
		return fallback
	}
	n, err := strconv.Atoi(text)
	if err != nil {
		problems.Add(name, msgPageNumber)
		return fallback
	}
	return n
}

func flag(problems *errs.Problems, v url.Values, name string) bool {
	text := v.Get(name)
	if text == "" {
		return false
	}
	b, err := strconv.ParseBool(text)
	if err != nil {
		problems.Add(name, "Choose true or false.")
	}
	return b
}

func day(problems *errs.Problems, v url.Values, name string) *date.Date {
	text := v.Get(name)
	if text == "" {
		return nil
	}
	d, err := date.Parse(text)
	if err != nil {
		problems.Add(name, msgDate)
		return nil
	}
	return &d
}

// contentDisposition is the header of a download of name: an ASCII file name with "-" for other characters,
// and the UTF-8 name in filename* (RFC 6266, RFC 5987).
func contentDisposition(name string) string {
	ascii := strings.Map(func(r rune) rune {
		if r == '"' || r == '\\' || r < 0x20 || r > 0x7e {
			return '-'
		}
		return r
	}, name)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(name))
}
