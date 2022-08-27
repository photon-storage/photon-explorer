package orm

import (
	"time"
)

// StorageContract is a gorm table definition represents
// the storage_contracts.
type StorageContract struct {
	ID                  uint64 `gorm:"primary_key"`
	CommitTransactionID uint64
	Owner               uint64
	Provider            uint64
	Auditor             uint64
	ObjectHash          string
	Status              int32
	Size                uint64
	Fee                 uint64
	Bond                uint64
	StartEpoch          uint64
	EndEpoch            uint64
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
