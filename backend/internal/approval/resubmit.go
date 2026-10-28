package approval

import (
	"context"
	"fmt"
)

const (
	kindSubmit   = "submit"
	kindResubmit = "resubmit"
)

// supersede ends the open round a re-submission replaces, at the level it waited at.
func (s *Service) supersede(ctx context.Context, sb *submitting) error {
	if sb.open == nil {
		return nil
	}
	open := *sb.open
	open.Outcome, open.EndedAt = OutcomeSuperseded, &sb.now
	open.Touch(sb.a, sb.now)
	if err := s.repo.PutRound(ctx, open); err != nil {
		return err
	}
	entry := Action{Action: ActionSuperseded, Level: open.CurrentLevel, Actor: sb.a, At: sb.now}
	if err := s.addAction(ctx, open, entry); err != nil {
		return err
	}
	if !sb.sub.Kind.isPurchaseRequest() {
		return nil
	}
	moved, err := s.moved(ctx, sb.a, sb.sub.as(open.kind()), OutcomeSuperseded)
	sb.sub = moved.as(sb.sub.Kind)
	return err
}

// submissionKind says whether sub can be submitted now (kindSubmit), re-submitted (kindResubmit) or neither ("")
// (data-model §9); open is its open round, nil when it has none.
func submissionKind(sub subject, open *Round) string {
	openOfKind := open != nil && open.kind() == sub.Kind
	switch sub.Kind {
	case KindPurchaseRequest:
		return purchaseSubmission(sub.Status, openOfKind)
	case KindModifiedPurchaseRequest:
		return modificationSubmission(sub.Status, open != nil, openOfKind)
	case KindActivity:
	}
	switch sub.Status {
	case StatusCreated:
		return kindSubmit
	case StatusDeclined, StatusSubmitted:
		return kindResubmit
	}
	return ""
}

func purchaseSubmission(status string, openOfKind bool) string {
	switch {
	case status == StatusCreated:
		return kindSubmit
	case status == StatusDeclined || openOfKind:
		return kindResubmit
	}
	return ""
}

func modificationSubmission(status string, open, openOfKind bool) string {
	switch {
	case openOfKind:
		return kindResubmit
	case !open && status == StatusApproved:
		return kindSubmit
	}
	return ""
}

func notSubmittableMessage(kind SubjectKind) string {
	switch kind {
	case KindPurchaseRequest:
		return purchaseNotSubmittableMessage
	case KindModifiedPurchaseRequest:
		return modificationNotSubmittableMessage
	case KindActivity:
	}
	return onlyCreatedOrDeclinedMessage
}

// submitWords are the history entry, the log verb and the notification text of a submission.
type submitWords struct {
	action ActionKind
	verb   string
	text   string
}

func submitWordsOf(kind SubjectKind, submission string) submitWords {
	words := submitWords{action: ActionSubmitted, verb: "submitted", text: submittedText}
	if submission == kindResubmit {
		words = submitWords{action: ActionResubmitted, verb: "re-submitted", text: resubmittedText}
	}
	if kind == KindModifiedPurchaseRequest {
		words.verb += " changes to"
		words.text = fmt.Sprintf(changesText, words.verb)
	}
	return words
}

func openRoundPointer(rounds []Round) *Round {
	if open, found := openRound(rounds); found {
		return &open
	}
	return nil
}

// replaced is the round a re-submission of sub would end, for the confirmation; nil when none.
func replaced(sub subject, open *Round) *ReplacedRound {
	if open == nil || submissionKind(sub, open) != kindResubmit {
		return nil
	}
	level := open.Levels[open.CurrentLevel-1]
	return &ReplacedRound{RoundNumber: open.Number, Level: level.Number, LevelLabel: levelLabel(level.Name, level.Number)}
}
