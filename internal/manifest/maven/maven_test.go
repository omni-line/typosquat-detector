package maven_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/maven"
)

func TestParse(t *testing.T) {
	deps, err := maven.Parse([]byte(`<project><dependencies>
<dependency><groupId>junit</groupId><artifactId>junit</artifactId><version>4.13.2</version></dependency>
</dependencies></project>`))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 1 || deps[0].Name != "junit:junit" {
		t.Fatalf("%+v", deps)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`<project><dependencies><dependency><groupId>a</groupId><artifactId>b</artifactId></dependency></dependencies></project>`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = maven.Parse(data)
	})
}
