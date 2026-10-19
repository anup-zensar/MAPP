// Package record holds the bookkeeping every business record carries: its version and who created and
// last changed it, and when (data model "Bookkeeping", research R5).
//
// A record embeds Meta, so its fields appear at the top level of the record's JSON:
//
//	type Folder struct {
//		ID   string `json:"id"`
//		Name string `json:"name"`
//		record.Meta
//	}
//
// The version protects against lost updates (FR-029): every update request sends the version it started
// from, and the service refuses it with CheckVersion when someone else has saved in between.
package record

import (
	"time"

	"northstar/internal/platform/actor"
	"northstar/internal/platform/errs"
)

// Meta is the bookkeeping of a business record.
type Meta struct {
	Version   int         `json:"version"` // 1 on create, +1 on every saved change
	CreatedAt time.Time   `json:"createdAt"`
	CreatedBy actor.Actor `json:"createdBy"`
	UpdatedAt time.Time   `json:"updatedAt"` // equals CreatedAt until the first change
	UpdatedBy actor.Actor `json:"updatedBy"` // equals CreatedBy until the first change
}

// NewMeta returns the bookkeeping of a record that by creates now.
func NewMeta(by actor.Actor, now time.Time) Meta {
	return Meta{Version: 1, CreatedAt: now, CreatedBy: by, UpdatedAt: now, UpdatedBy: by}
}

// Touch records a saved change: by changed the record now, and the version goes up by one.
func (m *Meta) Touch(by actor.Actor, now time.Time) {
	m.Version++
	m.UpdatedAt = now
	m.UpdatedBy = by
}

// CheckVersion refuses a save based on an out-of-date copy of the record: when expected (the version the
// person started editing from) is not the stored version, it returns a version_conflict error that names
// who changed the record last and when. noun names the kind of record in the message, as in "Not saved —
// Daniel Okoro changed this sample folder".
func (m Meta) CheckVersion(expected int, noun string) error {
	if expected == m.Version {
		return nil
	}
	return errs.VersionConflict(noun, m.UpdatedBy.Name, m.UpdatedAt)
}
