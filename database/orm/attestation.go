package orm

import "time"

// Attestation is a gorm table definition represents the attestations.
type Attestation struct {
	ID              uint64 `gorm:"primary_key"`
	BlockID         uint64
	CommitteeIndex  uint64
	AggregationBits string
	SourceEpoch     uint64
	SourceHash      string
	TargetEpoch     uint64
	TargetHash      string
	Signature       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
