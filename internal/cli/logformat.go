package cli

import "github.com/hrodrig/kzero/internal/log"

func resolvedLogFormat() (log.Format, error) {
	return log.ParseFormat(logFormat)
}
