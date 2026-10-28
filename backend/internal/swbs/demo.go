package swbs

import (
	"context"
	"time"

	"mapp/internal/platform/actor"
	"mapp/internal/platform/ids"
)

type DemoLoad struct {
	Requests []Request
	Items    Receiver
	At       time.Time
}

// LoadDemo makes the demo data's run inside the caller's Write: requests go through the simulated GSAP and the
// same send as any run, as MAPP at demo.At, source demoData, with no activity-log entry (D-188).
func LoadDemo(ctx context.Context, repo Store, generator *ids.Generator, demo DemoLoad) (Run, error) {
	d := delivery{
		repo: repo, ids: generator, gsap: NewSimulated(repo), items: demo.Items, actor: actor.System, source: SourceDemo,
		at: demo.At,
	}
	return d.send(ctx, demo.Requests)
}
