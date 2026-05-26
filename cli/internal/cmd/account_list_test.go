// account_list_test.go tests the account list command output.
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
)

func TestAccountListAWS(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := configstore.RegisterAWSAccount(path, "123456789012", "rh-control"); err != nil {
		t.Fatal(err)
	}
	cfg, err := configstore.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = cfg.SetLinkedAccount("osd-tenant-1", configstore.LinkedAccount{
		AccountID:  "111111111111",
		PayerAlias: "rh-control",
		Role:       "OrganizationAccountAccessRole",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := configstore.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	awsFlags.ConfigPath = path
	t.Cleanup(func() { awsFlags.ConfigPath = "" })

	buf := new(bytes.Buffer)
	accountListCmd.SetOut(buf)
	accountListCmd.SetErr(buf)

	if err := runAccountList(accountListCmd, []string{"aws"}); err != nil {
		t.Fatalf("run: %v", err)
	}

	out := buf.String()
	for _, want := range []string{
		"Registered AWS accounts:",
		"AWS accounts",
		"ALIAS",
		"rh-control",
		"123456789012",
		"payer",
		"osd-tenant-1",
		"111111111111",
		"linked",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestAccountListAWSEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("aws:\n  account_aliases: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	awsFlags.ConfigPath = path
	t.Cleanup(func() { awsFlags.ConfigPath = "" })

	buf := new(bytes.Buffer)
	accountListCmd.SetOut(buf)
	accountListCmd.SetErr(buf)

	if err := runAccountList(accountListCmd, nil); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(buf.String(), "No AWS accounts registered.") {
		t.Fatalf("output = %q", buf.String())
	}
}
