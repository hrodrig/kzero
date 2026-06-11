// Package redact masks secrets in log lines, notify payloads, and subprocess output.
package redact

import (
	"regexp"
	"strings"
)

var (
	bearerRE    = regexp.MustCompile(`(?i)(Bearer\s+)(\S+)`)
	secretEnvRE = regexp.MustCompile(`(?i)([A-Za-z0-9_]*(?:TOKEN|SECRET|KEY|PASSWORD|WEBHOOK_URL)[A-Za-z0-9_]*=)(\S+)`)
	urlRE       = regexp.MustCompile(`https?://[^\s"'<>]+`)
)

// String applies common secret scrubbing patterns to free-form text.
func String(s string) string {
	if s == "" {
		return s
	}
	s = bearerRE.ReplaceAllString(s, `${1}***`)
	s = secretEnvRE.ReplaceAllString(s, `${1}***`)
	s = urlRE.ReplaceAllStringFunc(s, URL)
	return s
}

// URL masks credentials and truncates long webhook-style URLs for safe logs.
func URL(u string) string {
	if i := strings.Index(u, "://"); i >= 0 {
		rest := u[i+3:]
		if j := strings.Index(rest, "@"); j >= 0 {
			return u[:i+3] + "***@" + rest[j+1:]
		}
	}
	if len(u) > 24 {
		return u[:12] + "…" + u[len(u)-4:]
	}
	if strings.Contains(u, "://") {
		return "***"
	}
	return u
}
