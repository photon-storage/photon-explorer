package indexer

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

const defaultPageSize = 100

func processEpoch(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
) error {
	if err := updateAllValidators(ctx, node, dbTx); err != nil {
		return err
	}

	return updateAllAuditors(ctx, node, dbTx)
}

func updateAllValidators(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
) error {
	nextPageToken := ""
	for ok := true; ok; ok = nextPageToken != "" {
		vs, err := node.Validators(ctx, nextPageToken, defaultPageSize)
		if err != nil {
			return err
		}

		for _, v := range vs.Validators {
			if err := resetAccountBalance(
				ctx,
				node,
				dbTx,
				v.PublicKey,
			); err != nil {
				return err
			}

			accountID, err := getAccountIDByPublicKey(dbTx, v.PublicKey)
			if err != nil {
				return err
			}

			if err := dbTx.Model(&orm.Validator{}).
				Where("account_id = ?", accountID).
				Updates(
					map[string]interface{}{
						"status":           pbc.ValidatorStatus_value[v.Status],
						"activation_epoch": v.ActivationEpoch,
						"exit_epoch":       v.ExitEpoch,
					},
				).Error; err != nil {
				return err
			}
		}

		nextPageToken = vs.NextPageToken
	}

	return nil
}

func updateAllAuditors(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
) error {
	nextPageToken := ""
	for ok := true; ok; ok = nextPageToken != "" {
		as, err := node.Auditors(ctx, nextPageToken, defaultPageSize)
		if err != nil {
			if strings.Contains(err.Error(), gateway.ErrNullAuditors.Error()) {
				return nil
			}

			return err
		}

		for _, a := range as.Auditors {
			if err := resetAccountBalance(
				ctx,
				node,
				dbTx,
				a.PublicKey,
			); err != nil {
				return err
			}

			accountID, err := getAccountIDByPublicKey(dbTx, a.PublicKey)
			if err != nil {
				return err
			}

			if err := dbTx.Model(&orm.Auditor{}).
				Where("account_id = ?", accountID).
				Updates(
					map[string]interface{}{
						"status":           pbc.AuditorStatus_value[a.Status],
						"activation_epoch": a.ActivationEpoch,
						"exit_epoch":       a.ExitEpoch,
					},
				).Error; err != nil {
				return err
			}
		}

		nextPageToken = as.NextPageToken
	}

	return nil
}

func resetAccountBalance(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	pk string,
) error {
	account, err := node.Account(ctx, pk)
	if err != nil {
		return err
	}

	return dbTx.Model(&orm.Account{}).
		Where("public_key", pk).
		Update("balance", account.Balance).
		Error
}
