package server

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-common/log"

	"github.com/photon-storage/photon-explorer/api/service"
)

// Server defines an instance of a server that handles the requests of
// the third-party application.
type Server struct {
	port   int
	engine *gin.Engine
}

// New returns a new instance of the server.
func New(port int, service *service.Service) *Server {
	server := &Server{
		port:   port,
		engine: gin.Default(),
	}

	server.registerRouter(service)
	return server
}

func (s *Server) registerRouter(service *service.Service) {
	s.engine.Use(handleError())
	g := s.engine.Group("photon/v1")

	g.GET("ping", s.handle(service.Ping))
	g.GET("stats", s.handle(service.Stats))
	g.GET("storage-contract", s.handle(service.StorageContract))
	g.GET("storage-contracts", s.handle(service.StorageContracts))
	g.GET("transaction", s.handle(service.Transaction))
	g.GET("transactions", s.handle(service.Transactions))
	g.GET("query", s.handle(service.QueryType))
	g.GET("latest-blocks", s.handle(service.LatestBlocks))
	g.GET("block", s.handle(service.Block))
	g.GET("account", s.handle(service.Account))
	g.GET("validators", s.handle(service.Validators))
}

// Run the server
func (s *Server) Run() {
	if err := s.engine.Run(fmt.Sprintf(":%d", s.port)); err != nil {
		log.Error("run the server failed", "error", err)
		os.Exit(1)
	}
}
