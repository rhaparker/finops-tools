// Package progress writes human-readable status lines to stderr during long operations.
package progress

import (
	"fmt"
	"io"
)

// Writer emits step messages to w (typically os.Stderr).
type Writer struct {
	w     io.Writer
	quiet bool
}

// New returns a progress writer. When quiet is true, Step is a no-op.
func New(w io.Writer, quiet bool) *Writer {
	return &Writer{w: w, quiet: quiet}
}

// Step prints a status line prefixed with an arrow.
func (p *Writer) Step(message string) {
	if p == nil || p.quiet || p.w == nil {
		return
	}
	_, _ = fmt.Fprintf(p.w, "→ %s\n", message)
}
