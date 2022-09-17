package orm

import "time"

// Validator is a gorm table definition represents the validators.
type Validator struct {
	ID              uint64 `gorm:"primary_key"`
	AccountID       uint64
	Index           uint64 `gorm:"column:idx"`
	Deposit         uint64
	Status          int32
	ActivationEpoch uint64
	ExitEpoch       uint64
	AttestBlockID   uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time

	Account     *Account `gorm:"foreignkey:AccountID"`
	AttestBlock *Block   `gorm:"foreignkey:AttestBlockID"`
}
