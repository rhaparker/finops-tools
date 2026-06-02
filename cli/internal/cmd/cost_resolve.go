// cost_resolve.go resolves cost/report account targets including OU expansion.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openshift-online/finops-tools/cli/internal/configstore"
	coreaccount "github.com/openshift-online/finops-tools/core/account"
	"github.com/openshift-online/finops-tools/core/cost"
	"github.com/spf13/cobra"
)

type costTargetSelector struct {
	AccountIDs   []string
	Aliases      []string
	OUIDs        []string
	PayerAlias   string
	OUDirectOnly bool
}

func parseCostTargetSelector(accountFlag, aliasFlag, ouFlag, payerFlag string, ouDirect bool) (costTargetSelector, error) {
	sel := costTargetSelector{
		PayerAlias:   strings.TrimSpace(payerFlag),
		OUDirectOnly: ouDirect,
	}
	var err error

	if strings.TrimSpace(accountFlag) != "" {
		sel.AccountIDs, err = configstore.ParseAWSAccountIDs(accountFlag)
		if err != nil {
			return costTargetSelector{}, err
		}
	}
	if strings.TrimSpace(aliasFlag) != "" {
		sel.Aliases, err = configstore.ParseAccountAliases(aliasFlag)
		if err != nil {
			return costTargetSelector{}, err
		}
	}
	if strings.TrimSpace(ouFlag) != "" {
		sel.OUIDs, err = configstore.ParseOUIDs(ouFlag)
		if err != nil {
			return costTargetSelector{}, err
		}
	}
	return sel, nil
}

func validateCostTargetSelector(sel costTargetSelector) error {
	if len(sel.AccountIDs) == 0 && len(sel.Aliases) == 0 && len(sel.OUIDs) == 0 {
		return errors.New("at least one of --account, --account-alias, or --ou is required")
	}
	if len(sel.OUIDs) > 0 && sel.PayerAlias == "" {
		return fmt.Errorf("--ou requires --payer")
	}
	if sel.OUDirectOnly && len(sel.OUIDs) == 0 {
		return fmt.Errorf("--ou-direct requires --ou")
	}
	if sel.PayerAlias != "" && len(sel.AccountIDs) == 0 && len(sel.OUIDs) == 0 {
		return fmt.Errorf("--payer requires --account or --ou")
	}
	return nil
}

func resolveCostTargetsWithOU(
	ctx context.Context,
	cmd *cobra.Command,
	cfg configstore.File,
	sel costTargetSelector,
	configPath, credentialsFile, authMethod string,
) ([]cost.AccountTarget, error) {
	var ouTargets, explicitTargets []cost.AccountTarget
	var err error

	if len(sel.OUIDs) > 0 {
		payerID, ok := cfg.PayerAccountIDForAlias(sel.PayerAlias)
		if !ok {
			return nil, fmt.Errorf("unknown payer alias %q (register payer with: finops account add aws <12-digit-id> --alias %s)", sel.PayerAlias, sel.PayerAlias)
		}
		payerTarget := cost.AccountTarget{AccountID: payerID}
		if err := ensureCostCredentials(ctx, cmd, cfg, []cost.AccountTarget{payerTarget}, configPath, credentialsFile, authMethod); err != nil {
			return nil, err
		}
		payerCfg, err := loadAWSConfigForCredentialsAccount(ctx, cfg, payerID, credentialsFile)
		if err != nil {
			return nil, err
		}

		memberIDs := make([]string, 0)
		seenMembers := make(map[string]struct{})
		for _, ouID := range sel.OUIDs {
			accounts, err := coreaccount.ListAccountsInOU(ctx, payerCfg, ouID, coreaccount.ListAccountsInOUOptions{
				DirectOnly: sel.OUDirectOnly,
			})
			if err != nil {
				return nil, fmt.Errorf("OU %s: %w", ouID, err)
			}
			if len(accounts) == 0 {
				return nil, fmt.Errorf("no active accounts found in OU %s", ouID)
			}
			for _, acct := range accounts {
				if _, ok := seenMembers[acct.ID]; ok {
					continue
				}
				seenMembers[acct.ID] = struct{}{}
				memberIDs = append(memberIDs, acct.ID)
			}
		}

		ouTargets, err = configstore.ResolveOUAccountTargets(cfg, memberIDs, sel.PayerAlias)
		if err != nil {
			return nil, err
		}
	}

	if len(sel.AccountIDs) > 0 || len(sel.Aliases) > 0 {
		explicitTargets, err = configstore.ResolveCostTargets(cfg, sel.AccountIDs, sel.Aliases, sel.PayerAlias)
		if err != nil {
			return nil, err
		}
	}

	targets := mergeCostTargets(ouTargets, explicitTargets)
	if len(targets) == 0 {
		return nil, errors.New("no accounts selected")
	}
	return targets, nil
}

func mergeCostTargets(segments ...[]cost.AccountTarget) []cost.AccountTarget {
	seen := make(map[string]cost.AccountTarget)
	order := make([]string, 0)
	for _, segment := range segments {
		for _, target := range segment {
			id := strings.TrimSpace(target.AccountID)
			if id == "" {
				continue
			}
			if existing, ok := seen[id]; ok {
				if existing.DisplayAlias == "" && target.DisplayAlias != "" {
					seen[id] = target
				}
				continue
			}
			seen[id] = target
			order = append(order, id)
		}
	}
	out := make([]cost.AccountTarget, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out
}
