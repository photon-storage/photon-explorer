package indexer

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/photon-storage/go-common/log"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

// EventProcessor is the processor for synchronizing photon change events.
type EventProcessor struct {
	refreshInterval uint64
	db              *gorm.DB
	node            *chain.NodeClient
	currentSlot     uint64
	currentHash     string
}

// NewEventProcessor returns the new instance of EventProcessor.
func NewEventProcessor(refreshInterval uint64, nodeEndpoint string, db *gorm.DB) *EventProcessor {
	currentSlot, currentHash, err := currentChainStatus(db)
	if err != nil {
		log.Fatal("fail query chain status", "error", err)
	}

	return &EventProcessor{
		refreshInterval: refreshInterval,
		db:              db,
		node:            chain.NewNodeClient(nodeEndpoint),
		currentSlot:     currentSlot,
		currentHash:     currentHash,
	}
}

// Run executing the timing task of processing chain data.
func (e *EventProcessor) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(e.refreshInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			headSlot, err := e.node.HeadSlot(ctx)
			if err != nil {
				log.Error("request head slot from photon node failed", "error", err)
				break
			}

			if e.currentSlot >= headSlot {
				log.Info("local slot is best slot")
				break
			}

			for e.currentSlot < headSlot {
				if err := e.processEvents(ctx); err != nil {
					log.Error("keeper fail on sync chain events", "error", err)
					break
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

func (e *EventProcessor) processEvents(ctx context.Context) error {
	nextBlock, err := e.node.BlockBySlot(ctx, e.currentSlot+1)
	if err != nil {
		return err
	}

	if nextBlock.ParentHash != e.currentHash {
		log.Error("fail on block hash mismatch",
			"remote parent block hash", nextBlock.ParentHash,
			"db block hash", e.currentHash,
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
