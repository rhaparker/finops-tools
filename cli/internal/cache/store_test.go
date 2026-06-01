package cache

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadFreshAndGetOrFetch(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	store := New(configPath)
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	entry := Entry[string]{
		Key:       "item-1",
		FetchedAt: now,
		Value:     "hello",
	}
	if err := Save(store, "demo", entry); err != nil {
		t.Fatal(err)
	}

	got, err := Load[string](store, "demo", "item-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Value != "hello" {
		t.Fatalf("value = %q", got.Value)
	}
	if !Fresh(got, DefaultTTL, now.Add(30*time.Minute), nil) {
		t.Fatal("expected fresh entry")
	}
	if Fresh(got, DefaultTTL, now.Add(2*DefaultTTL), nil) {
		t.Fatal("expected stale entry")
	}

	fetchCalled := false
	result, err := GetOrFetch(store, "demo", "item-1", Policy{}, now.Add(10*time.Minute), nil, func() (string, error) {
		fetchCalled = true
		return "live", nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if fetchCalled {
		t.Fatal("fetch should not run for fresh cache hit")
	}
	if result.Source != SourceCache || result.Value != "hello" {
		t.Fatalf("result = %+v", result)
	}
}

func TestGetOrFetchSkipAndRefresh(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	store := New(configPath)
	now := time.Now().UTC()

	if err := Save(store, "demo", Entry[int]{
		Key:       "count",
		FetchedAt: now,
		Value:     1,
	}); err != nil {
		t.Fatal(err)
	}

	result, err := GetOrFetch(store, "demo", "count", Policy{Skip: true}, now, nil, func() (int, error) {
		return 99, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value != 99 || result.Source != SourceFetch {
		t.Fatalf("result = %+v", result)
	}

	cached, err := Load[int](store, "demo", "count")
	if err != nil {
		t.Fatal(err)
	}
	if cached.Value != 1 {
		t.Fatal("skip mode should not update cache")
	}

	result, err = GetOrFetch(store, "demo", "count", Policy{Refresh: true}, now, nil, func() (int, error) {
		return 2, nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value != 2 {
		t.Fatalf("result = %+v", result)
	}

	cached, err = Load[int](store, "demo", "count")
	if err != nil {
		t.Fatal(err)
	}
	if cached.Value != 2 {
		t.Fatal("refresh mode should update cache")
	}
}

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	store := New(filepath.Join(dir, "config.yaml"))
	_, err := Load[string](store, "demo", "missing")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not exist, got %v", err)
	}
}
