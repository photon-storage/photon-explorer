package indexer

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/photon-storage/go-common/log"
	"github.com/photon-storage/go-photon/crypto/sha256"
	"github.com/photon-storage/go-photon/sak/time/slots"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/chain"
	"github.com/photon-storage/photon-explorer/database/orm"
)

// EventProcessor is the processor for synchronizing photon change events.
type EventProcessor struct {
	ctx             context.Context
	cancel          context.CancelFunc
	refreshInterval uint64
	db              *gorm.DB
	node            *chain.NodeClient
}

// NewEventProcessor returns the new instance of EventProcessor.
func NewEventProcessor(
	ctx context.Context,
	refreshInterval uint64,
	nodeEndpoint string,
	db *gorm.DB,
) *EventProcessor {
	ctx, cancel := context.WithCancel(ctx)
	return &EventProcessor{
		ctx:             ctx,
		cancel:          cancel,
		refreshInterval: refreshInterval,
		db:              db,
		node:            chain.NewNodeClient(nodeEndpoint),
	}
}

// Run executing the timing task of processing chain data.
func (e *EventProcessor) Run() {
	ticker := time.NewTicker(time.Duration(e.refreshInterval) * time.Second)
	defer ticker.Stop()

	nextSlot, currentHash := uint64(0), sha256.Zero.Hex()
	for {
		select {
		case <-e.ctx.Done():
			return

		case <-ticker.C:

		}

		var err error
		nextSlot, currentHash, err = chainStatus(e.db)
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Error("Fail to query chain status", "error", err)
			continue
		} else if err == nil {
			break
		}

		if err := e.db.Transaction(func(dbTx *gorm.DB) error {
			if err := dbTx.Model(&orm.ChainStatus{}).
				Create(&orm.ChainStatus{
					NextSlot:    0,
					CurrentHash: sha256.Zero.Hex(),
				}).
				Error; err != nil {
				return err
			}

			return e.processGenesisValidators(dbTx)
		}); err != nil {
			log.Error("Process genesis events failed", "error", err)
		} else {
			break
		}
	}

	for {
		select {
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
		if nextSlot > headSlot {
			continue
		}

		for nextSlot <= headSlot {
			select {
			case <-e.ctx.Done():
				return

			default:

			}

			if err := e.db.Transaction(func(dbTx *gorm.DB) error {
				hash, err := e.processSlots(dbTx, nextSlot, currentHash)
				if err != nil {
					return err
				}

				currentHash = hash
				nextSlot++
				return nil
			}); err != nil {
				log.Error("indexer fail on sync chain events", "error", err)
				break
			}

			if slots.IsEpochStart(pbc.Slot(nextSlot)) {
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
}

// Stop exits event processor
func (e *EventProcessor) Stop() {
	e.cancel()
}

func (e *EventProcessor) processSlots(
	dbTx *gorm.DB,
	slot uint64,
	hash string,
) (string, error) {
	nextBlock, err := e.node.BlockBySlot(e.ctx, slot)
	if err != nil {
		return "", err
	}

	if nextBlock.BlockHash == sha256.Zero.Hex() {
		if err := dbTx.Model(&orm.Block{}).
			Create(&orm.Block{Slot: nextBlock.Slot}).
			Error; err != nil {
			return "", err
		}

		return hash, updateChainSlot(dbTx, slot)
	}

	if nextBlock.ParentHash != hash {
		log.Error("fail on block hash mismatch",
			"remote parent block hash", nextBlock.ParentHash,
			"current block hash", hash,
		)

		rollbackBlock, err := e.node.BlockByHash(e.ctx, hash)
		if err != nil {
			return "", err
		}

		return e.rollbackBlock(rollbackBlock)
	}

	return e.processBlock(dbTx, nextBlock)
}

func chainStatus(db *gorm.DB) (uint64, string, error) {
	cs := &orm.ChainStatus{}
	if err := db.Model(cs).First(cs).Error; err != nil {
		return 0, sha256.Zero.Hex(), err
	}

	return cs.NextSlot, cs.CurrentHash, nil
}

func updateChainStatus(dbTx *gorm.DB, slot uint64, hash string) error {
	return dbTx.Model(&orm.ChainStatus{}).
		Where("id = 1").
		Updates(map[string]interface{}{
			"next_slot":    slot,
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
		Update("next_slot", slot).
		Error
}
