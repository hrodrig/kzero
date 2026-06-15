package configs

import (
	"strings"
	"testing"
)

func TestSampleYAML(t *testing.T) {
	s := SampleYAML()
	if !strings.Contains(s, `schema_version: "1.0"`) {
		t.Fatalf("missing schema_version in sample")
	}
	if !strings.Contains(s, "helm:") {
		t.Fatal("missing helm block")
	}
}
