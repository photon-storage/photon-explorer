package orm

import (
	"time"
)

// ChainStatus is a gorm table definition represents the chain status.
type ChainStatus struct {
	ID            uint64 `gorm:"primary_key"`
	NextSlot      uint64
	CurrentHash   string
	FinalizedSlot uint64
	FinalizedHash string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TableName the default table name is chain_statuses,
// we change default table name to chain_status.
func (c ChainStatus) TableName() string {
	return "chain_status"
}
