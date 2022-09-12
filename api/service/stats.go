package service

import (
	"github.com/docker/go-units"
	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-photon/sak/time/slots"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/database/orm"
)

type statsResp struct {
	CurrentSlot        uint64 `json:"current_slot"`
	CurrentEpoch       uint64 `json:"current_epoch"`
	FinalizedSlot      uint64 `json:"finalized_slot"`
	FinalizedEpoch     uint64 `json:"finalized_epoch"`
	ValidatorCount     int64  `json:"validator_count"`
	AuditorCount       int64  `json:"auditor_count"`
	NetworkStorageSize string `json:"network_storage_size"`
	ContractCount      int64  `json:"contract_count"`
}

// Stats handles the /stats request.
func (s *Service) Stats(_ *gin.Context) (*statsResp, error) {
	cs := &orm.ChainStatus{}
	if err := s.db.Model(&orm.ChainStatus{}).First(&cs).Error; err != nil {
		return nil, err
	}

	vc := int64(0)
	if err := s.db.Model(&orm.Validator{}).Count(&vc).Error; err != nil {
		return nil, err
	}

	ac := int64(0)
	if err := s.db.Model(&orm.Auditor{}).Count(&ac).Error; err != nil {
		return nil, err
	}

	scc := int64(0)
	if err := s.db.Model(&orm.StorageContract{}).Count(&scc).Error; err != nil {
		return nil, err
	}

	result := new(struct{ Size uint64 })
	if err := s.db.Model(&orm.StorageContract{}).
		Select("sum(size) as size").
		Scan(result).
		Error; err != nil {
		return nil, err
	}

	return &statsResp{
		CurrentSlot:        cs.CurrentSlot,
		CurrentEpoch:       uint64(slots.ToEpoch(pbc.Slot(cs.CurrentSlot))),
		FinalizedSlot:      cs.FinalizedSlot,
		FinalizedEpoch:     uint64(slots.ToEpoch(pbc.Slot(cs.FinalizedSlot))),
		ValidatorCount:     vc,
		AuditorCount:       ac,
		NetworkStorageSize: units.HumanSize(float64(result.Size)),
		ContractCount:      scc,
	}, nil
}
