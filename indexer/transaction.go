package indexer

import (
	"encoding/json"
	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/database/orm"
)

func (e *EventProcessor) processTransactions(
	dbTx *gorm.DB,
	blockID uint64,
	block *gateway.BlockResp,
) error {
	for i, tx := range block.Txs {
		txID, err := e.createTransaction(dbTx, blockID, uint64(i), tx)
		if err != nil {
			return err
		}

		switch tx.Type {
		case pbc.TxType_BALANCE_TRANSFER.String():
			if err := e.processBalanceTransferTx(dbTx, tx); err != nil {
				return err
			}

		case pbc.TxType_OBJECT_COMMIT.String():
			if err := e.processObjectCommitTx(dbTx, txID, tx.TxHash, block.BlockHash); err != nil {
				return err
			}

		case pbc.TxType_OBJECT_AUDIT.String():
			if err := processObjectAuditTx(dbTx, txID, tx.ObjectAudit.Hash); err != nil {
				return err
			}

		case pbc.TxType_VALIDATOR_DEPOSIT.String():
			if err := e.processValidatorDepositTx(
				dbTx,
				tx.From,
				tx.ValidatorDeposit.Amount,
			); err != nil {
				return err
			}

		case pbc.TxType_AUDITOR_DEPOSIT.String():
			if err := e.processAuditorDepositTx(
				dbTx,
				tx.From,
				tx.AuditorDeposit.Amount,
			); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *EventProcessor) processBalanceTransferTx(
	dbTx *gorm.DB,
	tx *gateway.Tx,
) error {
	if err := e.upsertAccount(
		dbTx,
		tx.From,
		-1*int64(tx.BalanceTransfer.Amount),
	); err != nil {
		return err
	}

	return e.upsertAccount(
		dbTx,
		tx.BalanceTransfer.To,
		int64(tx.BalanceTransfer.Amount),
	)
}

func (e *EventProcessor) createTransaction(
	dbTx *gorm.DB,
	blockID,
	position uint64,
	tx *gateway.Tx,
) (uint64, error) {
	raw, err := json.Marshal(tx)
	if err != nil {
		return 0, err
	}

	fromID, err := e.firstOrCreateAccount(dbTx, tx.From)
	if err != nil {
		return 0, err
	}

	ormTx := &orm.Transaction{
		BlockID:       blockID,
		Hash:          tx.TxHash,
		FromAccountID: fromID,
		Position:      position,
		GasPrice:      tx.GasPrice,
		Type:          pbc.TxType_value[tx.Type],
		Raw:           raw,
	}

	if err := dbTx.Model(&orm.Transaction{}).Create(ormTx).Error; err != nil {
		return 0, err
	}

	return ormTx.ID, nil
}

func (e *EventProcessor) processObjectCommitTx(
	dbTx *gorm.DB,
	txID uint64,
	txHash string,
	blockHash string,
) error {
	sc, err := e.node.StorageContract(e.ctx, txHash, blockHash)
	if err != nil {
		return err
	}

	ownerID, err := e.firstOrCreateAccount(dbTx, sc.Owner)
	if err != nil {
		return err
	}

	depotID, err := e.firstOrCreateAccount(dbTx, sc.Depot)
	if err != nil {
		return err
	}

	auditorID := uint64(0)
	if sc.Auditor != "" {
		auditorID, err = e.firstOrCreateAccount(dbTx, sc.Auditor)
		if err != nil {
			return err
		}
	}

	storage := &orm.StorageContract{
		CommitTransactionID: txID,
		OwnerID:             ownerID,
		DepotID:             depotID,
		AuditorID:           auditorID,
		ObjectHash:          sc.ObjectHash,
		Status:              pbc.StorageStatus_value[sc.Status],
		Size:                sc.Size,
		Fee:                 sc.Fee,
		Pledge:              sc.Pledge,
		StartSlot:           sc.Start,
		EndSlot:             sc.End,
	}

	if err := dbTx.Model(&orm.StorageContract{}).Create(storage).Error; err != nil {
		return err
	}

	return dbTx.Model(&orm.TransactionContract{}).
		Create(&orm.TransactionContract{
			TransactionID: txID,
			ContractID:    storage.ID,
		}).Error
}

func processObjectAuditTx(dbTx *gorm.DB, txID uint64, hash string) error {
	contractID := uint64(0)
	if err := dbTx.Model(&orm.StorageContract{}).
		Where("object_hash = ?", hash).
		Pluck("id", &contractID).
		Error; err != nil {
		return err
	}

	return dbTx.Model(&orm.TransactionContract{}).
		Create(&orm.TransactionContract{
			TransactionID: txID,
			ContractID:    contractID,
		}).Error
}

func (e *EventProcessor) processValidatorDepositTx(
	dbTx *gorm.DB,
	pk string,
	amount uint64,
) error {
	accountID, err := e.firstOrCreateAccount(dbTx, pk)
	if err != nil {
		return err
	}

	if err := dbTx.Model(&orm.Validator{}).
		Where("account_id = ?", accountID).
		First(nil).
		Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	} else if err == nil {
		return dbTx.Model(&orm.Validator{}).
			Where("account_id = ?", accountID).
			Update("deposit", gorm.Expr("deposit + ?", amount)).
			Error
	}

	validator, err := e.node.Validator(e.ctx, pk)
	if err != nil {
		return err
	}

	return dbTx.Model(&orm.Validator{}).Create(&orm.Validator{
		AccountID:       accountID,
		Index:           validator.Index,
		Deposit:         amount,
		Status:          pbc.ValidatorStatus_value[validator.Status],
		ActivationEpoch: validator.ActivationEpoch,
		ExitEpoch:       validator.ExitEpoch,
	}).Error
}

func (e *EventProcessor) processAuditorDepositTx(
	dbTx *gorm.DB,
	pk string,
	amount uint64,
) error {
	accountID, err := e.firstOrCreateAccount(dbTx, pk)
	if err != nil {
		return err
	}

	if err := dbTx.Model(&orm.Auditor{}).
		Where("account_id = ?", accountID).
		First(nil).
		Error; err != nil && err != gorm.ErrRecordNotFound {
		return err
	} else if err == nil {
		return dbTx.Model(&orm.Auditor{}).
			Where("account_id = ?", accountID).
			Update("deposit", gorm.Expr("deposit + ?", amount)).
			Error
	}

	auditor, err := e.node.Auditor(e.ctx, pk)
	if err != nil {
		return err
	}

	return dbTx.Model(&orm.Auditor{}).Create(&orm.Auditor{
		AccountID:       accountID,
		Deposit:         amount,
		Status:          pbc.AuditorStatus_value[auditor.Status],
		ActivationEpoch: auditor.ActivationEpoch,
		ExitEpoch:       auditor.ExitEpoch,
	}).Error
}
