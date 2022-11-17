package indexer

import (
	"gorm.io/gorm"

	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/database/orm"
)

func (e *EventProcessor) processEpoch() error {
	if err := e.updateAllValidators(); err != nil {
		return err
	}

	return e.updateAllAuditors()
}

func (e *EventProcessor) updateAllValidators() error {
	vs := make([]*orm.Validator, 0)
	if err := e.db.Model(&orm.Validator{}).
		Preload("Account").
		Find(&vs).Error; err != nil {
		return err
	}

	for _, v := range vs {
		if err := e.db.Transaction(func(dbTx *gorm.DB) error {
			pk := v.Account.PublicKey
			if err := e.updateAccountBalance(dbTx, pk); err != nil {
				return err
			}

			validator, err := e.node.Validator(e.ctx, pk)
			if err != nil {
				return err
			}

			return dbTx.Model(&orm.Validator{}).
				Where("account_id = ?", v.AccountID).
				Updates(
					map[string]interface{}{
						"status":           pbc.ValidatorStatus_value[validator.Status],
						"activation_epoch": validator.ActivationEpoch,
						"exit_epoch":       validator.ExitEpoch,
					},
				).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func (e *EventProcessor) updateAllAuditors() error {
	as := make([]*orm.Auditor, 0)
	if err := e.db.Model(&orm.Auditor{}).
		Preload("Account").
		Find(&as).Error; err != nil {
		return err
	}

	for _, a := range as {
		if err := e.db.Transaction(func(dbTx *gorm.DB) error {
			pk := a.Account.PublicKey
			if err := e.updateAccountBalance(dbTx, pk); err != nil {
				return err
			}

			auditor, err := e.node.Auditor(e.ctx, pk)
			if err != nil {
				return err
			}

			return dbTx.Model(&orm.Auditor{}).
				Where("account_id = ?", a.AccountID).
				Updates(
					map[string]interface{}{
						"status":           pbc.AuditorStatus_value[auditor.Status],
						"activation_epoch": auditor.ActivationEpoch,
						"exit_epoch":       auditor.ExitEpoch,
					}).Error
		}); err != nil {
			return err
		}
	}

	return nil
}

func (e *EventProcessor) updateAccountBalance(dbTx *gorm.DB, pk string) error {
	account, err := e.node.Account(e.ctx, pk)
	if err != nil {
		return err
	}

	return dbTx.Model(&orm.Account{}).
		Where("public_key", pk).
		Update("balance", account.Balance).
		Error
}
