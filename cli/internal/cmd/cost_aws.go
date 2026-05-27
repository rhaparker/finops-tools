// cost_aws.go ensures AWS credentials for each cost target (payer and linked) before calling core/cost.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/account"
	"github.com/openshift-online/finops-tools/cli/internal/awsauth"
	awsconfig "github.com/openshift-online/finops-tools/cli/internal/aws"
	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/openshift-online/finops-tools/core/cost"
	"github.com/spf13/cobra"
)

func ensureCostCredentials(
	ctx context.Context,
	cmd *cobra.Command,
	cfg configstore.File,
	targets []cost.AccountTarget,
	configPath, credentialsFile, authMethod string,
) error {
	seen := make(map[string]struct{})
	for i := range targets {
		credID := targets[i].CredentialsAccountID()
		if _, ok := seen[credID]; ok {
			continue
		}
		seen[credID] = struct{}{}

		ensureOpts, err := newAWSEnsureOptions(cmd, awsEnsureConfig{
			configPath:      configPath,
			authMethodFlag:  authMethod,
			credentialsFile: credentialsFile,
		})
		if err != nil {
			return err
		}
		ensureOpts.AccountName = credID
		ensureOpts.ProfileNames = account.AWSProfileNames(credID, cfg.PayerAliasForAccountID(credID), nil)

		if _, err := awsauth.EnsureAccountCredentials(ctx, ensureOpts); err != nil {
			return fmt.Errorf("%s: %w", credID, mapCredentialError(credID, err))
		}
	}
	return nil
}

func prepareCostTargets(
	ctx context.Context,
	store configstore.File,
	targets []cost.AccountTarget,
	credentialsFile string,
) ([]cost.AccountTarget, error) {
	credConfigs := make(map[string]aws.Config)
	for i := range targets {
		credID := targets[i].CredentialsAccountID()
		if awsCfg, ok := credConfigs[credID]; ok {
			targets[i].AWSConfig = awsCfg
			if err := enrichCostTargetDisplayName(ctx, &targets[i], store, credentialsFile); err != nil {
				return nil, err
			}
			continue
		}

		profileNames := account.AWSProfileNames(credID, store.PayerAliasForAccountID(credID), nil)
		res, status, err := awsconfig.ResolveCredentials(ctx, awsconfig.ResolveOptions{
			AccountName:     credID,
			ProfileNames:    profileNames,
			CredentialsPath: credentialsFile,
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", credID, err)
		}
		if status != awsconfig.CredentialsValid {
			return nil, fmt.Errorf("%s: %w", credID, mapCredentialStatusError(credID, status))
		}

		profile := res.Profile
		if profile == "" {
			profile = awsconfig.SanitizeProfileName(credID)
		}
		awsCfg, err := awsconfig.LoadSharedConfigProfile(ctx, profile)
		if err != nil {
			return nil, fmt.Errorf("%s: load AWS profile %q: %w", credID, profile, err)
		}
		credConfigs[credID] = awsCfg
		targets[i].AWSConfig = awsCfg

		if err := enrichCostTargetDisplayName(ctx, &targets[i], store, credentialsFile); err != nil {
			return nil, err
		}
	}
	return targets, nil
}

func enrichCostTargetDisplayName(
	ctx context.Context,
	target *cost.AccountTarget,
	store configstore.File,
	_ string,
) error {
	accountID := strings.TrimSpace(target.AccountID)
	if accountID == "" {
		return nil
	}

	if name, err := awsconfig.AccountName(ctx, target.AWSConfig, accountID); err == nil && strings.TrimSpace(name) != "" {
		target.DisplayName = name
		return nil
	}

	lookupProfiles := account.AWSProfileNames(accountID, store.AliasForAccountID(accountID), nil)
	if payerID := target.PayerAccountID; payerID != "" {
		lookupProfiles = mergeProfileNames(
			lookupProfiles,
			account.AWSProfileNames(payerID, store.PayerAliasForAccountID(payerID), nil),
		)
	}

	for _, profile := range lookupProfiles {
		awsCfg, err := awsconfig.LoadSharedConfigProfile(ctx, profile)
		if err != nil {
			continue
		}
		name, err := awsconfig.AccountName(ctx, awsCfg, accountID)
		if err == nil {
			target.DisplayName = name
			return nil
		}
	}

	return nil
}

func mapCredentialError(accountID string, err error) error {
	if errors.Is(err, awsconfig.ErrCredentialsNotFound) {
		return fmt.Errorf("%w (run: finops account add aws %s)", err, accountID)
	}
	if errors.Is(err, awsconfig.ErrCredentialsInvalid) {
		return fmt.Errorf("%w (run: finops account add aws %s)", err, accountID)
	}
	return err
}

func mapCredentialStatusError(accountID string, status awsconfig.ResolveStatus) error {
	switch status {
	case awsconfig.CredentialsInvalid:
		return fmt.Errorf("%w (run: finops account add aws %s)", awsconfig.ErrCredentialsInvalid, accountID)
	default:
		return fmt.Errorf("%w (run: finops account add aws %s)", awsconfig.ErrCredentialsNotFound, accountID)
	}
}

func mergeProfileNames(segments ...[]string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, segment := range segments {
		for _, name := range segment {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}
