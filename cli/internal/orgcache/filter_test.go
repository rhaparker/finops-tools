package orgcache

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/openshift-online/finops-tools/cli/internal/cache"
	coreaccount "github.com/openshift-online/finops-tools/core/account"
)

func TestFilterOrganizationAccountsByTagUsesCache(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	payerID := "123456789012"
	store := cache.New(configPath)
	scan := []coreaccount.OrganizationAccountTags{
		{
			Account: coreaccount.OrganizationAccount{ID: "111111111111", Name: "Prod"},
			Tags:    []coreaccount.Tag{{Key: "env", Value: "prod"}},
		},
		{
			Account: coreaccount.OrganizationAccount{ID: "222222222222", Name: "Stage"},
			Tags:    []coreaccount.Tag{{Key: "env", Value: "stage"}},
		},
	}
	if err := cache.Save(store, namespace, cache.Entry[[]coreaccount.OrganizationAccountTags]{
		Key:       payerID,
		FetchedAt: time.Now().UTC(),
		Value:     scan,
	}); err != nil {
		t.Fatal(err)
	}

	origScan := scanOrganizationAccountTags
	t.Cleanup(func() { scanOrganizationAccountTags = origScan })
	scanOrganizationAccountTags = func(context.Context, aws.Config, coreaccount.TagFilterProgress) ([]coreaccount.OrganizationAccountTags, error) {
		t.Fatal("live scan should not run when cache is fresh")
		return nil, nil
	}

	matches, err := FilterOrganizationAccountsByTag(
		context.Background(),
		aws.Config{},
		"env", "prod",
		nil,
		Options{ConfigPath: configPath, PayerID: payerID},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 || matches[0].ID != "111111111111" {
		t.Fatalf("matches = %+v", matches)
	}
}

func TestFilterOrganizationAccountsByTagSkipCache(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	payerID := "123456789012"
	store := cache.New(configPath)
	if err := cache.Save(store, namespace, cache.Entry[[]coreaccount.OrganizationAccountTags]{
		Key:       payerID,
		FetchedAt: time.Now().UTC(),
		Value: []coreaccount.OrganizationAccountTags{
			{
				Account: coreaccount.OrganizationAccount{ID: "111111111111", Name: "Prod"},
				Tags:    []coreaccount.Tag{{Key: "env", Value: "prod"}},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	called := false
	origScan := scanOrganizationAccountTags
	t.Cleanup(func() { scanOrganizationAccountTags = origScan })
	scanOrganizationAccountTags = func(context.Context, aws.Config, coreaccount.TagFilterProgress) ([]coreaccount.OrganizationAccountTags, error) {
		called = true
		return []coreaccount.OrganizationAccountTags{
			{
				Account: coreaccount.OrganizationAccount{ID: "222222222222", Name: "Stage"},
				Tags:    []coreaccount.Tag{{Key: "env", Value: "stage"}},
			},
		}, nil
	}

	matches, err := FilterOrganizationAccountsByTag(
		context.Background(),
		aws.Config{},
		"env", "stage",
		nil,
		Options{ConfigPath: configPath, PayerID: payerID, Skip: true},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected live scan")
	}
	if len(matches) != 1 || matches[0].ID != "222222222222" {
		t.Fatalf("matches = %+v", matches)
	}

	got, err := cache.Load[[]coreaccount.OrganizationAccountTags](store, namespace, payerID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Value) != 1 || got.Value[0].Account.ID != "111111111111" {
		t.Fatal("skip mode should not update cache")
	}
}
