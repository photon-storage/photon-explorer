package orm

import "time"

// TransactionType represents transaction type
type TransactionType uint8

const (
	Invalid TransactionType = iota + 1
	BalanceTransfer
	ValidatorDeposit
	ValidatorExit
	AuditorDeposit
	AuditorExit
	ObjectCommit
	ObjectAudit
	ObjectPor
)

var (
	txTypeValue = map[TransactionType]string{
		Invalid:          "TX_INVALID",
		BalanceTransfer:  "BALANCE_TRANSFER",
		ValidatorDeposit: "VALIDATOR_DEPOSIT",
		ValidatorExit:    "VALIDATOR_EXIT",
		AuditorDeposit:   "AUDITOR_DEPOSIT",
		AuditorExit:      "AUDITOR_EXIT",
		ObjectCommit:     "OBJECT_COMMIT",
		ObjectAudit:      "OBJECT_AUDIT",
		ObjectPor:        "OBJECT_POR",
	}

	txValueType = map[string]TransactionType{
		"TX_INVALID":        Invalid,
		"BALANCE_TRANSFER":  BalanceTransfer,
		"VALIDATOR_DEPOSIT": ValidatorDeposit,
		"VALIDATOR_EXIT":    ValidatorExit,
		"AUDITOR_DEPOSIT":   AuditorDeposit,
		"AUDITOR_EXIT":      AuditorExit,
		"OBJECT_COMMIT":     ObjectCommit,
		"OBJECT_AUDIT":      ObjectAudit,
		"OBJECT_POR":        ObjectPor,
	}
)

// StrToType converts type string to transaction type
func StrToType(str string) TransactionType {
	if _, ok := txValueType[str]; !ok {
		return 0
	}

	return txValueType[str]
}

// String returns the string of transaction type
func (t TransactionType) String() string {
	if _, ok := txTypeValue[t]; !ok {
		return "unknown"
	}

	return txTypeValue[t]
}

// Transaction is a gorm table definition represents the transactions.
type Transaction struct {
	ID        uint64 `gorm:"primary_key"`
	BlockID   uint64
	Hash      string
	From      string
	Position  uint64
	GasPrice  uint64
	Type      TransactionType
	Raw       string
	CreatedAt time.Time
	UpdatedAt time.Time
}
