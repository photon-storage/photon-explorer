package service

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/crypto/bls"

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
			ExitEpoch:       validator.ExitEpoch,
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
			ExitEpoch:       auditor.ExitEpoch,
		}
	}

	return resp, nil
}
