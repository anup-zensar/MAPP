package sample

import (
	"context"
	"strings"
	"unicode/utf8"

	"mapp/internal/history"
	"mapp/internal/platform/actor"
	"mapp/internal/platform/errs"
	"mapp/internal/platform/record"
)

// CreateNote adds a note titled input.Title, without its surrounding spaces, to the sample folder folderID,
// and writes a "created" log entry with it, in one Write.
func (s *Service) CreateNote(ctx context.Context, a actor.Actor, folderID string, input NoteInput) (Note, error) {
	title := strings.TrimSpace(input.Title)
	var problems errs.Problems
	checkTitle(&problems, title)
	if err := problems.Err(); err != nil {
		return Note{}, err
	}

	var note Note
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetFolder(ctx, folderID)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(folderGoneMessage)
		}
		id, err := s.ids.Next(ctx, notePrefix)
		if err != nil {
			return err
		}
		note = Note{ID: id, FolderID: folderID, Title: title, Meta: record.NewMeta(a, s.now())}
		if err = s.repo.PutNote(ctx, note); err != nil {
			return err
		}
		_, err = s.history.Log(ctx, a, history.LogInput{
			Kind:       noteKind,
			RecordID:   note.ID,
			RecordName: note.Title,
			Action:     history.ActionCreated,
		})
		return err
	})
	if err != nil {
		return Note{}, err
	}
	return note, nil
}

// DeleteNote moves the sample note id to the recycle bin (FR-024), and returns the recycle-bin entry so
// that the person can undo the delete.
func (s *Service) DeleteNote(ctx context.Context, a actor.Actor, id string) (history.BinEntry, error) {
	var entry history.BinEntry
	err := s.tx.Write(ctx, func(ctx context.Context) error {
		_, found, err := s.repo.GetNote(ctx, id)
		if err != nil {
			return err
		}
		if !found {
			return errs.NotFound(noteGoneMessage)
		}
		entry, err = s.history.MoveToBin(ctx, a, noteKind, id)
		return err
	})
	if err != nil {
		return history.BinEntry{}, err
	}
	return entry, nil
}

// checkTitle adds the problem with a note title, already trimmed, to problems: the title is required and
// has at most 120 characters.
func checkTitle(problems *errs.Problems, title string) {
	if title == "" {
		problems.Add("title", enterTitleMessage)
	} else if utf8.RuneCountInString(title) > maxTitleLength {
		problems.Add("title", titleTooLongMessage)
	}
}
