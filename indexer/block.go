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

func (e *EventProcessor) processBlock(block *gateway.BlockResp) error {
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

		return e.processTransactions(tx, b.ID, block.Txs)
	}); err != nil {
		return err
	}

	e.currentSlot = block.Slot
	e.currentHash = block.BlockHash
	return nil
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
	txs := make([]*orm.Transaction, len(transactions))
	for i, t := range transactions {
		raw, err := json.Marshal(t)
		if err != nil {
			return err
		}

		txs[i] = &orm.Transaction{
			BlockID:  blockID,
			Hash:     t.TxHash,
			From:     t.From,
			Position: uint64(i),
			GasPrice: t.GasPrice,
			Type:     pbc.TxType_value[t.Type],
			Raw:      string(raw),
		}

		if t.Type == pbc.TxType_BALANCE_TRANSFER.String() {
			if err := e.applyTransfer(
				db,
				t.From,
				-1*int64(t.BalanceTransfer.Amount),
			); err != nil {
				return err
			}

			if err := e.applyTransfer(
				db,
				t.BalanceTransfer.To,
				int64(t.BalanceTransfer.Amount),
			); err != nil {
				return err
			}
		}
	}
	return db.Create(txs).Error
}

func (e *EventProcessor) applyTransfer(
	db *gorm.DB,
	address string,
	changeAmount int64,
) error {
	err := db.Model(&orm.Account{}).Where("address = ?", address).First(nil).Error
	switch err {
	case gorm.ErrRecordNotFound:
		account, err := e.node.Account(e.ctx, address)
		if err != nil {
			return err
		}

		return db.Model(&orm.Account{}).Create(&orm.Account{
			Address: address,
			Nonce:   account.Nonce,
			Balance: account.Balance,
		}).Error

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

func (e *EventProcessor) rollbackBlock(block *gateway.BlockResp) error {
	// TODO(doris)
	return nil
}
