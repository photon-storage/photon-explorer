package service

import (
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-photon/crypto/bls"

	"github.com/photon-storage/photon-explorer/database/orm"
)

type queryResp struct {
	Type string `json:"type"`
}

var (
	slotReg      = regexp.MustCompile(`^\d+$`)
	hashReg      = regexp.MustCompile("^[0-9a-fA-F]{64}$")
	publicKeyReg = regexp.MustCompile("^[0-9a-fA-F]{96}$")
)

// QueryType handles the /query request.
func (s *Service) QueryType(c *gin.Context) (*queryResp, error) {
	value := c.Query("value")
	if value == "" {
		return nil, errMissingQueryValue
	}

	queryType := "block"
	switch {
	case slotReg.MatchString(value):

	case hashReg.MatchString(value):
		txID := uint64(0)
		if err := s.db.Model(&orm.Transaction{}).
			Where("hash = ?", value).
			Pluck("id", &txID).
			Error; err == nil {
			queryType = "transaction"

			if err := s.db.Model(&orm.StorageContract{}).
				Where("commit_transaction_id = ?", txID).
				First(nil).
				Error; err == nil {
				queryType = "contract"
			}
		}

	case publicKeyReg.MatchString(value):
		if _, err := bls.PublicKeyFromHex(value); err != nil {
			return nil, err
		}

		if err := s.db.Model(&orm.Account{}).
			Where("public_key = ?", value).
			First(nil).
			Error; err == nil {
			queryType = "account"
		}

	default:
		queryType = "unknown"
	}

	return &queryResp{Type: queryType}, nil
}
