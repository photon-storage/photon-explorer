package orm

import (
	"time"
)

// ChainStatus represents the chain status of gorm table
type ChainStatus struct {
	ID        uint64 `gorm:"primary_key"`
	Slot      uint64
	Hash      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TableName change default table name
func (c ChainStatus) TableName() string {
	return "chain_status"
}
