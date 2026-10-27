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

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
)

// The names of the records as they read mid-sentence: "Not saved — Alex Morgan changed this model".
const (
	modelNoun        = "model"
	activityTypeNoun = "activity type"
	attributeNoun    = "Shell attribute"
	optionNoun       = "option"
	currencyNoun     = "currency"
	budgetNoun       = "finance budget"
	itemNoun         = "finance budget item"
)

const (
	reloadMessage    = "Reload the page and try again."
	enterNameMessage = "Enter a name."
	inUseMessage     = "'%s' is in use (%s) — set it inactive instead."
)

const (
	deactivatedVerb = "deactivated"
	reactivatedVerb = "reactivated"
)

type Service struct {
	tx      Transactor
	repo    Store
	ids     *ids.Generator
	history *history.Service
	now     clock.Clock
	sources []ReferenceSource
	feed    RateFeed // nil reads the GSAP daily rate file
}

func NewService(tools history.Tools, repo Store) *Service {
	return &Service{tx: tools.Tx, repo: repo, ids: tools.IDs, history: tools.Log, now: tools.Now}
}

// AddReferenceSource is for internal/app while the API starts, before it serves requests.
func (s *Service) AddReferenceSource(source ReferenceSource) {
	s.sources = append(s.sources, source)
}

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

func countActive[T any](list []T, active func(T) bool) int {
	n := 0
	for _, record := range list {
		if active(record) {
			n++
		}
	}
	return n
}

// checkText takes text already trimmed.
func checkText(problems *errs.Problems, field, text, empty string, max int) {
	if text == "" {
		problems.Add(field, empty)
		return
	}
	checkLength(problems, field, text, max)
}

func checkLength(problems *errs.Problems, field, text string, max int) {
	if utf8.RuneCountInString(text) > max {
		problems.Add(field, tooLongMessage(max))
	}
}

func tooLongMessage(max int) string {
	return fmt.Sprintf("Use %s characters or fewer.", thousands(int64(max)))
}

func checkVersion(problems *errs.Problems, version *int) {
	if version == nil || *version < 1 {
		problems.Add("version", reloadMessage)
	}
}

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

func sameName(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

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

func inUseError(name string, usage Usage) error {
	return errs.InUse(fmt.Sprintf(inUseMessage, name, midSentence(usage.Summary)))
}

func (s *Service) logCreated(ctx context.Context, a actor.Actor, kind, id, name, details string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kind, RecordID: id, RecordName: name, Action: history.ActionCreated, Details: details,
	})
	return err
}

func (s *Service) logChanged(ctx context.Context, a actor.Actor, kind, id, name string, changes []history.Change, details string) error {
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind: kind, RecordID: id, RecordName: name, Action: history.ActionChanged, Changes: changes, Details: details,
	})
	return err
}

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

func jsonOf(v any) json.RawMessage {
	encoded, _ := json.Marshal(v) // cannot fail: callers pass a bool, a string or a list of strings
	return encoded
}

func withoutFields(changes []history.Change, fields ...string) []history.Change {
	kept := []history.Change{}
	for _, c := range changes {
		if !slices.Contains(fields, c.Field) {
			kept = append(kept, c)
		}
	}
	return kept
}

func unique(list []string) []string {
	kept := []string{}
	for _, item := range list {
		if item != "" && !slices.Contains(kept, item) {
			kept = append(kept, item)
		}
	}
	return kept
}

// midSentence lowers the first letter only when the second is lower too: "Region" → "region", "GL code" stays.
func midSentence(label string) string {
	first, size := utf8.DecodeRuneInString(label)
	second, _ := utf8.DecodeRuneInString(label[size:])
	if size == 0 || !unicode.IsUpper(first) || !unicode.IsLower(second) {
		return label
	}
	return string(unicode.ToLower(first)) + label[size:]
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return "1 " + singular
	}
	return thousands(int64(n)) + " " + plural
}

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

// compareIDs orders ids as the store lists them: SA-999 before SA-1001.
func compareIDs(a, b string) int {
	return cmp.Or(cmp.Compare(len(a), len(b)), cmp.Compare(a, b))
}

func compareNames(a, b string) int {
	return cmp.Compare(strings.ToLower(a), strings.ToLower(b))
}

// restoreName refuses an Undo whose name another record has taken, since Undo puts the record back exactly;
// a Restore renames it "<name> (restored)", "<name> (restored 2)", … and says so in a note.
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
