package synchorn

import (
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/photo-storage/photon-explorer/chain"
	"github.com/photo-storage/photon-explorer/config"
	"github.com/photo-storage/photon-explorer/database/orm"
)

// Keeper is the keeper for synchronizing photon change events
type Keeper struct {
	syncSeconds uint64
	db          *gorm.DB
	node        *chain.Node
}

// NewKeeper return the new instance of keeper
func NewKeeper(cfg *config.SyncerConfig, db *gorm.DB) *Keeper {
	return &Keeper{
		syncSeconds: cfg.SyncSeconds,
		db:          db,
		node:        chain.NewNode(cfg.NodeURL),
	}
}

// Run executing the timing task of synchronizing chain data
func (k *Keeper) Run() {
	ticker := time.NewTicker(time.Duration(k.syncSeconds) * time.Second)
	defer ticker.Stop()

	for ; true; <-ticker.C {
		for {
			needSync, err := k.syncEvents()
			if err != nil {
				log.WithField("error", err).Errorln("keeper fail on process block")
				break
			}

			if !needSync {
				break
			}
		}
	}
}

func (k *Keeper) syncEvents() (bool, error) {
	localSlot, localHash, err := k.localChainStatus()
	if err != nil {
		return false, err
	}

	headSlot, err := k.node.HeadSlot()
	if err != nil {
		return false, err
	}

	if localSlot == headSlot {
		return false, nil
	}

	nextBlock, err := k.node.BlockBySlot(localSlot + 1)
	if err != nil {
		return false, err
	}

	if nextBlock.ParentHash != localHash {
		log.WithFields(log.Fields{
			"remote parent block hash": nextBlock.ParentHash,
			"db block hash":            localHash,
		}).Error("fail on block hash mismatch")

		rollbackBlock, err := k.node.BlockByHash(localHash)
		if err != nil {
			return false, err
		}

		return true, k.rollbackBlock(rollbackBlock)
	}

	return true, k.processBlock(nextBlock)
}

func (k *Keeper) localChainStatus() (uint64, string, error) {
	cs := &orm.ChainStatus{}
	if err := k.db.Model(cs).First(cs).Error; err != nil {
		return 0, "", err
	}

	return cs.Slot, cs.Hash, nil
}
