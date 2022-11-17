package indexer

import (
	"sort"

	"gorm.io/gorm"

	"github.com/photon-storage/go-photon/config/config"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/database/orm"
)

func processGenesis(dbTx *gorm.DB) error {
	if err := processGenesisBalances(dbTx); err != nil {
		return err
	}

	return processGenesisValidators(dbTx)
}

func processGenesisValidators(dbTx *gorm.DB) error {
	vm := config.Consensus().GenesisConfig.Validators
	var pks []string
	for v := range vm {
		pks = append(pks, v)
	}
	sort.Strings(pks)

	for i, pk := range pks {
		accountID, err := getAccountIDByPublicKey(dbTx, pk)
		if err != nil {
			return err
		}

		v := &orm.Validator{
			AccountID:       accountID,
			Index:           uint64(i),
			Deposit:         vm[pk],
			Status:          int32(pbc.ValidatorStatus_VALIDATOR_ACTIVE),
			ActivationEpoch: 0,
			ExitEpoch:       uint64(config.Consensus().FarFutureEpoch),
		}

		if err := dbTx.Model(&orm.Validator{}).Create(v).Error; err != nil {
			return err
		}
	}

	return nil
}

func processGenesisBalances(dbTx *gorm.DB) error {
	balances := config.Consensus().GenesisConfig.Balances
	var pks []string
	for b := range balances {
		pks = append(pks, b)
	}
	sort.Strings(pks)
	for _, pk := range pks {
		account := &orm.Account{
			PublicKey: pk,
			Balance:   balances[pk],
		}
		if err := dbTx.Model(&orm.Account{}).Create(account).Error; err != nil {
			return err
		}
	}

	return nil
}
