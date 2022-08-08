package sync

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
	syncSeconds uint64
	db          *gorm.DB
	node        *chain.Node
}

// NewEventProcessor returns the new instance of EventProcessor.
func NewEventProcessor(syncSeconds uint64, nodeURL string, db *gorm.DB) *EventProcessor {
	return &EventProcessor{
		syncSeconds: syncSeconds,
		db:          db,
		node:        chain.NewNode(nodeURL),
	}
}

// Run executing the timing task of processing chain data.
func (e *EventProcessor) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(e.syncSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			localSlot, localHash, err := e.localChainStatus()
			if err != nil {
				log.Error("fail query chain status", "error", err)
				break
			}

			headSlot, err := e.node.HeadSlot(ctx)
			if err != nil {
				log.Error("request head slot from photon node failed", "error", err)
				break
			}

			if localSlot >= headSlot {
				log.Info("local slot is best slot")
				break
			}

			if err := e.processEvents(ctx, localSlot, localHash); err != nil {
				log.Error("keeper fail on sync chain events", "error", err)
				break
			}

		case <-ctx.Done():
			return
		}
	}
}

func (e *EventProcessor) processEvents(ctx context.Context, slot uint64, hash string) error {
	nextBlock, err := e.node.BlockBySlot(ctx, slot+1)
	if err != nil {
		return err
	}

	if nextBlock.ParentHash != hash {
		log.Error("fail on block hash mismatch",
			"remote parent block hash", nextBlock.ParentHash,
			"db block hash", hash,
		)

		rollbackBlock, err := e.node.BlockByHash(ctx, hash)
		if err != nil {
			return err
		}

		return e.rollbackBlock(rollbackBlock)
	}

	return e.processBlock(nextBlock)
}

func (e *EventProcessor) localChainStatus() (uint64, string, error) {
	cs := &orm.ChainStatus{}
	if err := e.db.Model(cs).First(cs).Error; err != nil {
		return 0, "", err
	}

	return cs.Slot, cs.Hash, nil
}
