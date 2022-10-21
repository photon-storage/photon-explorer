package orm

import (
	"time"
)

// StorageContract is a gorm table definition represents
// the storage_contracts.
type StorageContract struct {
	ID                  uint64 `gorm:"primary_key"`
	CommitTransactionID uint64
	OwnerID             uint64
	DepotID             uint64
	AuditorID           uint64 `gorm:"default:null"`
	ObjectHash          string
	Status              int32
	Size                uint64
	Fee                 uint64
	Pledge              uint64
	StartSlot           uint64
	EndSlot             uint64
	CreatedAt           time.Time
	UpdatedAt           time.Time

	Owner             *Account     `gorm:"foreignkey:OwnerID"`
	Depot             *Account     `gorm:"foreignkey:DepotID"`
	Auditor           *Account     `gorm:"foreignkey:AuditorID"`
	CommitTransaction *Transaction `gorm:"foreignkey:CommitTransactionID"`
}
