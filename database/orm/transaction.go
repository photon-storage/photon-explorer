package orm

import (
	"time"
)

// Transaction is a gorm table definition represents the transactions.
type Transaction struct {
	ID            uint64 `gorm:"primary_key"`
	BlockID       uint64
	Hash          string
	FromPublicKey string
	Position      uint64
	GasPrice      uint64
	Type          int32
	Raw           []byte
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Block *Block `gorm:"foreignkey:BlockID"`
}
