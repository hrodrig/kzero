package configs

import _ "embed"

//go:embed kzero.sample.yml
var sampleYAML []byte

// SampleYAML returns the annotated sample configuration (same as configs/kzero.sample.yml).
func SampleYAML() string {
	return string(sampleYAML)
}
