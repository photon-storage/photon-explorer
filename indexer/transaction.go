package indexer

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"
	fieldparams "github.com/photon-storage/go-photon/config/fieldparams"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

func processTransactions(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	blockID uint64,
	block *gateway.BlockResp,
) error {
	for i, tx := range block.Txs {
		fromID, err := getAccountIDByPublicKey(dbTx, tx.From)
		if err != nil {
			return err
		}

		txID, err := createTransaction(dbTx, fromID, blockID, uint64(i), tx)
		if err != nil {
			return err
		}

		gasUsage := uint64(0)
		switch tx.Type {
		case pbc.TxType_BALANCE_TRANSFER.String():
			if err := processBalanceTransferTx(dbTx, tx); err != nil {
				return err
			}

			gasUsage = fieldparams.BalanceTransferGas
		case pbc.TxType_OBJECT_COMMIT.String():
			if err := processObjectCommitTx(
				ctx,
				node,
				dbTx,
				txID,
				tx.TxHash,
				block.BlockHash,
			); err != nil {
				return err
			}

			gasUsage = fieldparams.ObjectCommitGas
		case pbc.TxType_OBJECT_AUDIT.String():
			if err := processObjectAuditTx(dbTx, txID, tx.ObjectAudit.Hash); err != nil {
				return err
			}

			gasUsage = fieldparams.ObjectAuditGas
		case pbc.TxType_VALIDATOR_DEPOSIT.String():
			if err := processValidatorDepositTx(
				ctx,
				node,
				dbTx,
				tx.From,
				tx.ValidatorDeposit.Amount,
			); err != nil {
				return err
			}

			gasUsage = fieldparams.ValidatorDepositGas
		case pbc.TxType_AUDITOR_DEPOSIT.String():
			if err := processAuditorDepositTx(
				ctx,
				node,
				dbTx,
				tx.From,
				tx.AuditorDeposit.Amount,
			); err != nil {
				return err
			}

			gasUsage = fieldparams.AuditorDepositGas
		}

		if err := dbTx.Model(&orm.Account{}).
			Where("id", fromID).
			Updates(map[string]interface{}{
				"nonce":   gorm.Expr("nonce + 1"),
				"balance": gorm.Expr("balance - ?", tx.GasPrice*gasUsage),
			}).Error; err != nil {
			return err
		}
	}

	return nil
}

func processBalanceTransferTx(
	dbTx *gorm.DB,
	tx *gateway.Tx,
) error {
	amount := int64(tx.BalanceTransfer.Amount)
	if err := updateAccountBalance(dbTx, tx.From, -amount); err != nil {
		return err
	}

	to := &orm.Account{PublicKey: tx.BalanceTransfer.To}
	if err := dbTx.Model(&orm.Account{}).
		Where(to).
		FirstOrCreate(to).
		Error; err != nil {
		return err
	}

	return updateAccountBalance(dbTx, to.PublicKey, amount)
}

func createTransaction(
	dbTx *gorm.DB,
	fromID uint64,
	blockID uint64,
	position uint64,
	tx *gateway.Tx,
) (uint64, error) {
	raw, err := json.Marshal(tx)
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

func processObjectCommitTx(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	txID uint64,
	txHash string,
	blockHash string,
) error {
	sc, err := node.StorageContract(ctx, txHash, blockHash)
	if err != nil {
		return err
	}

	ownerID, err := getAccountIDByPublicKey(dbTx, sc.Owner)
	if err != nil {
		return err
	}

	depotID, err := getAccountIDByPublicKey(dbTx, sc.Depot)
	if err != nil {
		return err
	}

	auditorID := uint64(0)
	if sc.Auditor != "" {
		auditorID, err = getAccountIDByPublicKey(dbTx, sc.Auditor)
		if err != nil {
			return err
		}
	}

	if err := updateAccountBalance(dbTx, sc.Owner, -int64(sc.Fee)); err != nil {
		return err
	}

	if err := updateAccountBalance(dbTx, sc.Depot, -int64(sc.Pledge)); err != nil {
		return err
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

func processValidatorDepositTx(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	pk string,
	amount uint64,
) error {
	if err := updateAccountBalance(dbTx, pk, -int64(amount)); err != nil {
		return err
	}

	accountID, err := getAccountIDByPublicKey(dbTx, pk)
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

	validator, err := node.Validator(ctx, pk)
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

func processAuditorDepositTx(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	pk string,
	amount uint64,
) error {
	if err := updateAccountBalance(dbTx, pk, -int64(amount)); err != nil {
		return err
	}

	accountID, err := getAccountIDByPublicKey(dbTx, pk)
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

	auditor, err := node.Auditor(ctx, pk)
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
