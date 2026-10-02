package manifest_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

func TestReadFileStripsBOM(t *testing.T) {
	p := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(p, []byte("\xEF\xBB\xBF{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := manifest.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Fatalf("got %q", data)
	}
}

func TestReadFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "secret")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "package.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if _, err := manifest.ReadFile(link); !errors.Is(err, manifest.ErrNotRegular) {
		t.Fatalf("err=%v want ErrNotRegular", err)
	}
}

func TestReadFileTooLarge(t *testing.T) {
	p := filepath.Join(t.TempDir(), "requirements.txt")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(manifest.MaxFileSize + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := manifest.ReadFile(p); !errors.Is(err, manifest.ErrTooLarge) {
		t.Fatalf("err=%v want ErrTooLarge", err)
	}
}

func TestLineAt(t *testing.T) {
	data := []byte("a\nb\nc")
	for off, want := range map[int]int{-1: 0, 0: 1, 2: 2, 4: 3, 99: 3} {
		if got := manifest.LineAt(data, off); got != want {
			t.Errorf("LineAt(%d)=%d want %d", off, got, want)
		}
	}
}
