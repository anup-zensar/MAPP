package swbs_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"mapp/internal/history"
	"mapp/internal/history/historytest"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/authz"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/fake"
	"mapp/internal/platform/ids"
	"mapp/internal/swbs"
	"mapp/internal/swbs/swbstest"
)

var (
	alex  = actor.Actor{ID: "U-1001", Name: "Alex Morgan"}
	start = time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC)
)

type lineItem struct {
	swbs.Request
	ready bool
	code  string
	at    time.Time
}

type fakeLineItems struct {
	items  []*lineItem
	failed bool
}

func (f *fakeLineItems) add(id, name, companyCode string, ready bool) {
	f.items = append(f.items, &lineItem{Request: swbs.Request{LineItemID: id, LineItemName: name, CompanyCode: companyCode}, ready: ready})
}

func (f *fakeLineItems) Candidates(_ context.Context, _ authz.Scope, named []string) ([]swbs.Candidate, error) {
	candidates := []swbs.Candidate{}
	for _, li := range f.items {
		if named == nil || slices.Contains(named, li.LineItemID) {
			candidates = append(candidates, swbs.Candidate{Request: li.Request, Ready: li.ready && li.code == ""})
		}
	}
	return candidates, nil
}

func (f *fakeLineItems) Receive(_ context.Context, _ actor.Actor, answers []swbs.Answer, at time.Time) error {
	if f.failed {
		return errors.New("line item gone")
	}
	for _, a := range answers {
		for _, li := range f.items {
			if li.LineItemID == a.LineItemID {
				li.code, li.at = a.SwbsCode, at
			}
		}
	}
	return nil
}

func (f *fakeLineItems) codeOf(id string) string {
	for _, li := range f.items {
		if li.LineItemID == id {
			return li.code
		}
	}
	return ""
}

type fixture struct {
	service *swbs.Service
	store   swbstest.Store
	items   *fakeLineItems
	history *history.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := fake.NewDB()
	now := func() time.Time { return start }
	generator := ids.NewGenerator(fake.NewCounters(db))
	f := &fixture{store: swbstest.NewStore(db), items: &fakeLineItems{}}
	f.history = history.NewService(db, historytest.NewStore(db), generator, now)
	f.service = swbs.NewService(history.Tools{Tx: db, IDs: generator, Log: f.history, Now: now}, f.store, f.items,
		swbs.NewSimulated(f.store))
	for _, k := range f.service.Kinds() {
		if err := f.history.Register(k); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}
	return f
}

func (f *fixture) log(t *testing.T) []history.Entry {
	t.Helper()
	entries, _, err := f.history.ListLog(t.Context(), alex, history.LogFilter{Limit: 200})
	if err != nil {
		t.Fatalf("ListLog: %v", err)
	}
	return entries
}

func (f *fixture) run(t *testing.T, in swbs.RunInput) swbs.RunView {
	t.Helper()
	run, err := f.service.Run(t.Context(), alex, in)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return run
}

func assertCode(t *testing.T, err error, status int, code string) errs.Error {
	t.Helper()
	got, ok := errors.AsType[*errs.Error](err)
	if !ok || got.Status != status || got.Code != code {
		t.Fatalf("error = %v, want %d %s", err, status, code)
	}
	return *got
}

func (f *fixture) threeReady() {
	f.items.add("L-1006", "Stand build", "GB01", true)
	f.items.add("L-999", "Old stand", "GB01", true)
	f.items.add("L-1005", "Media booking", "NL02", true)
	f.items.add("L-1001", "Agency fee", "DE03", false)
}
