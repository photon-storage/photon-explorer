package indexer

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"

	"github.com/photon-storage/photon-explorer/database/orm"
)

func (e *EventProcessor) processBlock(dbTx *gorm.DB, block *gateway.BlockResp) (string, error) {
	blockID, err := createBlock(dbTx, block)
	if err != nil {
		return "", err
	}

	if err := processAttestations(dbTx, blockID, block.Attestations); err != nil {
		return "", err
	}

	if err := e.processTransactions(dbTx, blockID, block.Txs); err != nil {
		return "", err
	}

	if err := updateChainStatus(dbTx, block.Slot, block.BlockHash); err != nil {
		return "", err
	}

	return block.BlockHash, nil
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
	pk string,
	changeAmount int64,
) error {
	err := dbTx.Model(&orm.Account{}).Where("public_key = ?", pk).First(nil).Error
	switch err {
	case gorm.ErrRecordNotFound:
		_, err := e.createAccount(dbTx, pk)
		return err

	case nil:
		nonceExpr := gorm.Expr("nonce")
		if changeAmount < 0 {
			nonceExpr = gorm.Expr("nonce + 1")
		}
		return dbTx.Model(&orm.Account{}).
			Where("public_key = ?", pk).
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
