package orm

import (
	"time"

	"gorm.io/gorm"
)

// Transaction is a gorm table definition represents the transactions.
type Transaction struct {
	ID            uint64 `gorm:"primary_key"`
	BlockID       uint64
	Hash          string
	FromAccountID uint64
	Position      uint64
	GasPrice      uint64
	Type          int32
	Raw           []byte
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt

	Block       *Block   `gorm:"foreignkey:BlockID"`
	FromAccount *Account `gorm:"foreignkey:FromAccountID"`
}
