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
	slotRegMatch      = `^\d+$`
	hashRegMatch      = "^[0-9a-fA-F]{64}$"
	publicKeyRegMatch = "^[0-9a-fA-F]{96}$"
)

// QueryType handles the /query request.
func (s *Service) QueryType(c *gin.Context) (*queryResp, error) {
	value := c.Query("value")
	if value == "" {
		return nil, errMissingQueryValue
	}

	queryType := "block"
	switch {
	case regexp.MustCompile(slotRegMatch).MatchString(value):

	case regexp.MustCompile(hashRegMatch).MatchString(value):
		if err := s.db.Model(&orm.Transaction{}).
			Where("hash = ?", value).
			First(nil).
			Error; err == nil {
			queryType = "transaction"
		}

	case regexp.MustCompile(publicKeyRegMatch).MatchString(value):
		_, err := bls.PublicKeyFromHex(value)
		if err != nil {
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
