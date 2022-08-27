package orm

import "time"

// TransactionContract is a gorm table definition represents
// the transaction_contracts.
type TransactionContract struct {
	ID            uint64 `gorm:"primary_key"`
	TransactionID uint64
	ContractID    uint64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
