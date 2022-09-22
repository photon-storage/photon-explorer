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

type storageContractResp struct {
	Hash         string             `json:"hash"`
	ObjectHash   string             `json:"object_hash"`
	Status       string             `json:"status"`
	Fee          string             `json:"fee"`
	Bond         string             `json:"bond"`
	Owner        string             `json:"owner"`
	Depot        string             `json:"depot"`
	Auditor      string             `json:"auditor"`
	StartEpoch   uint64             `json:"start_epoch"`
	EndEpoch     uint64             `json:"end_epoch"`
	Transactions []*baseTransaction `json:"transactions"`
}

// StorageContract handles the /storage-contract request.
func (s *Service) StorageContract(c *gin.Context) (*storageContractResp, error) {
	hash := c.Query("hash")
	if hash == "" {
		return nil, errMissingContractHash
	}

	sc := &orm.StorageContract{}
	if err := s.db.Model(&orm.StorageContract{}).
		Preload("Owner").
		Preload("Depot").
		Preload("Auditor").
		Joins("join transactions as t on t.id = storage_contracts.commit_transaction_id").
		Where("t.hash = ?", hash).
		First(sc).
		Error; err != nil {
		return nil, err
	}

	txs := make([]*orm.Transaction, 0)
	if err := s.db.Model(&orm.Transaction{}).
		Preload("Block").
		Joins("join transaction_contracts as tc on tc.transaction_id = transactions.id").
		Where("tc.contract_id = ?", sc.ID).
		Order("id desc").
		Find(&txs).
		Error; err != nil {
		return nil, err
	}

	txsResp := make([]*baseTransaction, len(txs))
	for i, tx := range txs {
		txsResp[i] = newBaseTransaction(tx)
	}

	return &storageContractResp{
		Hash:         hash,
		ObjectHash:   sc.ObjectHash,
		Status:       pbc.StorageStatus_name[sc.Status],
		Fee:          phoAmount(sc.Fee),
		Bond:         phoAmount(sc.Bond),
		Owner:        sc.Owner.PublicKey,
		Depot:        sc.Depot.PublicKey,
		Auditor:      sc.Auditor.PublicKey,
		StartEpoch:   uint64(slots.ToEpoch(pbc.Slot(sc.StartSlot))),
		EndEpoch:     uint64(slots.ToEpoch(pbc.Slot(sc.EndSlot))),
		Transactions: txsResp,
	}, nil
}

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
		Preload("Owner").
		Preload("Depot")

	if pk := c.Query("public_key"); pk != "" {
		query = query.Joins("join accounts on accounts.id = "+
			"storage_contracts.owner_id").
			Where("public_key = ?", pk)
	}

	scs := make([]*orm.StorageContract, 0)
	if err := query.Offset(page.Start).
		Limit(page.Limit).
		Order("id desc").
		Find(&scs).
		Error; err != nil {
		return nil, err
	}

	nextSlot := uint64(0)
	if err := s.db.Model(&orm.ChainStatus{}).
		Where("id = 1").
		Pluck("next_slot", &nextSlot).
		Error; err != nil {
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
			TimeSinceCommit: (nextSlot - sc.StartSlot - 1) * config.Consensus().SecondsPerSlot,
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
