package service

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/photon-storage/photon-explorer/chain"
)

// Service defines an instance of service that handles third-party requests.
type Service struct {
	db   *gorm.DB
	node *chain.NodeClient
}

// New creates a new service instance.
func New(ctx context.Context, db *gorm.DB, nodeEndpoint string) *Service {
	return &Service{
		db:   db,
		node: chain.NewNodeClient(ctx, nodeEndpoint),
	}
}

type pingResp struct {
	Pong string `json:"pong"`
}

func (s *Service) Ping(_ *gin.Context) (*pingResp, error) {
	return &pingResp{Pong: "pong"}, nil
}
