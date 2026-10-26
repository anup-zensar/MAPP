package apitest

import (
	"context"
	"io/fs"
	"os"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"mapp/internal/platform/store"
	"mapp/internal/stores"
	"mapp/seed"
)

// demoData holds the data folder of the demo data, loaded once per test binary: loading the seed costs a
// fifth of a second, copying its files a few milliseconds.
type demoData struct {
	once    sync.Once
	files   fstest.MapFS
	failure error
}

var withHistory, withoutHistory demoData //nolint:gochecknoglobals // a read-only fixture of the test binary, as in _test.go files

// copyDemoData fills an empty dir with the demo data; a folder that already holds data is left as it is.
func copyDemoData(tb testing.TB, dir string, noHistory bool) {
	tb.Helper()
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return
	}
	data, options := &withHistory, []seed.Option(nil)
	if noHistory {
		data, options = &withoutHistory, []seed.Option{seed.WithoutHistory()}
	}
	data.once.Do(func() { data.files, data.failure = loadDemoData(options) })
	if data.failure != nil {
		tb.Fatalf("apitest: loading the demo data: %v", data.failure)
	}
	if err := os.CopyFS(dir, data.files); err != nil {
		tb.Fatalf("apitest: copying the demo data: %v", err)
	}
}

func loadDemoData(options []seed.Option) (fstest.MapFS, error) {
	dir, err := os.MkdirTemp("", "apitest-demo-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	db, err := store.Open(dir)
	if err != nil {
		return nil, err
	}
	at := func() time.Time { return startTime }
	if err = seed.Load(context.Background(), stores.JSON(db), at, options...); err != nil {
		return nil, err
	}
	return readFolder(dir)
}

func readFolder(dir string) (fstest.MapFS, error) {
	files := fstest.MapFS{}
	folder := os.DirFS(dir)
	err := fs.WalkDir(folder, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		content, err := fs.ReadFile(folder, path)
		files[path] = &fstest.MapFile{Data: content, Mode: 0o644}
		return err
	})
	return files, err
}
