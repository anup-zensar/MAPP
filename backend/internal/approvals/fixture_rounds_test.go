package approvals

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
)

// approve approves the current level of the activity's open round as a, and stops the test when that fails.
func (f *fixture) approve(t *testing.T, a actor.Actor, id, comment string) ActivityApproval {
	t.Helper()
	round := f.openRound(t, id)
	approval, err := f.service.Approve(t.Context(), a, id, DecisionInput{RoundID: round.ID, Level: version(round.CurrentLevel), Comment: comment})
	if err != nil {
		t.Fatalf("Approve(%s): %v", id, err)
	}
	return approval
}

// decline declines the activity at the current level of its open round as a, and stops the test when that
// fails.
func (f *fixture) decline(t *testing.T, a actor.Actor, id, comment string) ActivityApproval {
	t.Helper()
	round := f.openRound(t, id)
	approval, err := f.service.Decline(t.Context(), a, id, DecisionInput{RoundID: round.ID, Level: version(round.CurrentLevel), Comment: comment})
	if err != nil {
		t.Fatalf("Decline(%s): %v", id, err)
	}
	return approval
}

// openRound returns the open round of the activity id, and stops the test when there is none.
func (f *fixture) openRound(t *testing.T, id string) Round {
	t.Helper()
	round, found := openRound(f.rounds(t, id))
	if !found {
		t.Fatalf("%s has no open round", id)
	}
	return round
}

// assertNotified checks the notifications of person p, newest first, each written as
// "<kind>: <title> — <text>".
func assertNotified(t *testing.T, f *fixture, p actor.Actor, want ...string) {
	t.Helper()
	var got []string
	for _, n := range f.notifications(t, p.ID) {
		got = append(got, fmt.Sprintf("%s: %s — %s", n.Kind, n.Title, n.Text))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s's notifications =\n%s\nwant\n%s", p.Name, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// assertNothingHappened checks that the activity id still has status, no round, no history entry, no log
// entry and that nobody was notified.
func assertNothingHappened(t *testing.T, f *fixture, id, status string) {
	t.Helper()
	if got := f.activities.facts[id].Status; got != status {
		t.Errorf("%s: status = %s, want %s", id, got, status)
	}
	if rounds, actions, entries := f.rounds(t, id), f.actions(t, id), f.log(t, id); len(rounds)+len(actions)+len(entries) > 0 {
		t.Errorf("%s: rounds %+v, history %+v, log %+v; want none", id, rounds, actions, entries)
	}
	for _, p := range []actor.Actor{alex, priya, daniel, mei, carlos, sara} {
		if got := f.notifications(t, p.ID); len(got) > 0 {
			t.Errorf("%s was notified: %+v", p.Name, got)
		}
	}
}

// assertLog checks the activity log of the record id, newest first, each entry written as
// "<actor> <verb> <type> '<name>' [<old> → <new>] "<comment>" (<details>)".
func assertLog(t *testing.T, f *fixture, id string, want []string) {
	t.Helper()
	var got []string
	for _, e := range f.log(t, id) {
		got = append(got, logLine(e))
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("log of %s =\n%s\nwant\n%s", id, strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// logLine writes one activity-log entry as assertLog compares it.
func logLine(e history.Entry) string {
	line := e.Actor.Name + " " + e.Verb + " " + e.RecordTypeLabel + " '" + e.RecordName + "'"
	for _, c := range e.Changes {
		var oldValue, newValue any
		_ = json.Unmarshal(c.Old, &oldValue)
		_ = json.Unmarshal(c.New, &newValue)
		line += " [" + text(oldValue) + " → " + text(newValue) + "]"
	}
	if e.Comment != "" {
		line += ` "` + e.Comment + `"`
	}
	if e.Details != "" {
		line += " (" + e.Details + ")"
	}
	return line
}

// text writes a decoded JSON value for logLine.
func text(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	encoded, _ := json.Marshal(v)
	return string(encoded)
}
