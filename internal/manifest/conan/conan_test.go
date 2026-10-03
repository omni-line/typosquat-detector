package conan_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/conan"
)

func TestParse(t *testing.T) {
	deps, err := conan.Parse([]byte("[requires]\nopenssl/3.2.0\nzlib/1.3.1@user/channel\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 || deps[0].Name != "openssl" {
		t.Fatalf("%+v", deps)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("[requires]\na/1.0\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = conan.Parse(data)
	})
}
