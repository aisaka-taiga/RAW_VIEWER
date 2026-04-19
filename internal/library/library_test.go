package library

import "testing"

func TestIsSupportedIncludesHeifFamily(t *testing.T) {
	for _, path := range []string{"sample.heic", "sample.heif", "sample.hif"} {
		if !IsSupported(path) {
			t.Fatalf("expected %s to be supported", path)
		}
	}

	if IsSupported("sample.txt") {
		t.Fatal("txt should not be supported")
	}
}
