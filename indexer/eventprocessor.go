package indexer

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/photon-storage/go-common/log"
	"github.com/photon-storage/go-photon/crypto/sha256"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

// EventProcessor is the processor for synchronizing photon change events.
type EventProcessor struct {
	ctx             context.Context
	refreshInterval uint64
	db              *gorm.DB
	node            *chain.NodeClient
	quit            chan struct{}
}

// NewEventProcessor returns the new instance of EventProcessor.
func NewEventProcessor(
	ctx context.Context,
	refreshInterval uint64,
	nodeEndpoint string,
	db *gorm.DB,
) *EventProcessor {
	return &EventProcessor{
		ctx:             ctx,
		refreshInterval: refreshInterval,
		db:              db,
		node:            chain.NewNodeClient(nodeEndpoint),
		quit:            make(chan struct{}),
	}
}

// Run executing the timing task of processing chain data.
func (e *EventProcessor) Run() {
	ticker := time.NewTicker(time.Duration(e.refreshInterval) * time.Second)
	defer ticker.Stop()

	currentSlot, currentHash := uint64(0), ""
	for ; true; <-ticker.C {
		var err error
		currentSlot, currentHash, err = currentChainStatus(e.db)
		if err == nil {
			break
		} else if err == gorm.ErrRecordNotFound {
			block, err := e.node.BlockBySlot(e.ctx, 0)
			if err != nil {
				log.Error("request block slot 0 failed", "error", err)
				continue
			}

			currentHash, err = e.processBlock(block)
			if err != nil {
				log.Error("process block slot 0 failed", "error", err)
				continue
			}

			break
		}

		log.Error("fail query chain status", "error", err)
	}

	for {
		select {
		case <-e.quit:
			return

		case <-e.ctx.Done():
			return

		case <-ticker.C:

		}

		cs, err := e.node.ChainStatus(e.ctx)
		if err != nil {
			log.Error("request chain status from photon node failed",
				"error", err,
			)
			continue
		}

		headSlot := cs.Best.Slot
		if currentSlot >= headSlot {
			continue
		}

		for currentSlot < headSlot {
			hash, err := e.processEvents(e.ctx, currentSlot+1, currentHash)
			if err != nil {
				log.Error("indexer fail on sync chain events", "error", err)
				break
			}

			currentHash = hash
			currentSlot++
		}

		if currentSlot == headSlot {
			// TODO(doris) update finalized chain status in every epoch
			if err := updateFinalizedChainStatus(
				e.db,
				cs.Finalized.Slot,
				cs.Finalized.Hash,
			); err != nil {
				log.Error("update finalized chain status failed", "error", err)
			}
		}
	}
}

// Stop exits event processor
func (e *EventProcessor) Stop() {
	close(e.quit)
}

func (e *EventProcessor) processEvents(
	ctx context.Context,
	slot uint64,
	hash string,
) (string, error) {
	nextBlock, err := e.node.BlockBySlot(ctx, slot)
	if err != nil {
		return "", err
	}

	if nextBlock.BlockHash == sha256.Zero.Hex() {
		if err := e.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&orm.Block{}).
				Create(&orm.Block{
					Slot: nextBlock.Slot,
				},
				).
				Error; err != nil {
				return err
			}

			return updateChainSlot(tx, slot)
		}); err != nil {
			return "", err
		}

		return hash, nil
	}

	if nextBlock.ParentHash != hash {
		log.Error("fail on block hash mismatch",
			"remote parent block hash", nextBlock.ParentHash,
			"current block hash", hash,
		)

		rollbackBlock, err := e.node.BlockByHash(ctx, hash)
		if err != nil {
			return "", err
		}

		return e.rollbackBlock(rollbackBlock)
	}

	return e.processBlock(nextBlock)
}

func currentChainStatus(db *gorm.DB) (uint64, string, error) {
	cs := &orm.ChainStatus{}
	if err := db.Model(cs).First(cs).Error; err != nil {
		return 0, "", err
	}

	return cs.CurrentSlot, cs.CurrentHash, nil
}

func updateCurrentChainStatus(db *gorm.DB, slot uint64, hash string) error {
	return db.Model(&orm.ChainStatus{}).
		Where("id = 1").
		Updates(map[string]interface{}{
			"current_slot": slot,
			"current_hash": hash,
		}).
		Error
}

func updateFinalizedChainStatus(db *gorm.DB, slot uint64, hash string) error {
	return db.Model(&orm.ChainStatus{}).
		Where("id = 1").
		Updates(map[string]interface{}{
			"finalized_slot": slot,
			"finalized_hash": hash,
		}).
		Error
}

func updateChainSlot(db *gorm.DB, slot uint64) error {
	return db.Model(&orm.ChainStatus{}).
		Where("id = 1").
		Update("current_slot", slot).
		Error
}
