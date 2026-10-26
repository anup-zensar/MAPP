package database

import (
	"time"

	"mapp/internal/platform/date"
)

// DateColumn is an optional calendar date as a date column: NULL when it is nil.
func DateColumn(d *date.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time()
	return &t
}

func DateOf(column *time.Time) *date.Date {
	if column == nil {
		return nil
	}
	d := date.Today(*column)
	return &d
}
