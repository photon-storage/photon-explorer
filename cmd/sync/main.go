package main

import (
	"github.com/photon-storage/go-common/log"

	"github.com/photo-storage/photon-explorer/config"
	"github.com/photo-storage/photon-explorer/database/mysql"
	"github.com/photo-storage/photon-explorer/synchorn"
)

func main() {
	log.Init(log.DebugLevel, log.TextFormat)
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
