package core

import (
	"context"
	"testing"
)

func TestHello(t *testing.T) {
	got, err := Hello(context.Background())
	if err != nil {
		t.Fatalf("Hello() error = %v", err)
	}
	if got != "hello" {
		t.Errorf("Hello() = %q, want %q", got, "hello")
	}
}
