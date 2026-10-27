package approval

import (
	"context"
	"encoding/json"
	"fmt"
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
)

const activityKind = "activity"

const maxCommentLength = 2000

type Service struct {
	tx           Transactor
	repo         Store
	people       People
	activities   Activities
	ids          *ids.Generator
	history      *history.Service
	now          clock.Clock
	displayRates DisplayRates // nil shows USD
	access       Access
	choices      RuleChoices
	approvers    Approvers // nil: everyone may approve
}

// NewService starts with everyone allowed and nothing for rule conditions to name, until UseAccess and UseRuleChoices.
func NewService(tools history.Tools, repo Store, peopleRepo People, activities Activities) *Service {
	return &Service{tx: tools.Tx, repo: repo, people: peopleRepo, activities: activities, ids: tools.IDs,
		history: tools.Log, now: tools.Now, access: openAccess{}, choices: noChoices{}}
}

func (s *Service) UseAccess(access Access) {
	s.access = access
}

func (s *Service) UseRuleChoices(choices RuleChoices) {
	s.choices = choices
}

// openAccess is a service without feature 014: everyone holds every permission, the access rule is on, no roles.
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

type noChoices struct{}

func (noChoices) ActivityTypes(context.Context) ([]Choice, error) { return []Choice{}, nil }

func (noChoices) BusinessUnits(context.Context) ([]Choice, error) { return []Choice{}, nil }

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

// directory is every person by id, read once per operation.
type directory struct {
	byID      map[string]people.Person
	approving map[string]bool // nil: access control is not wired
}

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

func (d directory) person(id string) PersonView {
	p, found := d.byID[id]
	if !found {
		return PersonView{ID: id, Name: id, Initials: people.Person{Name: id}.Initials(), Active: false}
	}
	return PersonView{ID: p.ID, Name: p.Name, Initials: p.Initials(), Active: p.Active, CanApprove: d.canApprove(p.ID)}
}

// copied shows a person with the name copied at the time, and the live active flag.
func (d directory) copied(a actor.Actor) PersonView {
	return PersonView{ID: a.ID, Name: a.Name, Initials: people.Person{Name: a.Name}.Initials(), Active: d.byID[a.ID].Active,
		CanApprove: d.canApprove(a.ID)}
}

func (d directory) canApprove(id string) *bool {
	if d.approving == nil {
		return nil
	}
	can := d.approving[id]
	return &can
}

func (d directory) copiedAll(actors []actor.Actor) []PersonView {
	views := make([]PersonView, 0, len(actors))
	for _, a := range actors {
		views = append(views, d.copied(a))
	}
	return views
}

func (d directory) actors(personIDs []string) []actor.Actor {
	list := make([]actor.Actor, 0, len(personIDs))
	for _, id := range personIDs {
		p := d.person(id)
		list = append(list, actor.Actor{ID: p.ID, Name: p.Name})
	}
	return list
}

func levelLabel(name string, n int) string {
	if name == "" {
		return fmt.Sprintf("Level %d", n)
	}
	return name
}

func names(actors []actor.Actor) string {
	list := make([]string, 0, len(actors))
	for _, a := range actors {
		list = append(list, a.Name)
	}
	return strings.Join(list, ", ")
}

func includes(actors []actor.Actor, personID string) bool {
	return slices.ContainsFunc(actors, func(a actor.Actor) bool { return a.ID == personID })
}

func tooLong(text string, limit int) bool {
	return utf8.RuneCountInString(text) > limit
}

// decidedConflict refuses a decision on a level or round that has moved on, saying what happened, who and when.
func decidedConflict(message, by string, at time.Time) *errs.Error {
	conflict := errs.VersionConflict("", by, at)
	conflict.Message = message
	return conflict
}

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

func activityText(facts ActivityFacts) string {
	return fmt.Sprintf("'%s' (%s)", facts.Name, facts.ID)
}

// notifier tells each person once per action, whatever the number of sends, and never the actor.
type notifier struct {
	service *Service
	actor   actor.Actor
	facts   ActivityFacts
	at      time.Time
	told    map[string]bool
}

func (s *Service) notifier(a actor.Actor, facts ActivityFacts, at time.Time) *notifier {
	return &notifier{service: s, actor: a, facts: facts, at: at, told: map[string]bool{a.ID: true}}
}

// send skips an empty id: an activity without owner.
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

func quoted(comment string) string {
	runes := []rune(comment)
	if len(runes) <= 80 {
		return comment
	}
	return string(runes[:80]) + "…"
}

func idsOf(actors []actor.Actor) []string {
	list := make([]string, 0, len(actors))
	for _, a := range actors {
		list = append(list, a.ID)
	}
	return list
}

type statusLog struct {
	verb             string // "submitted", "approved level 1 of", …
	oldStatus, newer string // both "" when the status stays
	comment, details string
}

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
