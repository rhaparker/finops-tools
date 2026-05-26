// account_add_flags_test.go tests account add command flag validation and wiring.
package cmd

import (
	"path/filepath"
	"testing"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
)

func TestAccountAddPreRunLinkedWithPayerOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := configstore.RegisterAWSAccount(path, "123456789012", "rh-control"); err != nil {
		t.Fatal(err)
	}

	accountAddPayer = "rh-control"
	accountAddRole = ""
	awsFlags.ConfigPath = path
	err := accountAddCmd.PreRunE(accountAddCmd, []string{"aws", "111111111111"})
	accountAddPayer = ""
	awsFlags.ConfigPath = ""
	if err != nil {
		t.Fatalf("got %v", err)
	}
}

func TestResolveLinkedRoleARNUsesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := configstore.Default()
	if err := configstore.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	cmd := accountAddCmd
	arn, err := resolveLinkedRoleARN(cmd, path, "111111111111", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "arn:aws:iam::111111111111:role/OrganizationAccountAccessRole"
	if arn != want {
		t.Fatalf("got %q want %q", arn, want)
	}
}

func TestResolveLinkedRoleARNUsesConfigDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := configstore.SetDefault(path, "aws.linked_role", "CustomRole"); err != nil {
		t.Fatal(err)
	}

	cmd := accountAddCmd
	arn, err := resolveLinkedRoleARN(cmd, path, "111111111111", "")
	if err != nil {
		t.Fatal(err)
	}
	want := "arn:aws:iam::111111111111:role/CustomRole"
	if arn != want {
		t.Fatalf("got %q want %q", arn, want)
	}
}
