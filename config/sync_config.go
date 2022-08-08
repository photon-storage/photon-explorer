package config

import "github.com/photon-storage/photon-explorer/database/mysql"

// SyncerConfig represent sync of config
type SyncerConfig struct {
	MySQL       mysql.Config `json:"mysql"`
	SyncSeconds uint64       `json:"sync_seconds"`
	NodeURL     string       `json:"node_url"`
}
