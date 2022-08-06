package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/photo-storage/photon-explorer/config"
	"github.com/photo-storage/photon-explorer/database/mysql"
	"github.com/photo-storage/photon-explorer/synchorn"
)

func main() {
	cfg := &config.SyncerConfig{}
	if err := config.LoadConfig(cfg); err != nil {
		log.Fatal("fail on read config", "error", err)
	}

	db, err := mysql.NewMySQLDB(cfg.MySQL)
	if err != nil {
		log.Fatal("initialize mysql db error", "error", err)
	}

	synchorn.NewKeeper(cfg, db).Run()
}
