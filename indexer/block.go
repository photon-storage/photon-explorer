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

func (e *EventProcessor) processBlock(block *gateway.BlockResp) (string, error) {
	if err := e.db.Transaction(func(tx *gorm.DB) error {
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

		if err := tx.Model(&orm.Block{}).Create(b).Error; err != nil {
			return err
		}

		if err := processAttestations(tx, b.ID, block.Attestations); err != nil {
			return err
		}

		if err := e.processTransactions(tx, b.ID, block.Txs); err != nil {
			return err
		}

		return updateChainStatus(tx, block.Slot, block.BlockHash)
	}); err != nil {
		return "", err
	}

	return block.BlockHash, nil
}

func processAttestations(
	db *gorm.DB,
	blockID uint64,
	attestations []*gateway.Attestation,
) error {
	atts := make([]*orm.Attestation, len(attestations))
	for i, a := range attestations {
		bits := strings.Trim(
			strings.Join(strings.Fields(fmt.Sprint(a.AggregationBits)), ","),
			"[]",
		)
		atts[i] = &orm.Attestation{
			BlockID:         blockID,
			CommitteeIndex:  a.CommitteeIndex,
			AggregationBits: bits,
			SourceEpoch:     a.Source.Epoch,
			SourceHash:      a.Source.Hash,
			TargetEpoch:     a.Target.Epoch,
			TargetHash:      a.Target.Hash,
			Signature:       a.Signature,
		}
	}
	return db.Create(atts).Error
}

func (e *EventProcessor) processTransactions(
	db *gorm.DB,
	blockID uint64,
	transactions []*gateway.Tx,
) error {
	for i, tx := range transactions {
		txID, err := createTransaction(db, blockID, uint64(i), tx)
		if err != nil {
			return err
		}

		switch tx.Type {
		case pbc.TxType_BALANCE_TRANSFER.String():
			if err := e.processBalanceTransferTx(db, tx); err != nil {
				return err
			}

		case pbc.TxType_OBJECT_COMMIT.String():
			if err := e.processObjectCommitTx(db, txID, tx.TxHash); err != nil {
				return err
			}

		case pbc.TxType_OBJECT_AUDIT.String():
			if err := processObjectAuditTx(db, txID, tx.ObjectAudit.Hash); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *EventProcessor) processBalanceTransferTx(
	db *gorm.DB,
	tx *gateway.Tx,
) error {
	if err := e.upsertAccount(
		db,
		tx.From,
		-1*int64(tx.BalanceTransfer.Amount),
	); err != nil {
		return err
	}

	return e.upsertAccount(
		db,
		tx.BalanceTransfer.To,
		int64(tx.BalanceTransfer.Amount),
	)
}

func (e *EventProcessor) processObjectCommitTx(
	db *gorm.DB,
	txID uint64,
	hash string,
) error {
	sr, err := e.node.Storage(e.ctx, hash)
	if err != nil {
		return err
	}

	ownerID, err := e.firstOrCreateAccount(db, sr.Owner)
	if err != nil {
		return err
	}

	providerID, err := e.firstOrCreateAccount(db, sr.Depot)
	if err != nil {
		return err
	}

	auditorID, err := e.firstOrCreateAccount(db, sr.Auditor)
	if err != nil {
		return err
	}

	storage := &orm.StorageContract{
		CommitTransactionID: txID,
		Owner:               ownerID,
		Provider:            providerID,
		Auditor:             auditorID,
		ObjectHash:          sr.ObjectHash,
		Status:              pbc.StorageStatus_value[sr.Status],
		Size:                sr.Size,
		Fee:                 sr.Fee,
		Bond:                sr.Bond,
		StartEpoch:          sr.Start,
		EndEpoch:            sr.End,
	}

	if err := db.Model(&orm.StorageContract{}).Create(storage).Error; err != nil {
		return err
	}

	return db.Model(&orm.TransactionContract{}).Create(&orm.TransactionContract{
		TransactionID: txID,
		ContractID:    storage.ID,
	}).Error
}

func processObjectAuditTx(db *gorm.DB, txID uint64, hash string) error {
	contractID := uint64(0)
	if err := db.Model(&orm.StorageContract{}).Where("object_hash = ?", hash).
		Pluck("id", &contractID).Error; err != nil {
		return err
	}

	return db.Model(&orm.TransactionContract{}).Create(&orm.TransactionContract{
		TransactionID: txID,
		ContractID:    contractID,
	}).Error
}

func createTransaction(
	db *gorm.DB,
	blockID,
	position uint64,
	tx *gateway.Tx,
) (uint64, error) {
	raw, err := json.Marshal(tx)
	if err != nil {
		return 0, err
	}

	ormTx := &orm.Transaction{
		BlockID:  blockID,
		Hash:     tx.TxHash,
		From:     tx.From,
		Position: position,
		GasPrice: tx.GasPrice,
		Type:     pbc.TxType_value[tx.Type],
		Raw:      raw,
	}

	if err := db.Model(&orm.Transaction{}).Create(ormTx).Error; err != nil {
		return 0, err
	}

	return ormTx.ID, nil
}

func (e *EventProcessor) firstOrCreateAccount(db *gorm.DB, address string) (uint64, error) {
	account := &orm.Account{}
	err := db.Model(&orm.Account{}).Where("address = ?", address).First(account).Error
	switch err {
	case gorm.ErrRecordNotFound:
		return e.createAccount(db, address)

	case nil:
		return account.ID, err

	default:
		return 0, err
	}
}

func (e *EventProcessor) upsertAccount(
	db *gorm.DB,
	address string,
	changeAmount int64,
) error {
	err := db.Model(&orm.Account{}).Where("address = ?", address).First(nil).Error
	switch err {
	case gorm.ErrRecordNotFound:
		_, err := e.createAccount(db, address)
		return err

	case nil:
		nonceExpr := gorm.Expr("nonce")
		if changeAmount < 0 {
			nonceExpr = gorm.Expr("nonce + 1")
		}
		return db.Model(&orm.Account{}).Where("address = ?", address).Updates(
			map[string]interface{}{
				"nonce":   nonceExpr,
				"balance": gorm.Expr("balance + ?", changeAmount),
			}).Error

	default:
		return err
	}
}

func (e *EventProcessor) createAccount(db *gorm.DB, address string) (uint64, error) {
	account, err := e.node.Account(e.ctx, address)
	if err != nil {
		return 0, err
	}

	a := &orm.Account{
		Address: address,
		Nonce:   account.Nonce,
		Balance: account.Balance,
	}

	if err := db.Model(&orm.Account{}).Create(a).Error; err != nil {
		return 0, err
	}

	return a.ID, nil
}

func (e *EventProcessor) rollbackBlock(block *gateway.BlockResp) (string, error) {
	// TODO(doris)
	return "", nil
}
