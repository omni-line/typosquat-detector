package discover_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/discover"
	"github.com/omni-line/typosquat-detector/internal/match"
)

func write(t *testing.T, root, rel string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func classify(rel string) string {
	if strings.HasSuffix(rel, "package.json") {
		return "npm"
	}
	return ""
}

func rels(ms []discover.Manifest) []string {
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.Rel
	}
	sort.Strings(out)
	return out
}

func TestWalk(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"package.json",
		"apps/web/package.json",
		"node_modules/x/package.json",
		"packages/legacy/package.json",
		"README.md",
	} {
		write(t, root, rel)
	}
	if err := os.Symlink(filepath.Join(root, "package.json"), filepath.Join(root, "apps", "package.json")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	var warned []string
	ms, err := discover.Walk(context.Background(), root, discover.Options{
		Classify: classify,
		Exclude:  match.New("packages/legacy"),
		Warn:     func(rel string, _ error) { warned = append(warned, rel) },
	})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(rels(ms), ",")
	if got != "apps/web/package.json,package.json" {
		t.Fatalf("got %s", got)
	}
	if len(warned) != 1 || warned[0] != "apps/package.json" {
		t.Fatalf("warned=%v want symlink warning", warned)
	}
}

func TestWalkSingleFile(t *testing.T) {
	root := t.TempDir()
	write(t, root, "package.json")
	ms, err := discover.Walk(context.Background(), filepath.Join(root, "package.json"), discover.Options{Classify: classify})
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].Rel != "package.json" {
		t.Fatalf("got %+v", ms)
	}
}

func TestWalkCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := discover.Walk(ctx, t.TempDir(), discover.Options{Classify: classify})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
