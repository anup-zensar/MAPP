// Package ids gives records their permanent readable IDs: "P-1001", "P-1002", … (FR-027, research R4).
//
// Each prefix has its own counter, which only ever goes up, so an ID is never given out twice, not even
// after its record was deleted. The counter is read and raised inside the caller's Write, so a new ID and
// its new record are saved together. Every record with a readable ID, seed records included, gets it
// from a Generator.
//
// Prefixes: P, C, I, A, L for planning records; U people; LOG activity-log entries; BIN recycle-bin
// entries; SF and SN the developer sample records.
package ids

import (
	"context"
	"fmt"
)

// firstNumber is the number of the first ID of every prefix: "P-1001".
const firstNumber = 1001

// Counters is the repository of the last number given per prefix. The JSON store keeps it in the table
// "counters"; PostgreSQL will have its own implementation.
type Counters interface {
	// Increment raises the counter of prefix by one and returns the new number, 1001 the first time.
	// It runs inside the caller's Write.
	Increment(ctx context.Context, prefix string) (int, error)
}

// Generator gives out readable IDs.
type Generator struct {
	counters Counters
}

// NewGenerator returns a Generator that keeps its counters in c.
func NewGenerator(c Counters) *Generator {
	return &Generator{counters: c}
}

// Next returns the next ID of prefix, such as "P-1001". Call it inside a Write.
func (g *Generator) Next(ctx context.Context, prefix string) (string, error) {
	number, err := g.NextNumber(ctx, prefix)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", prefix, number), nil
}

// NextNumber returns the number of the next ID of prefix, without the prefix. The activity log uses it to
// order its entries (LOG-1001 has the sequence number 1001). Call it inside a Write.
func (g *Generator) NextNumber(ctx context.Context, prefix string) (int, error) {
	return g.counters.Increment(ctx, prefix)
}
