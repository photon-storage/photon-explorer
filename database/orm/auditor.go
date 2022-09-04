package orm

import "time"

// Auditor is a gorm table definition represents the auditors.
type Auditor struct {
	ID              uint64 `gorm:"primary_key"`
	AccountID       uint64
	Deposit         uint64
	ActivationEpoch uint64
	ExitEpoch       uint64
	AttestBlockID   uint64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
