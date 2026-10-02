//go:build unix

package manifest_test

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

func TestReadFileRejectsFIFOWithoutBlocking(t *testing.T) {
	p := filepath.Join(t.TempDir(), "package.json")
	if err := syscall.Mkfifo(p, 0o600); err != nil {
		t.Skipf("mkfifo: %v", err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := manifest.ReadFile(p)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, manifest.ErrNotRegular) {
			t.Fatalf("err=%v want ErrNotRegular", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("ReadFile blocked on a FIFO")
	}
}
