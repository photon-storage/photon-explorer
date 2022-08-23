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
	currentSlot     uint64
	currentHash     string
	quit            chan struct{}
}

// NewEventProcessor returns the new instance of EventProcessor.
func NewEventProcessor(
	ctx context.Context,
	refreshInterval uint64,
	nodeEndpoint string,
	db *gorm.DB,
) *EventProcessor {
	currentSlot, currentHash, err := currentChainStatus(db)
	if err != nil {
		log.Fatal("fail query chain status", "error", err)
	}

	return &EventProcessor{
		ctx:             ctx,
		refreshInterval: refreshInterval,
		db:              db,
		node:            chain.NewNodeClient(nodeEndpoint),
		currentSlot:     currentSlot,
		currentHash:     currentHash,
		quit:            make(chan struct{}),
	}
}

// Run executing the timing task of processing chain data.
func (e *EventProcessor) Run() {
	ticker := time.NewTicker(time.Duration(e.refreshInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.quit:
			return

		case <-e.ctx.Done():
			return

		case <-ticker.C:

		}

		headSlot, err := e.node.HeadSlot(e.ctx)
		if err != nil {
			log.Error("request head slot from photon node failed", "error", err)
			continue
		}

		if e.currentSlot >= headSlot {
			log.Info("local slot is best slot")
			continue
		}

		for e.currentSlot < headSlot {
			if err := e.processEvents(e.ctx); err != nil {
				log.Error("indexer fail on sync chain events", "error", err)
				continue
			}
		}
	}
}

// Stop exits event processor
func (e *EventProcessor) Stop() {
	close(e.quit)
}

func (e *EventProcessor) processEvents(ctx context.Context) error {
	nextBlock, err := e.node.BlockBySlot(ctx, e.currentSlot+1)
	if err != nil {
		return err
	}

	if nextBlock.BlockHash == sha256.Zero.Hex() {
		e.currentSlot = nextBlock.Slot
		return e.db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&orm.Block{}).Create(&orm.Block{
				Slot: nextBlock.Slot},
			).Error; err != nil {
				return err
			}

			return updateChainStatus(tx, e.currentSlot, e.currentHash)
		})
	}

	if nextBlock.ParentHash != e.currentHash {
		log.Error("fail on block hash mismatch",
			"remote parent block hash", nextBlock.ParentHash,
			"current block hash", e.currentHash,
		)

		rollbackBlock, err := e.node.BlockByHash(ctx, e.currentHash)
		if err != nil {
			return err
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

	return cs.Slot, cs.Hash, nil
}

func updateChainStatus(db *gorm.DB, slot uint64, hash string) error {
	return db.Model(&orm.ChainStatus{}).Where("id = 1").Updates(
		map[string]interface{}{
			"slot": slot,
			"hash": hash,
		}).Error
}
