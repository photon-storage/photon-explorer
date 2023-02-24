package indexer

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"
	fieldparams "github.com/photon-storage/go-photon/config/fieldparams"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/database/orm"
)

func (e *EventProcessor) rollbackBlock(
	dbTx *gorm.DB,
	block *gateway.BlockResp,
) error {
	if err := e.rollbackTransactions(
		dbTx,
		block.BlockHash,
		block.Txs,
	); err != nil {
		return err
	}

	parentBlock, err := e.node.BlockByHash(block.ParentHash)
	if err != nil {
		return err
	}

	for s := block.Slot; s > parentBlock.Slot; s-- {
		if err := deleteSingleRow(
			dbTx,
			&orm.Block{},
			"slot = ?",
			s,
		); err != nil {
			return err
		}
	}

	/*
	 275-276-277
	    -277
	*/

	if err := updateChainStatus(
		dbTx,
		parentBlock.Slot+1,
		parentBlock.ParentHash,
	); err != nil {
		return err
	}

	e.currentHash = block.ParentHash
	e.nextSlot = parentBlock.Slot + 1
	return nil
}

func (e *EventProcessor) rollbackTransactions(
	dbTx *gorm.DB,
	blockHash string,
	txs []*gateway.Tx,
) error {
	for _, tx := range txs {
		gasUsage := uint64(0)
		switch tx.Type {
		case pbc.TxType_BALANCE_TRANSFER.String():
			if err := rollbackBalanceTransferTx(dbTx, tx); err != nil {
				return err
			}

			gasUsage = fieldparams.BalanceTransferGas
		case pbc.TxType_OBJECT_COMMIT.String():
			if err := e.rollbackObjectCommitTx(
				dbTx,
				blockHash,
				tx,
			); err != nil {
				return err
			}

			gasUsage = fieldparams.ObjectCommitGas
		case pbc.TxType_VALIDATOR_DEPOSIT.String():
			if err := e.rollbackValidatorDepositTx(dbTx, tx); err != nil {
				return err
			}

			gasUsage = fieldparams.ValidatorDepositGas
		case pbc.TxType_AUDITOR_DEPOSIT.String():
			if err := e.rollbackAuditorDepositTx(dbTx, tx); err != nil {
				return err
			}

			gasUsage = fieldparams.AuditorDepositGas
		}

		if err := dbTx.Model(&orm.Account{}).
			Where("public_key", tx.From).
			Updates(map[string]interface{}{
				"nonce":   gorm.Expr("nonce - 1"),
				"balance": gorm.Expr("balance + ?", tx.GasPrice*gasUsage),
			}).Error; err != nil {
			return err
		}

		if err := deleteSingleRow(
			dbTx,
			&orm.Transaction{},
			"hash = ?",
			tx.TxHash,
		); err != nil {
			return err
		}
	}

	return nil
}

func rollbackBalanceTransferTx(dbTx *gorm.DB, tx *gateway.Tx) error {
	amount := int64(tx.BalanceTransfer.Amount)
	if err := updateAccountBalance(
		dbTx,
		tx.BalanceTransfer.To,
		-amount,
	); err != nil {
		return err
	}

	return updateAccountBalance(dbTx, tx.From, amount)
}

func (e *EventProcessor) rollbackObjectCommitTx(
	dbTx *gorm.DB,
	blockHash string,
	tx *gateway.Tx,
) error {
	sc, err := e.node.StorageContract(tx.TxHash, blockHash)
	if err != nil {
		return err
	}

	if err := updateAccountBalance(dbTx, sc.Owner, int64(sc.Fee)); err != nil {
		return err
	}

	if err := updateAccountBalance(
		dbTx,
		sc.Depot,
		int64(sc.Pledge),
	); err != nil {
		return err
	}

	txID := 0
	if err := dbTx.Model(&orm.Transaction{}).
		Where("hash = ?", tx.TxHash).
		Select("id").
		Scan(&txID).
		Error; err != nil {
		return err
	}

	contractID := 0
	if err := dbTx.Model(&orm.StorageContract{}).
		Where("commit_transaction_id = ?", txID).
		Select("id").
		Scan(&contractID).
		Error; err != nil {
		return err
	}

	if err := deleteSingleRow(
		dbTx,
		&orm.TransactionContract{},
		"transaction_id = ? && contract_id = ?",
		txID, contractID,
	); err != nil {
		return err
	}

	return deleteSingleRow(
		dbTx,
		&orm.StorageContract{},
		"id = ?", contractID,
	)
}

func (e *EventProcessor) rollbackValidatorDepositTx(
	dbTx *gorm.DB,
	tx *gateway.Tx,
) error {
	validator, err := e.node.Validator(tx.From)
	if err != nil {
		return err
	}

	accountID, err := getAccountIDByPublicKey(dbTx, tx.From)
	if err != nil {
		return err
	}

	return dbTx.Model(orm.Validator{}).
		Where("account_id = ?", accountID).
		Updates(map[string]interface{}{
			"deposit":          validator.Balance,
			"status":           pbc.ValidatorStatus_value[validator.Status],
			"activation_epoch": validator.ActivationEpoch,
			"exit_epoch":       validator.ExitEpoch,
		}).Error
}

func (e *EventProcessor) rollbackAuditorDepositTx(
	dbTx *gorm.DB,
	tx *gateway.Tx,
) error {
	auditor, err := e.node.Auditor(tx.From)
	if err != nil {
		return err
	}

	accountID, err := getAccountIDByPublicKey(dbTx, tx.From)
	if err != nil {
		return err
	}

	return dbTx.Model(orm.Validator{}).
		Where("account_id = ?", accountID).
		Updates(map[string]interface{}{
			"deposit":          auditor.Balance,
			"status":           pbc.ValidatorStatus_value[auditor.Status],
			"activation_epoch": auditor.ActivationEpoch,
			"exit_epoch":       auditor.ExitEpoch,
		}).Error
}

func deleteSingleRow(
	dbTx *gorm.DB,
	model any,
	query any,
	args ...any,
) error {
	d := dbTx.Model(model).Where(query, args...).Delete(model)
	if err := d.Error; err != nil {
		return err
	}

	if ra := d.RowsAffected; ra != 1 {
		return fmt.Errorf(
			"error deleting single row, row affected is %d",
			ra,
		)
	}

	return nil
}
