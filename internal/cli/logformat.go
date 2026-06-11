package cli

import "github.com/hrodrig/kzero/internal/log"

func resolvedLogFormat() (log.Format, error) {
	return log.ParseFormat(logFormat)
}

func applyLogLevel() error {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		return err
	}
	log.SetMinLevel(level)
	return nil
}
