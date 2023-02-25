package orm

import (
	"time"

	"gorm.io/gorm"
)

// Block is a gorm table definition represents the blocks.
type Block struct {
	ID                uint64 `gorm:"primary_key"`
	Slot              uint64
	Hash              string
	ParentHash        string
	StateHash         string
	ProposalIndex     uint64
	ProposalSignature string
	RandaoReveal      string
	Graffiti          string
	Timestamp         uint64
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         gorm.DeletedAt
}
