package service

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/config/config"
	"github.com/photon-storage/go-photon/crypto/bls"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/api/pagination"
	"github.com/photon-storage/photon-explorer/database/orm"
)

type accountResp struct {
	PublicKey string  `json:"public_key"`
	Balance   string  `json:"balance"`
	Nonce     uint64  `json:"nonce"`
	Validator *status `json:"validator,omitempty"`
	Auditor   *status `json:"auditor,omitempty"`
}

type status struct {
	Balance         string `json:"balance"`
	ActivationEpoch uint64 `json:"activation_epoch"`
	ExitEpoch       uint64 `json:"exit_epoch"`
}

// Account handles the /account request.
func (s *Service) Account(c *gin.Context) (*accountResp, error) {
	pk := c.Query("public_key")
	if pk == "" {
		return nil, errMissingPublicKey
	}

	if _, err := bls.PublicKeyFromHex(pk); err != nil {
		return nil, err
	}

	account := &orm.Account{}
	if err := s.db.Model(&orm.Account{}).
		Where("public_key = ?", pk).
		First(account).
		Error; err != nil {
		return nil, err
	}

	resp := &accountResp{
		PublicKey: pk,
		Balance:   phoAmount(account.Balance),
		Nonce:     account.Nonce,
	}

	validator := &orm.Validator{}
	if err := s.db.Model(&orm.Validator{}).
		Where("account_id = ?", account.ID).
		First(validator).
		Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err == nil {
		resp.Validator = &status{
			Balance:         phoAmount(validator.Deposit),
			ActivationEpoch: validator.ActivationEpoch,
			ExitEpoch:       convertExitEpoch(validator.ExitEpoch),
		}
	}

	auditor := &orm.Auditor{}
	if err := s.db.Model(&orm.Auditor{}).
		Where("account_id = ?", account.ID).
		First(auditor).
		Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	} else if err == nil {
		resp.Auditor = &status{
			Balance:         phoAmount(auditor.Deposit),
			ActivationEpoch: auditor.ActivationEpoch,
			ExitEpoch:       convertExitEpoch(auditor.ExitEpoch),
		}
	}

	return resp, nil
}

type baseAccount struct {
	PublicKey       string `json:"public_key"`
	DepotAmount     string `json:"depot_amount"`
	Status          string `json:"status"`
	ActivationEpoch uint64 `json:"activation_epoch"`
	ExitEpoch       uint64 `json:"exit_epoch"`
}

type validator struct {
	baseAccount
	LatestAttestation uint64 `json:"latest_attestation"`
}

// Validators handles the /validators request.
func (s *Service) Validators(
	_ *gin.Context,
	page *pagination.Query,
) (*pagination.Result, error) {
	vs := make([]*orm.Validator, 0)
	if err := s.db.Model(&orm.Validator{}).
		Preload("Account").
		Preload("AttestBlock").
		Offset(page.Start).
		Limit(page.Limit).
		Find(&vs).
		Error; err != nil {
		return nil, err
	}

	validators := make([]*validator, len(vs))
	for i, v := range vs {
		timestamp := uint64(0)
		if v.AttestBlock != nil {
			timestamp = v.AttestBlock.Timestamp
		}

		validators[i] = &validator{
			baseAccount: baseAccount{
				PublicKey:       v.Account.PublicKey,
				DepotAmount:     phoAmount(v.Deposit),
				Status:          pbc.ValidatorStatus_name[v.Status],
				ActivationEpoch: v.ActivationEpoch,
				ExitEpoch:       convertExitEpoch(v.ExitEpoch),
			},
			LatestAttestation: timestamp,
		}
	}

	count := int64(0)
	if err := s.db.Model(&orm.Validator{}).Count(&count).Error; err != nil {
		return nil, err
	}

	return &pagination.Result{
		Data:  validators,
		Total: count,
	}, nil
}

type auditor struct {
	baseAccount
	LatestAudit string `json:"latest_audit"`
}

// Auditors handles the /auditors request.
func (s *Service) Auditors(
	_ *gin.Context,
	page *pagination.Query,
) (*pagination.Result, error) {
	as := make([]*orm.Auditor, 0)
	if err := s.db.Model(&orm.Auditor{}).
		Preload("Account").
		Offset(page.Start).
		Limit(page.Limit).
		Find(&as).
		Error; err != nil {
		return nil, err
	}

	auditors := make([]*auditor, len(as))
	for i, a := range as {
		auditHash := ""
		if err := s.db.Model(&orm.Transaction{}).
			Joins("join storage_contracts as sc on sc.commit_transaction_id = transactions.id").
			Where("sc.auditor_id = ?", a.AccountID).
			Order("sc.id desc").
			Limit(1).
			Pluck("transactions.hash", &auditHash).
			Error; err != nil {
			return nil, err
		}

		auditors[i] = &auditor{
			baseAccount: baseAccount{
				PublicKey:       a.Account.PublicKey,
				DepotAmount:     phoAmount(a.Deposit),
				Status:          pbc.AuditorStatus_name[a.Status],
				ActivationEpoch: a.ActivationEpoch,
				ExitEpoch:       convertExitEpoch(a.ExitEpoch),
			},
			LatestAudit: auditHash,
		}
	}

	count := int64(0)
	if err := s.db.Model(&orm.Auditor{}).Count(&count).Error; err != nil {
		return nil, err
	}

	return &pagination.Result{
		Data:  auditors,
		Total: count,
	}, nil
}

func convertExitEpoch(epoch uint64) uint64 {
	if epoch == uint64(config.Consensus().FarFutureEpoch) {
		return 0
	}

	return epoch
}
