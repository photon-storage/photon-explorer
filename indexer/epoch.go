package indexer

import (
	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

const defaultPageSize = 100

func (e *EventProcessor) processEpoch() error {
	if err := e.updateAllValidators(); err != nil {
		return err
	}

	return e.updateAllAuditors()
}

func (e *EventProcessor) updateAllValidators() error {
	nextPageToken := ""
	for ok := true; ok; ok = nextPageToken != "" {
		vs, err := e.node.Validators(nextPageToken, defaultPageSize)
		if err != nil {
			return err
		}

		for _, v := range vs.Validators {
			if err := e.db.Transaction(func(dbTx *gorm.DB) error {
				pk := v.PublicKey
				if err := resetAccountBalance(e.node, dbTx, pk); err != nil {
					return err
				}

				accountID, err := getAccountIDByPublicKey(dbTx, pk)
				if err != nil {
					return err
				}

				return dbTx.Model(&orm.Validator{}).
					Where("account_id = ?", accountID).
					Updates(
						map[string]interface{}{
							"status":           pbc.ValidatorStatus_value[v.Status],
							"activation_epoch": v.ActivationEpoch,
							"exit_epoch":       v.ExitEpoch,
						},
					).Error
			}); err != nil {
				return err
			}
		}

		nextPageToken = vs.NextPageToken
	}

	return nil
}

func (e *EventProcessor) updateAllAuditors() error {
	nextPageToken := ""
	for ok := true; ok; ok = nextPageToken != "" {
		as, err := e.node.Auditors(nextPageToken, defaultPageSize)
		if err != nil && err != gateway.ErrNullAuditors {
			return err
		}

		if err == gateway.ErrNullAuditors {
			return nil
		}

		for _, a := range as.Auditors {
			if err := e.db.Transaction(func(dbTx *gorm.DB) error {
				pk := a.PublicKey
				if err := resetAccountBalance(e.node, dbTx, pk); err != nil {
					return err
				}

				accountID, err := getAccountIDByPublicKey(dbTx, pk)
				if err != nil {
					return err
				}

				return dbTx.Model(&orm.Auditor{}).
					Where("account_id = ?", accountID).
					Updates(
						map[string]interface{}{
							"status":           pbc.AuditorStatus_value[a.Status],
							"activation_epoch": a.ActivationEpoch,
							"exit_epoch":       a.ExitEpoch,
						},
					).Error
			}); err != nil {
				return err
			}
		}

		nextPageToken = as.NextPageToken
	}

	return nil
}

func resetAccountBalance(
	node *chain.NodeClient,
	dbTx *gorm.DB,
	pk string,
) error {
	account, err := node.Account(pk)
	if err != nil {
		return err
	}

	return dbTx.Model(&orm.Account{}).
		Where("public_key", pk).
		Update("balance", account.Balance).
		Error
}
