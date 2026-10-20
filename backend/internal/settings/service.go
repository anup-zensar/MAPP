package settings

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"northstar/internal/history"
	"northstar/internal/platform/actor"
	"northstar/internal/platform/clock"
	"northstar/internal/platform/errs"
	"northstar/internal/platform/ids"
	"northstar/internal/platform/store"
)

// The mid-sentence names of the Settings records: in conflict messages ("Not saved — Alex Morgan changed
// this model"), in the activity log and in the recycle bin (plan "Package settings").
const (
	modelNoun        = "model"
	activityTypeNoun = "activity type"
	attributeNoun    = "Shell attribute"
	optionNoun       = "option"
	currencyNoun     = "currency"
	budgetNoun       = "finance budget"
	itemNoun         = "finance budget item"
)

// Messages of the plan's "User-visible messages" table that every record type shares.
const (
	reloadMessage    = "Reload the page and try again."
	enterNameMessage = "Enter a name."
	inUseMessage     = "'%s' is in use (%s) — set it inactive instead."
)

// The verbs of the activity-log entries of Set inactive and Reactivate (data model §12).
const (
	deactivatedVerb = "deactivated"
	reactivatedVerb = "reactivated"
)

// Service holds the rules of Settings: models, activity types, Shell attributes and their options,
// currencies and finance budgets (feature 002). Every method that a person calls takes the actor, works in
// one transaction and logs every change; the readers of readers.go and form.go serve later features.
type Service struct {
	tx      store.Transactor
	repo    Repository
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
	sources []ReferenceSource
	feed    RateFeed // how rate files are read (016 FR-008); nil reads the GSAP daily rate file
}

// NewService returns the Settings service. It keeps the records in repo, gives them IDs with generator,
// logs their changes, deletes them and reads the recycle bin through hist, works in transactions of tx and
// reads the time from now.
func NewService(tx store.Transactor, repo Repository, generator *ids.Generator, hist *history.Service, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, ids: generator, history: hist, now: now}
}

// AddReferenceSource adds a source of the records outside Settings that use Settings records (D-042):
// plans, campaigns, initiatives, activities, line items and people. internal/app calls it for features
// 003–007 while the API starts, before it serves requests.
func (s *Service) AddReferenceSource(source ReferenceSource) {
	s.sources = append(s.sources, source)
}

// Overview counts the active records of every Settings section, for the cards of the Settings page
// (FR-001). Every signed-in person may see Settings (FR-008), so a is not used yet.
func (s *Service) Overview(ctx context.Context, a actor.Actor) (SettingsOverview, error) {
	var overview SettingsOverview
	err := s.tx.Read(ctx, func(ctx context.Context) error {
		models, err := s.repo.ListModels(ctx)
		if err != nil {
			return err
		}
		types, err := s.repo.ListActivityTypes(ctx)
		if err != nil {
			return err
		}
		attributes, err := s.repo.ListAttributes(ctx)
		if err != nil {
			return err
		}
		currencies, err := s.repo.ListCurrencies(ctx)
		if err != nil {
			return err
		}
		items, err := s.repo.ListFinanceBudgetItems(ctx)
		if err != nil {
			return err
		}
		overview = SettingsOverview{
			Models:             countActive(models, func(m Model) bool { return m.Active }),
			ActivityTypes:      countActive(types, func(t ActivityType) bool { return t.Active }),
			ShellAttributes:    countActive(attributes, func(a Attribute) bool { return a.Active }),
			Currencies:         countActive(currencies, func(c Currency) bool { return c.Active }),
			FinanceBudgetItems: countActive(items, func(i FinanceBudgetItem) bool { return i.Active }),
		}
		return nil
	})
	return overview, err
}

// countActive counts the records of list that active says are active.
func countActive[T any](list []T, active func(T) bool) int {
	n := 0
	for _, record := range list {
		if active(record) {
			n++
		}
	}
	return n
}

// checkText adds the problem of a required text, already trimmed, to problems: empty is "empty", longer
// than max characters (characters, not bytes) is "Use <max> characters or fewer.".
func checkText(problems *errs.Problems, field, text, empty string, max int) {
	if text == "" {
		problems.Add(field, empty)
		return
	}
	checkLength(problems, field, text, max)
}

// checkLength adds "Use <max> characters or fewer." to problems when text has more than max characters.
func checkLength(problems *errs.Problems, field, text string, max int) {
	if utf8.RuneCountInString(text) > max {
		problems.Add(field, tooLongMessage(max))
	}
}

// tooLongMessage is the message of a text longer than max characters: "Use 2,000 characters or fewer.".
func tooLongMessage(max int) string {
	return fmt.Sprintf("Use %s characters or fewer.", thousands(int64(max)))
}

// checkVersion adds the problem of a missing version to problems: every update sends the version it
// started from, a whole number from 1 (001 research R5).
func checkVersion(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

// checkStatus reads a StatusUpdate: both the new status and the version are needed.
func checkStatus(update StatusUpdate) (active bool, version int, err error) {
	var problems errs.Problems
	if update.Active == nil {
		problems.Add("active", reloadMessage)
	}
	checkVersion(&problems, update.Version)
	if err := problems.Err(); err != nil {
		return false, 0, err
	}
	return *update.Active, *update.Version, nil
}

// sameName reports whether two names are the same name: equal ignoring letter case and surrounding
// spaces (FR-005).
func sameName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// findNamed returns the record of list, other than exceptID, whose name is name ignoring letter case and
// surrounding spaces, and false when there is none. key gives a record's id and name.
func findNamed[T any](list []T, name, exceptID string, key func(T) (id, name string)) (T, bool) {
	for _, record := range list {
		id, recordName := key(record)
		if id != exceptID && sameName(recordName, name) {
			return record, true
		}
	}
	var none T
	return none, false
}

// inUseError refuses the delete of the record name, which usage says is in use: "'Lubricants 1.0' is in use
// (used by 2 plans) — set it inactive instead." (FR-004, D-043).
func inUseError(name string, usage Usage) error {
	return errs.InUse(fmt.Sprintf(inUseMessage, name, midSentence(usage.Summary)))
}

// logCreated writes the "created" entry of a new record.
func (s *Service) logCreated(ctx context.Context, a actor.Actor, kind, id, name, details string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kind, RecordID: id, RecordName: name, Action: history.ActionCreated, Details: details,
	})
	return err
}

// logChanged writes one "changed" entry listing changes, the fields of the record that changed (FR-006).
func (s *Service) logChanged(ctx context.Context, a actor.Actor, kind, id, name string, changes []history.Change, details string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kind, RecordID: id, RecordName: name, Action: history.ActionChanged, Changes: changes, Details: details,
	})
	return err
}

// logStatus writes the entry of Set inactive (active false) or Reactivate (active true): one
// "statusChanged" entry with the change of the field "active" (data model §12).
func (s *Service) logStatus(ctx context.Context, a actor.Actor, kind, id, name string, active bool) error {
	verb := deactivatedVerb
	if active {
		verb = reactivatedVerb
	}
	change := history.Change{Field: "active", Old: jsonOf(!active), New: jsonOf(active)}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kind, RecordID: id, RecordName: name, Action: history.ActionStatusChanged, Verb: verb,
		Changes: []history.Change{change},
	})
	return err
}

// jsonOf encodes a value that always encodes: a bool, a string, a list of strings.
func jsonOf(v any) json.RawMessage {
	encoded, _ := json.Marshal(v) // cannot fail for these values
	return encoded
}

// withoutFields returns changes without the changes of the fields named, which are bookkeeping of the
// record rather than something a person changed (a currency's rateChangedAt).
func withoutFields(changes []history.Change, fields ...string) []history.Change {
	kept := []history.Change{}
	for _, c := range changes {
		if !slices.Contains(fields, c.Field) {
			kept = append(kept, c)
		}
	}
	return kept
}

// unique returns list without empty and repeated entries, in their first order. The result is never nil.
func unique(list []string) []string {
	kept := []string{}
	for _, item := range list {
		if item != "" && !slices.Contains(kept, item) {
			kept = append(kept, item)
		}
	}
	return kept
}

// midSentence writes a label as it reads in the middle of a sentence: the first letter in lower case when
// the second is lower case too ("Region" → "region", "Used by 2 plans" → "used by 2 plans"), and as it is
// otherwise ("GL code", "PIR").
func midSentence(label string) string {
	first, size := utf8.DecodeRuneInString(label)
	second, _ := utf8.DecodeRuneInString(label[size:])
	if size == 0 || !unicode.IsUpper(first) || !unicode.IsLower(second) {
		return label
	}
	return string(unicode.ToLower(first)) + label[size:]
}

// plural writes "<n> <noun>" with the singular for 1 everywhere: "1 model", "2 models".
func plural(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return thousands(int64(n)) + " " + plural
}

// thousands writes a whole number with thousands separators: 250000 → "250,000".
func thousands(n int64) string {
	digits := strconv.FormatInt(n, 10)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign, digits = "-", digits[1:]
	}
	var b strings.Builder
	for i, r := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return sign + b.String()
}

// compareIDs orders ids as the store lists them: shorter ids first, then alphabetically, so SA-999 comes
// before SA-1001.
func compareIDs(a, b string) int {
	return cmp.Or(cmp.Compare(len(a), len(b)), cmp.Compare(a, b))
}

// compareNames orders names the way people sort them: ignoring letter case.
func compareNames(a, b string) int {
	return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
}

// restoreName applies the name rule of a record coming back from the recycle bin (data model §11, 001's
// samples): when clash finds another record with the name, an Undo, which must put the record back
// exactly, is refused with name_taken, and a Restore renames it to "<name> (restored)", "<name> (restored
// 2)", … and says so in a note. It returns the name to restore and the notes.
func restoreName(rc history.RestoreContext, singular, name string, clash func(name string) (other string, taken bool)) (string, []string, error) {
	other, taken := clash(name)
	if !taken {
		return name, nil, nil
	}
	if rc.Undo {
		return "", nil, errs.NameTaken(fmt.Sprintf(undoNameTakenMessage, singular, other))
	}
	candidate := name + " (restored)"
	for n := 2; ; n++ {
		if _, taken := clash(candidate); !taken {
			break
		}
		candidate = fmt.Sprintf("%s (restored %d)", name, n)
	}
	return candidate, []string{fmt.Sprintf(renamedNote, candidate)}, nil
}
