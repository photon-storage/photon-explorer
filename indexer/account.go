package indexer

import (
	"gorm.io/gorm"

	"github.com/photon-storage/photon-explorer/database/orm"
)

func updateAccountBalance(db *gorm.DB, pk string, amount int64) error {
	return db.Model(&orm.Account{}).
		Where("public_key = ?", pk).
		Update("balance", gorm.Expr("balance + ?", amount)).
		Error
}

func getAccountIDByPublicKey(dbTx *gorm.DB, pk string) (uint64, error) {
	account := &orm.Account{}
	return account.ID, dbTx.Model(&orm.Account{}).
		Where("public_key = ?", pk).
		First(account).
		Error
}
