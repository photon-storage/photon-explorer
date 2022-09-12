package indexer

import (
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/database/orm"
)

const validatorPageSize = 100

func (e *EventProcessor) processBlock(block *gateway.BlockResp) (string, error) {
	if err := e.db.Transaction(func(dbTx *gorm.DB) error {
		blockID, err := createBlock(dbTx, block)
		if err != nil {
			return err
		}

		if err := e.processValidators(dbTx); err != nil {
			return err
		}

		if err := processAttestations(dbTx, blockID, block.Attestations); err != nil {
			return err
		}

		if err := e.processTransactions(dbTx, blockID, block.Txs); err != nil {
			return err
		}

		return updateCurrentChainStatus(dbTx, block.Slot, block.BlockHash)
	}); err != nil {
		return "", err
	}

	return block.BlockHash, nil
}

func (e *EventProcessor) processValidators(dbTx *gorm.DB) error {
	count := int64(0)
	if err := dbTx.Model(&orm.Validator{}).Count(&count).Error; err != nil {
		return err
	}

	vs, err := e.node.Validators(e.ctx, 0, validatorPageSize)
	if err != nil {
		return err
	}

	if vs.TotalSize == int(count) {
		return nil
	}

	validators := make([]*orm.Validator, 0)
	for int(count) < vs.TotalSize {
		pageToken := int(count) / validatorPageSize
		vs, err := e.node.Validators(e.ctx, pageToken, validatorPageSize)
		if err != nil {
			return err
		}

		start := int(count) % validatorPageSize
		for i := start; i < len(vs.Validators); i++ {
			v := vs.Validators[i]
			accountID, err := e.firstOrCreateAccount(dbTx, v.PublicKey)
			if err != nil {
				return err
			}

			validators = append(validators, &orm.Validator{
				AccountID:       accountID,
				Index:           v.Index,
				Deposit:         v.Balance,
				ActivationEpoch: v.ActivationEpoch,
				ExitEpoch:       v.ExitEpoch,
			})
		}

		count += int64(len(vs.Validators) - start)
	}

	return dbTx.Model(&orm.Validator{}).Create(validators).Error
}

func processAttestations(
	dbTx *gorm.DB,
	blockID uint64,
	attestations []*gateway.Attestation,
) error {
	for _, a := range attestations {
		bits := strings.Trim(
			strings.Join(strings.Fields(fmt.Sprint(a.AggregationBits)), ","),
			"[]",
		)
		if err := dbTx.Model(&orm.Attestation{}).
			Create(&orm.Attestation{
				BlockID:         blockID,
				CommitteeIndex:  a.CommitteeIndex,
				AggregationBits: bits,
				SourceEpoch:     a.Source.Epoch,
				SourceHash:      a.Source.Hash,
				TargetEpoch:     a.Target.Epoch,
				TargetHash:      a.Target.Hash,
				Signature:       a.Signature,
			}).Error; err != nil {
			return err
		}

		for _, index := range a.AggregationBits {
			if err := dbTx.Model(&orm.Validator{}).
				Where("idx = ?", index).
				Update("attest_block_id", blockID).
				Error; err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *EventProcessor) processTransactions(
	dbTx *gorm.DB,
	blockID uint64,
	transactions []*gateway.Tx,
) error {
	for i, tx := range transactions {
		txID, err := createTransaction(dbTx, blockID, uint64(i), tx)
		if err != nil {
			return err
		}

		switch tx.Type {
		case pbc.TxType_BALANCE_TRANSFER.String():
			if err := e.processBalanceTransferTx(dbTx, tx); err != nil {
				return err
			}

		case pbc.TxType_OBJECT_COMMIT.String():
			if err := e.processObjectCommitTx(dbTx, txID, tx.TxHash); err != nil {
				return err
			}

		case pbc.TxType_OBJECT_AUDIT.String():
			if err := processObjectAuditTx(dbTx, txID, tx.ObjectAudit.Hash); err != nil {
				return err
			}

		case pbc.TxType_VALIDATOR_DEPOSIT.String():
			if err := processValidatorDepositTx(
				dbTx,
				tx.From,
				tx.ValidatorDeposit.Amount,
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

func (e *EventProcessor) processObjectCommitTx(
	dbTx *gorm.DB,
	txID uint64,
	hash string,
) error {
	sc, err := e.node.StorageContract(e.ctx, hash)
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

	auditorID, err := e.firstOrCreateAccount(dbTx, sc.Auditor)
	if err != nil {
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
		Bond:                sc.Bond,
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

func processValidatorDepositTx(dbTx *gorm.DB, address string, amount uint64) error {
	accountID := 0
	if err := dbTx.Model(&orm.Account{}).
		Where("address = ?", address).
		Pluck("id", &accountID).
		Error; err != nil {
		return err
	}

	return dbTx.Model(&orm.Validator{}).
		Where("account_id = ?", address).
		Update("deposit", gorm.Expr("deposit + ?", amount)).
		Error
}

func createBlock(dbTx *gorm.DB, block *gateway.BlockResp) (uint64, error) {
	b := &orm.Block{
		Slot:              block.Slot,
		Hash:              block.BlockHash,
		ParentHash:        block.ParentHash,
		StateHash:         block.StateHash,
		ProposalIndex:     block.ProposerIndex,
		ProposalSignature: block.ProposerSignature,
		RandaoReveal:      block.RandaoReveal,
		Graffiti:          block.Graffiti,
		Timestamp:         block.Timestamp,
	}

	if err := dbTx.Model(&orm.Block{}).Create(b).Error; err != nil {
		return 0, err
	}

	return b.ID, nil
}

func createTransaction(
	dbTx *gorm.DB,
	blockID,
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
		FromPublicKey: tx.From,
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

func (e *EventProcessor) firstOrCreateAccount(dbTx *gorm.DB, pk string) (uint64, error) {
	account := &orm.Account{}
	err := dbTx.Model(&orm.Account{}).Where("public_key = ?", pk).First(account).Error
	switch err {
	case gorm.ErrRecordNotFound:
		return e.createAccount(dbTx, pk)

	case nil:
		return account.ID, err

	default:
		return 0, err
	}
}

func (e *EventProcessor) upsertAccount(
	dbTx *gorm.DB,
	address string,
	changeAmount int64,
) error {
	err := dbTx.Model(&orm.Account{}).Where("address = ?", address).First(nil).Error
	switch err {
	case gorm.ErrRecordNotFound:
		_, err := e.createAccount(dbTx, address)
		return err

	case nil:
		nonceExpr := gorm.Expr("nonce")
		if changeAmount < 0 {
			nonceExpr = gorm.Expr("nonce + 1")
		}
		return dbTx.Model(&orm.Account{}).
			Where("address = ?", address).
			Updates(
				map[string]interface{}{
					"nonce":   nonceExpr,
					"balance": gorm.Expr("balance + ?", changeAmount),
				}).
			Error

	default:
		return err
	}
}

func (e *EventProcessor) createAccount(dbTx *gorm.DB, pk string) (uint64, error) {
	account, err := e.node.Account(e.ctx, pk)
	if err != nil {
		return 0, err
	}

	a := &orm.Account{
		PublicKey: pk,
		Nonce:     account.Nonce,
		Balance:   account.Balance,
	}

	if err := dbTx.Model(&orm.Account{}).Create(a).Error; err != nil {
		return 0, err
	}

	return a.ID, nil
}

func (e *EventProcessor) rollbackBlock(block *gateway.BlockResp) (string, error) {
	// TODO(doris)
	return "", nil
}
