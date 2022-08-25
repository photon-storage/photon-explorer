package orm

import "time"

// Account is a gorm table definition represents the accounts.
type Account struct {
	ID        uint64 `gorm:"primary_key"`
	Address   string
	Nonce     uint64
	Balance   uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}
