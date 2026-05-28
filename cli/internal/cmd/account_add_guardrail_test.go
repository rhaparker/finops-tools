package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	awsconfig "github.com/openshift-online/finops-tools/cli/internal/aws"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
)

func TestRunAccountAddBlocksDetectedLinkedAccount(t *testing.T) {
	path := testAccountAddConfigPath(t)
	restore := stubAccountAddDependencies(t, account.AddResult{
		Provider:  account.ProviderAWS,
		AccountID: "123456789012",
		Profile:   "linked-profile",
		ARN:       "arn:aws:sts::123456789012:assumed-role/OrganizationAccountAccessRole/test",
	}, awsconfig.AccountKindLinked, nil)
	defer restore()

	accountAddAlias = "finops-s1"
	accountAddPayer = ""
	awsFlags.ConfigPath = path
	t.Cleanup(func() {
		accountAddAlias = ""
		accountAddPayer = ""
		awsFlags.ConfigPath = ""
	})

	cmd := accountAddCmd
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))

	err := runAccountAdd(cmd, []string{"aws", "123456789012"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "linked/member account") {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg, err := configstore.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.AWSAccountIDForAlias("finops-s1"); ok {
		t.Fatal("linked-detected account should not be registered as payer")
	}
}

func TestRunAccountAddWarnsAndRegistersOnUnknown(t *testing.T) {
	path := testAccountAddConfigPath(t)
	restore := stubAccountAddDependencies(t, account.AddResult{
		Provider:  account.ProviderAWS,
		AccountID: "123456789012",
		Profile:   "unknown-profile",
		ARN:       "arn:aws:sts::123456789012:assumed-role/OrganizationAccountAccessRole/test",
	}, awsconfig.AccountKindUnknown, context.DeadlineExceeded)
	defer restore()

	accountAddAlias = "rh-control"
	accountAddPayer = ""
	awsFlags.ConfigPath = path
	t.Cleanup(func() {
		accountAddAlias = ""
		accountAddPayer = ""
		awsFlags.ConfigPath = ""
	})

	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	cmd := accountAddCmd
	cmd.SetOut(out)
	cmd.SetErr(errOut)

	if err := runAccountAdd(cmd, []string{"aws", "123456789012"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(errOut.String(), "warning: unable to determine whether account") {
		t.Fatalf("missing warning, stderr = %q", errOut.String())
	}

	cfg, err := configstore.Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	id, ok := cfg.AWSAccountIDForAlias("rh-control")
	if !ok || id != "123456789012" {
		t.Fatalf("unexpected alias registration: id=%q ok=%v", id, ok)
	}
}

func TestRunAccountAddAllowsDetectedPayer(t *testing.T) {
	path := testAccountAddConfigPath(t)
	restore := stubAccountAddDependencies(t, account.AddResult{
		Provider:  account.ProviderAWS,
		AccountID: "123456789012",
		Profile:   "payer-profile",
		ARN:       "arn:aws:sts::123456789012:assumed-role/OrganizationAccountAccessRole/test",
	}, awsconfig.AccountKindPayer, nil)
	defer restore()

	accountAddAlias = "rh-control"
	accountAddPayer = ""
	awsFlags.ConfigPath = path
	t.Cleanup(func() {
		accountAddAlias = ""
		accountAddPayer = ""
		awsFlags.ConfigPath = ""
	})

	errOut := new(bytes.Buffer)
	cmd := accountAddCmd
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(errOut)

	if err := runAccountAdd(cmd, []string{"aws", "123456789012"}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if strings.Contains(errOut.String(), "warning:") {
		t.Fatalf("unexpected warning: %q", errOut.String())
	}
}

func stubAccountAddDependencies(
	t *testing.T,
	result account.AddResult,
	kind awsconfig.AccountKind,
	kindErr error,
) func() {
	t.Helper()
	prevAdd := addAccountFn
	prevKind := detectAWSAccountKindFn
	addAccountFn = func(context.Context, account.AddOptions) (account.AddResult, error) {
		return result, nil
	}
	detectAWSAccountKindFn = func(context.Context, string, string) (awsconfig.AccountKind, error) {
		return kind, kindErr
	}
	return func() {
		addAccountFn = prevAdd
		detectAWSAccountKindFn = prevKind
	}
}

func testAccountAddConfigPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := configstore.Save(path, configstore.Default()); err != nil {
		t.Fatalf("save config: %v", err)
	}
	return path
}
