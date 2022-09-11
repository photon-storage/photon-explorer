package service

import (
	"fmt"

	"github.com/docker/go-units"
	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-photon/config/config"
	"github.com/photon-storage/go-photon/sak/time/slots"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/api/pagination"
	"github.com/photon-storage/photon-explorer/database/orm"
)

type storageContract struct {
	Hash            string `json:"hash"`
	Size            string `json:"size"`
	Owner           string `json:"owner"`
	Depot           string `json:"depot"`
	FeePerEpoch     string `json:"fee_per_epoch"`
	TimeSinceCommit uint64 `json:"time_since_commit"`
	Duration        uint64 `json:"duration"`
	Status          string `json:"status"`
}

// StorageContracts handles the /storage-contracts request.
func (s *Service) StorageContracts(
	c *gin.Context,
	page *pagination.Query,
) (*pagination.Result, error) {
	query := s.db.Model(&orm.StorageContract{}).
		Preload("Owner").Preload("Depot")

	if pk := c.Query("public_key"); pk != "" {
		query = query.Joins("join accounts on accounts.id = "+
			"storage_contracts.owner_id").Where("public_key = ?", pk)
	}

	scs := make([]*orm.StorageContract, 0)
	if err := query.Offset(page.Start).Limit(page.Limit).
		Order("id desc").Find(&scs).Error; err != nil {
		return nil, err
	}

	currentSlot := uint64(0)
	if err := s.db.Model(&orm.ChainStatus{}).Where("id = 1").
		Pluck("current_slot", &currentSlot).Error; err != nil {
		return nil, err
	}

	storageContracts := make([]*storageContract, len(scs))
	for i, sc := range scs {
		fpe := fmt.Sprintf("%.2f",
			float64(sc.Fee)/float64(slots.ToEpoch(pbc.Slot(sc.EndSlot-sc.StartSlot))),
		)

		storageContracts[i] = &storageContract{
			Hash:            sc.ObjectHash,
			Size:            units.HumanSize(float64(sc.Size)),
			Owner:           sc.Owner.PublicKey,
			Depot:           sc.Depot.PublicKey,
			FeePerEpoch:     fpe,
			TimeSinceCommit: (currentSlot - sc.StartSlot) * config.Consensus().SecondsPerSlot,
			Duration:        (sc.EndSlot - sc.StartSlot) * config.Consensus().SecondsPerSlot,
			Status:          pbc.StorageStatus_name[sc.Status],
		}
	}

	count := int64(0)
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}

	return &pagination.Result{
		Data:  storageContracts,
		Total: count,
	}, nil
}
