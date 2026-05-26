package report

import (
	"io"

	"github.com/nikolalohinski/gonja/v2/loaders"
)

// stableLoader wraps a gonja loader and ignores Inherit path shifts.
// Gonja's EmbedFSLoader double-prefixes paths when extends passes an already-resolved name.
type stableLoader struct {
	inner loaders.Loader
}

func newStableLoader(inner loaders.Loader) loaders.Loader {
	return &stableLoader{inner: inner}
}

func (s *stableLoader) Read(path string) (io.Reader, error) {
	return s.inner.Read(path)
}

func (s *stableLoader) Resolve(path string) (string, error) {
	return s.inner.Resolve(path)
}

func (s *stableLoader) Inherit(string) (loaders.Loader, error) {
	return s, nil
}
