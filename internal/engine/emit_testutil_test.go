package engine

import (
	"io"

	"github.com/hrodrig/kzero/internal/log"
)

func testEmitter(w io.Writer) *log.Emitter {
	return log.New(w, log.FormatText)
}
