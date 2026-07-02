package cluster

import (
	"regexp"
	"strings"
)

var filenameSanitizeRE = regexp.MustCompile(`[^a-z0-9-]+`)

// SanitizeForFilename returns a lowercase slug safe for log filenames and paths.
// Empty or all-punctuation input becomes "unknown".
func SanitizeForFilename(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return "unknown"
	}
	s = filenameSanitizeRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	if s == "" {
		return "unknown"
	}
	return s
}
