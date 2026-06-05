package log

import (
	"fmt"
	"strings"
)

// Format selects human legacy lines or one-JSON-object-per-line output.
type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

// ParseFormat normalizes CLI --log-format values.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "text":
		return FormatText, nil
	case "json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("log-format: unknown value %q (want text or json)", s)
	}
}
