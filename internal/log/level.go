package log

import (
	"fmt"
	"strings"
)

// Level is a syslog-style severity for text/JSON log lines.
type Level int

const (
	LevelDebug Level = 1
	LevelInfo  Level = 2
	LevelWarn  Level = 3
	LevelError Level = 4
)

var minLevel = LevelInfo

// SetMinLevel filters text and JSON events below the given level (default info).
func SetMinLevel(l Level) {
	minLevel = l
}

// MinLevel returns the active minimum level.
func MinLevel() Level {
	return minLevel
}

// Tag returns the three-letter level label (DBG, INF, WRN, ERR).
func (l Level) Tag() string {
	switch l {
	case LevelDebug:
		return "DBG"
	case LevelInfo:
		return "INF"
	case LevelWarn:
		return "WRN"
	case LevelError:
		return "ERR"
	default:
		return "INF"
	}
}

// Enabled reports whether l should be emitted when minLevel is active.
func (l Level) Enabled() bool {
	return l >= minLevel
}

// ParseLevel normalizes CLI and config values (debug, info, warn, error).
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info", "inf":
		return LevelInfo, nil
	case "debug", "dbg":
		return LevelDebug, nil
	case "warn", "warning", "wrn":
		return LevelWarn, nil
	case "error", "err":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("log-level: unknown value %q (want debug, info, warn, or error)", s)
	}
}

func levelForKind(entry Entry) Level {
	switch entry.Kind {
	case KindLive:
		return LevelInfo
	case KindDryRun:
		return LevelInfo
	case KindRetry:
		return LevelWarn
	case KindCommandSummary:
		if entry.Outcome == "failed" {
			return LevelError
		}
		return LevelInfo
	default:
		return LevelInfo
	}
}

func entryLevel(entry Entry) Level {
	if entry.Level != 0 {
		return entry.Level
	}
	return levelForKind(entry)
}
