package progress

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriterStep(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, false)
	w.Step("Fetching costs")
	if !strings.Contains(buf.String(), "Fetching costs") {
		t.Fatalf("got %q", buf.String())
	}
}

func TestWriterQuiet(t *testing.T) {
	var buf bytes.Buffer
	w := New(&buf, true)
	w.Step("hidden")
	if buf.Len() != 0 {
		t.Fatalf("got %q", buf.String())
	}
}
