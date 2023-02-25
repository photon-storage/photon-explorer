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
			log.Error("Error querying chain status", "error", err)
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

			return processGenesis(dbTx)
		}); err != nil {
			log.Error("Error processing genesis events", "error", err)
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
			log.Error("Error requesting chain status from photon node",
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
				hash, slot, err := processSlot(
					e.ctx,
					e.node,
					dbTx,
					currentHash,
					nextSlot,
				)
				if err != nil {
					return err
				}

				if slots.IsEpochStart(pbc.Slot(nextSlot-1)) && slot == nextSlot+1 {
					if err := updateFinalizedChainStatus(
						dbTx,
						cs.Finalized.Slot,
						cs.Finalized.Hash,
					); err != nil {
						return err
					}
				}

				currentHash = hash
				nextSlot = slot
				return nil
			}); err != nil {
				log.Error("Error processing chain events",
					"slot", nextSlot,
					"current_hash", currentHash,
					"error", err,
				)
				break
			}

			if slots.IsEpochStart(pbc.Slot(nextSlot - 1)) {
				if err := e.db.Transaction(func(dbTx *gorm.DB) error {
					return processEpoch(e.ctx, e.node, dbTx)
				}); err != nil {
					log.Error("Error processing epoch",
						"slot", nextSlot-1,
						"error", err,
					)
				}
			}
		}
	}
}

// Stop exits event processor
func (e *EventProcessor) Stop() {
	e.cancel()
}

func processSlot(
	ctx context.Context,
	node *chain.NodeClient,
	dbTx *gorm.DB,
	hash string,
	slot uint64,
) (string, uint64, error) {
	nextBlock, err := node.BlockBySlot(ctx, slot)
	if err != nil {
		return "", 0, err
	}

	if nextBlock.BlockHash == sha256.Zero.Hex() {
		if err := dbTx.Model(&orm.Block{}).
			Create(&orm.Block{Slot: nextBlock.Slot}).
			Error; err != nil {
			return "", 0, err
		}

		return hash, slot + 1, updateChainSlot(dbTx, slot)
	}

	if nextBlock.ParentHash != hash {
		log.Warn("Block hash mismatch",
			"remote parent block hash", nextBlock.ParentHash,
			"current block hash", hash,
		)

		rb, err := node.BlockByHash(ctx, hash)
		if err != nil {
			return "", 0, err
		}

		return rollbackBlock(ctx, node, dbTx, rb)
	}

	return processBlock(ctx, node, dbTx, nextBlock)
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
