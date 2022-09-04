package service

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/photon-storage/photon-explorer/chain"
)

type Service struct {
	db   *gorm.DB
	node *chain.NodeClient
}

func New(db *gorm.DB, nodeEndpoint string) *Service {
	return &Service{
		db:   db,
		node: chain.NewNodeClient(nodeEndpoint),
	}
}

type pingResp struct {
	Pong string `json:"pong"`
}

func (s *Service) Ping(_ *gin.Context) (*pingResp, error) {
	return &pingResp{Pong: "pong"}, nil
}
