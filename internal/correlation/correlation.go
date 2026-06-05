// Package correlation propagates client.id into hook environments, [dry-run]/[retry] log lines, and the Kubernetes target block.
package correlation

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/hrodrig/kzero/internal/config"
)

// EnvClientID is set on hook, custom, and release script subprocesses when client.id is configured.
const EnvClientID = "KZERO_CLIENT_ID"

// ClientID returns trimmed client.id from cfg, or empty when unset.
func ClientID(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Client.ID)
}

// AppendEnv adds KZERO_CLIENT_ID (when set), KZERO_OS_USER, and KZERO_OS_UID to subprocess env.
func AppendEnv(cfg *config.Config, env []string) []string {
	if id := ClientID(cfg); id != "" {
		env = append(env, EnvClientID+"="+id)
	}
	if u := OperatorUser(); u != "" {
		env = append(env, EnvOSUser+"="+u)
	}
	if uid := OperatorUID(); uid != "" {
		env = append(env, EnvOSUID+"="+uid)
	}
	return env
}

// LogPrefix returns structured key=value fields for engine log lines (includes trailing space when non-empty).
func LogPrefix(cfg *config.Config) string {
	id := ClientID(cfg)
	if id == "" {
		return ""
	}
	return "client_id=" + logValue(id) + " "
}

func logValue(s string) string {
	if needsQuotes(s) {
		return strconv.Quote(s)
	}
	return s
}

func needsQuotes(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) || r == '=' || r == '"' {
			return true
		}
	}
	return false
}
