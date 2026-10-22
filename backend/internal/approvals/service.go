package approvals

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/people"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/clock"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/ids"
	"mapp/internal/platform/store"
)

// activityKind is feature 006's history kind of activities: approval actions are logged on the activity
// (FR-026, plan "Activity-log entries").
const activityKind = "activity"

// maxCommentLength is the longest comment, reason or description, in characters (008-P16).
const maxCommentLength = 2000

// Service holds the rules of approval workflows, approval rounds and notifications.
type Service struct {
	tx         store.Transactor
	repo       Repository
	people     people.Repository
	activities Activities
	ids        *ids.Generator
	history    *history.Service
	now        clock.Clock
	// displayRates reads the reader's display currency (UseDisplayRates, D-213); nil shows USD.
	displayRates DisplayRates
	// access answers feature 014's questions (UseAccess); choices lists what rule conditions may name
	// (UseRuleChoices). NewService starts with everyone allowed and no choices (017 plan, Ports).
	access  Access
	choices RuleChoices
	// approvers says who may approve (UseApprovers, feature 014); nil: everyone.
	approvers Approvers
}

// NewService returns the approvals service. It keeps its records in repo, reads people from peopleRepo
// and activities through activities, gives out readable IDs with generator, logs through hist, works in
// transactions of tx and reads the time from now.
func NewService(tx store.Transactor, repo Repository, peopleRepo people.Repository, activities Activities,
	generator *ids.Generator, hist *history.Service, now clock.Clock) *Service {
	return &Service{tx: tx, repo: repo, people: peopleRepo, activities: activities, ids: generator, history: hist, now: now,
		access: openAccess{}, choices: noChoices{}}
}

// UseAccess sets where approvals reads people's permissions, the access rule and the roles (feature 014).
func (s *Service) UseAccess(access Access) {
	s.access = access
}

// UseRuleChoices sets where approvals reads the activity types and business units of rule conditions.
func (s *Service) UseRuleChoices(choices RuleChoices) {
	s.choices = choices
}

// openAccess is the Access of a service without feature 014: everyone holds every permission, the access
// rule is on and there are no roles (data-model §6: "everything allowed, rule on").
type openAccess struct{}

func (openAccess) CanApprove(_ context.Context, personIDs []string) (map[string]bool, error) {
	can := make(map[string]bool, len(personIDs))
	for _, id := range personIDs {
		can[id] = true
	}
	return can, nil
}

func (openAccess) Check(context.Context, string, Permission, *PlanFacts) (Grant, error) {
	return Grant{Allowed: true}, nil
}

func (openAccess) NoSelfApproval(context.Context) (bool, error) { return true, nil }

func (openAccess) Roles(context.Context) ([]RoleInfo, error) { return []RoleInfo{}, nil }

func (openAccess) RoleHolders(context.Context, string, PlanFacts) ([]string, error) {
	return []string{}, nil
}

// noChoices is the RuleChoices of a service without Settings: nothing to choose.
type noChoices struct{}

func (noChoices) ActivityTypes(context.Context) ([]Choice, error) { return []Choice{}, nil }

func (noChoices) BusinessUnits(context.Context) ([]Choice, error) { return []Choice{}, nil }

// activity returns the stored activity id, or not_found with the plan's message.
func (s *Service) activity(ctx context.Context, id string) (ActivityFacts, error) {
	facts, found, err := s.activities.Activity(ctx, id)
	if err != nil {
		return ActivityFacts{}, err
	}
	if !found {
		return ActivityFacts{}, errs.NotFound(activityGoneMessage)
	}
	return facts, nil
}

// directory is every person by id, read once per operation: names, initials and the live active flag.
type directory struct {
	byID      map[string]people.Person
	approving map[string]bool // who may approve (feature 014, FR-028); nil: access control is not wired
}

// directory reads every person, and who of them may approve.
func (s *Service) directory(ctx context.Context) (directory, error) {
	list, err := s.people.List(ctx)
	if err != nil {
		return directory{}, err
	}
	d := directory{byID: map[string]people.Person{}}
	ids := make([]string, 0, len(list))
	for _, p := range list {
		d.byID[p.ID] = p
		ids = append(ids, p.ID)
	}
	if s.approvers != nil {
		d.approving, err = s.approvers.CanApprove(ctx, ids)
	}
	return d, err
}

// person returns the person id as the API shows them, with their name now. An id nobody has is shown as
// itself and inactive.
func (d directory) person(id string) PersonView {
	p, found := d.byID[id]
	if !found {
		return PersonView{ID: id, Name: id, Initials: people.Person{Name: id}.Initials(), Active: false}
	}
	return PersonView{ID: p.ID, Name: p.Name, Initials: p.Initials(), Active: p.Active, CanApprove: d.canApprove(p.ID)}
}

// copied returns a person copied into a round or history entry (with their name at the time) as the API
// shows them: that name, with the live active flag (FR-024 "(inactive)").
func (d directory) copied(a actor.Actor) PersonView {
	return PersonView{ID: a.ID, Name: a.Name, Initials: people.Person{Name: a.Name}.Initials(), Active: d.byID[a.ID].Active,
		CanApprove: d.canApprove(a.ID)}
}

// canApprove says whether the person id may approve (feature 014, FR-028); nil while access control is not
// wired.
func (d directory) canApprove(id string) *bool {
	if d.approving == nil {
		return nil
	}
	can := d.approving[id]
	return &can
}

// copiedAll returns copied for each of actors; never nil.
func (d directory) copiedAll(actors []actor.Actor) []PersonView {
	views := make([]PersonView, 0, len(actors))
	for _, a := range actors {
		views = append(views, d.copied(a))
	}
	return views
}

// actors returns the people ids as actors with their names now, in order; never nil.
func (d directory) actors(personIDs []string) []actor.Actor {
	list := make([]actor.Actor, 0, len(personIDs))
	for _, id := range personIDs {
		p := d.person(id)
		list = append(list, actor.Actor{ID: p.ID, Name: p.Name})
	}
	return list
}

// levelLabel is the label of the level at position n: its name, or "Level <n>" when it has none.
func levelLabel(name string, n int) string {
	if name == "" {
		return fmt.Sprintf("Level %d", n)
	}
	return name
}

// names joins the names of actors with ", ".
func names(actors []actor.Actor) string {
	list := make([]string, 0, len(actors))
	for _, a := range actors {
		list = append(list, a.Name)
	}
	return strings.Join(list, ", ")
}

// includes reports whether personID is one of actors.
func includes(actors []actor.Actor, personID string) bool {
	return slices.ContainsFunc(actors, func(a actor.Actor) bool { return a.ID == personID })
}

// tooLong reports whether text has more than limit characters (characters, not bytes).
func tooLong(text string, limit int) bool {
	return utf8.RuneCountInString(text) > limit
}

// decidedConflict refuses a decision on a level or round that has moved on (FR-018, D-155):
// version_conflict whose message says what happened and whose conflict says who and when.
func decidedConflict(message, by string, at time.Time) *errs.Error {
	return &errs.Error{
		Status: http.StatusConflict, Code: errs.CodeVersionConflict, Message: message, Fields: []errs.FieldError{},
		Conflict: &errs.Conflict{ChangedBy: by, ChangedAt: at},
	}
}

// changeText is the sentence of one changed value of a restart (data model §3).
func changeText(c ChangeText) string {
	const notFilledIn = "Not filled in"
	switch {
	case c.OldText == "" && c.NewText == "":
		return c.Label // a ready-made sentence, such as a line-item change of 007
	case c.OldText == notFilledIn:
		return fmt.Sprintf("%s added: %s", c.Label, c.NewText)
	case c.NewText == notFilledIn:
		return fmt.Sprintf("%s removed (was %s)", c.Label, c.OldText)
	default:
		return fmt.Sprintf("%s changed from %s to %s", c.Label, c.OldText, c.NewText)
	}
}

// activityText names an activity in a notification: "'Launch film' (A-1042)".
func activityText(facts ActivityFacts) string {
	return fmt.Sprintf("'%s' (%s)", facts.Name, facts.ID)
}

// notifier writes the notifications of one approval action by a about an activity (data model §4): one
// per person, whatever the number of sends, and never to a about their own action (FR-036).
type notifier struct {
	service *Service
	actor   actor.Actor
	facts   ActivityFacts
	at      time.Time
	told    map[string]bool
}

// notifier starts the notifications of a's action on facts at the time at.
func (s *Service) notifier(a actor.Actor, facts ActivityFacts, at time.Time) *notifier {
	return &notifier{service: s, actor: a, facts: facts, at: at, told: map[string]bool{a.ID: true}}
}

// send writes one notification of kind for each recipient not told yet, inside the caller's Write. Empty
// ids (an activity without owner) are skipped.
func (n *notifier) send(ctx context.Context, recipientIDs []string, kind NotificationKind, title, text string) error {
	for _, id := range recipientIDs {
		if id == "" || n.told[id] {
			continue
		}
		n.told[id] = true
		notificationID, err := n.service.ids.Next(ctx, ids.PrefixNotification)
		if err != nil {
			return err
		}
		err = n.service.repo.PutNotification(ctx, Notification{
			ID: notificationID, RecipientID: id, Kind: kind, Title: title, Text: text,
			ActivityID: n.facts.ID, ActivityName: n.facts.Name, Actor: n.actor, At: n.at,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// quoted is the start of a comment as a notification quotes it: the first 80 characters, then "…" when
// there is more (008-P16).
func quoted(comment string) string {
	runes := []rune(comment)
	if len(runes) <= 80 {
		return comment
	}
	return string(runes[:80]) + "…"
}

// idsOf returns the ids of actors, in order.
func idsOf(actors []actor.Actor) []string {
	list := make([]string, 0, len(actors))
	for _, a := range actors {
		list = append(list, a.ID)
	}
	return list
}

// statusLog is one approval action as the activity log records it on the activity (FR-026).
type statusLog struct {
	verb             string // "submitted", "approved level 1 of", …
	oldStatus, newer string // the status change; both "" when the status stays
	comment, details string
}

// logOnActivity writes one statusChanged entry about facts, inside the caller's Write.
func (s *Service) logOnActivity(ctx context.Context, a actor.Actor, facts ActivityFacts, entry statusLog) error {
	changes := []history.Change{}
	if entry.oldStatus != entry.newer {
		oldValue, err := json.Marshal(entry.oldStatus)
		if err != nil {
			return err
		}
		newValue, err := json.Marshal(entry.newer)
		if err != nil {
			return err
		}
		changes = append(changes, history.Change{Field: "status", Old: oldValue, New: newValue})
	}
	_, err := s.history.Log(ctx, a, history.LogInput{
		Kind:       activityKind,
		RecordID:   facts.ID,
		RecordName: facts.Name,
		Action:     history.ActionStatusChanged,
		Verb:       entry.verb,
		Changes:    changes,
		Comment:    entry.comment,
		Details:    entry.details,
	})
	return err
}
