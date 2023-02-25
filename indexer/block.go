package indexer

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/chain/gateway"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

func processBlock(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	block *gateway.BlockResp,
) (string, uint64, error) {
	blockID, err := createBlock(dbTx, block)
	if err != nil {
		return "", 0, err
	}

	if err := processAttestations(
		dbTx,
		blockID,
		block.Attestations,
	); err != nil {
		return "", 0, err
	}

	if err := processTransactions(ctx, node, dbTx, blockID, block); err != nil {
		return "", 0, err
	}

	if err := updateChainStatus(
		dbTx,
		block.Slot+1,
		block.BlockHash,
	); err != nil {
		return "", 0, err
	}

	return block.BlockHash, block.Slot + 1, nil
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
