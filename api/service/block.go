package service

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-photon/sak/time/slots"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/api/pagination"
	"github.com/photon-storage/photon-explorer/database/orm"
)

type latestBlock struct {
	Epoch     uint64 `json:"epoch"`
	Slot      uint64 `json:"slot"`
	Position  uint64 `json:"position"`
	Timestamp uint64 `json:"timestamp"`
}

// LatestBlocks handles the /latest-blocks request.
func (s *Service) LatestBlocks(
	_ *gin.Context,
	page *pagination.Query,
) (*pagination.Result, error) {
	blks := make([]*orm.Block, 0)
	if err := s.db.Model(&orm.Block{}).Offset(page.Start).
		Limit(page.Limit).Find(&blks).Error; err != nil {
		return nil, err
	}

	lbs := make([]*latestBlock, len(blks))
	for i, blk := range blks {
		lbs[i] = &latestBlock{
			Epoch:     uint64(slots.ToEpoch(pbc.Slot(blk.Slot))),
			Slot:      blk.Slot,
			Position:  uint64(slots.SinceEpochStarts(pbc.Slot(blk.Slot))),
			Timestamp: blk.Timestamp,
		}
	}

	count := int64(0)
	if err := s.db.Model(&orm.Block{}).Count(&count).Error; err != nil {
		return nil, err
	}

	return &pagination.Result{
		Data:  lbs,
		Total: count,
	}, nil
}

type blockResp struct {
	Slot          uint64 `json:"slot"`
	Epoch         uint64 `json:"epoch"`
	TxCount       uint64 `json:"tx_count"`
	Timestamp     uint64 `json:"timestamp"`
	BlockHash     string `json:"block_hash"`
	ParentHash    string `json:"parent_hash"`
	StateHash     string `json:"state_hash"`
	ProposerIndex uint64 `json:"proposer_index"`
	Finalized     bool   `json:"finalized"`
}

// Block handles the /block request.
func (s *Service) Block(c *gin.Context) (*blockResp, error) {
	query := s.db.Model(&orm.Block{})
	if s := c.Query("slot"); s != "" {
		slot, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, err
		}

		query = query.Where("slot = ?", slot)
	} else if hash := c.Query("hash"); hash != "" {
		query = query.Where("hash = ?", hash)
	}

	blk := &orm.Block{}
	if err := query.First(blk).Error; err != nil {
		return nil, err
	}

	txCount := int64(0)
	if err := s.db.Model(&orm.Transaction{}).Where("block_id = ?", blk.ID).
		Count(&txCount).Error; err != nil {
		return nil, err
	}

	finalizedSlot := uint64(0)
	if err := s.db.Model(&orm.ChainStatus{}).Where("id = 1").
		Pluck("finalized_slot", &finalizedSlot).Error; err != nil {
		return nil, err
	}

	return &blockResp{
		Slot:          blk.Slot,
		Epoch:         uint64(slots.ToEpoch(pbc.Slot(blk.Slot))),
		TxCount:       uint64(txCount),
		Timestamp:     blk.Timestamp,
		BlockHash:     blk.Hash,
		ParentHash:    blk.ParentHash,
		StateHash:     blk.StateHash,
		ProposerIndex: blk.ProposalIndex,
		Finalized:     blk.Slot <= finalizedSlot,
	}, nil
}
