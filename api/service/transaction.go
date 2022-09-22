package service

import (
	"encoding/json"
	"fmt"

	"github.com/docker/go-units"
	"github.com/gin-gonic/gin"

	"github.com/photon-storage/go-photon/chain/gateway"
	pbc "github.com/photon-storage/photon-proto/consensus"

	"github.com/photon-storage/photon-explorer/api/pagination"
	"github.com/photon-storage/photon-explorer/database/orm"
)

type baseTransaction struct {
	Hash      string `json:"hash"`
	From      string `json:"from"`
	Slot      uint64 `json:"slot"`
	Type      string `json:"type"`
	Timestamp uint64 `json:"timestamp"`
	GasPrice  uint64 `json:"gas_price"`
}

// Transactions handles the /transactions request.
func (s *Service) Transactions(
	c *gin.Context,
	page *pagination.Query,
) (*pagination.Result, error) {
	query := s.db.Model(&orm.Transaction{}).
		Preload("Block")
	if pk := c.Query("public_key"); pk != "" {
		query = query.Where("from_public_key = ?", pk)
	}

	if bh := c.Query("block_hash"); bh != "" {
		query = query.Joins("join blocks on blocks.id = transactions.block_id").
			Where("blocks.hash = ?", bh)
	}

	count := int64(0)
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}

	txs := make([]*orm.Transaction, 0)
	if err := query.Offset(page.Start).
		Limit(page.Limit).
		Order("id desc").
		Find(&txs).
		Error; err != nil {
		return nil, err
	}

	baseTransactions := make([]*baseTransaction, len(txs))
	for i, tx := range txs {
		baseTransactions[i] = newBaseTransaction(tx)
	}

	return &pagination.Result{
		Data:  baseTransactions,
		Total: count,
	}, nil
}

func newBaseTransaction(tx *orm.Transaction) *baseTransaction {
	return &baseTransaction{
		Hash:      tx.Hash,
		From:      tx.FromPublicKey,
		Slot:      tx.Block.Slot,
		Type:      pbc.TxType_name[tx.Type],
		Timestamp: tx.Block.Timestamp,
		GasPrice:  tx.GasPrice,
	}
}

type transactionResp struct {
	*baseTransaction
	Position         uint64           `json:"position"`
	Finalized        bool             `json:"finalized"`
	BalanceTransfer  *BalanceTransfer `json:"balance_transfer,omitempty"`
	ValidatorDeposit *Deposit         `json:"validator_deposit,omitempty"`
	AuditorDeposit   *Deposit         `json:"auditor_deposit,omitempty"`
	ObjectCommit     *ObjectCommit    `json:"object_commit,omitempty"`
	ObjectAudit      *ObjectAudit     `json:"object_audit,omitempty"`
	ObjectPoR        *ObjectPoR       `json:"object_por,omitempty"`
}

// BalanceTransfer is the JSON representation of the BalanceTransfer tx.
type BalanceTransfer struct {
	To     string `json:"to"`
	Amount string `json:"amount"`
}

// Deposit is the JSON representation of Validator/Auditor Deposit tx.
type Deposit struct {
	Amount string `json:"amount"`
}

// ObjectCommit is the JSON representation of the ObjectCommit tx.
type ObjectCommit struct {
	Owner    string `json:"owner"`
	Depot    string `json:"depot"`
	Hash     string `json:"hash"`
	Size     string `json:"size"`
	Duration uint64 `json:"duration"`
	Fee      string `json:"fee"`
	Bond     string `json:"bond"`
}

// ObjectAudit is the JSON representation of the ObjectAudit tx.
type ObjectAudit struct {
	CommitTxHash string `json:"commit_tx_hash"`
	Auditor      string `json:"auditor"`
	Depot        string `json:"depot"`
	Hash         string `json:"hash"`
	Size         string `json:"size"`
	EncodedHash  string `json:"encoded_hash"`
	EncodedSize  string `json:"encoded_size"`
	NumBlocks    uint32 `json:"num_blocks"`
}

// ObjectPoR is the JSON representation of the ObjectPoR tx.
type ObjectPoR struct {
	CommitTxHash string   `json:"commit_tx_hash"`
	BlockAggs    []string `json:"block_aggs"`
	Sigma        string   `json:"sigma"`
}

// Transaction handles the /transaction request.
func (s *Service) Transaction(c *gin.Context) (*transactionResp, error) {
	hash := c.Query("hash")
	if hash == "" {
		return nil, errMissingTxHash
	}

	tx := &orm.Transaction{}
	if err := s.db.Model(&orm.Transaction{}).
		Preload("Block").
		Where("hash = ?", hash).
		First(tx).
		Error; err != nil {
		return nil, err
	}

	finalizedSlot := uint64(0)
	if err := s.db.Model(&orm.ChainStatus{}).
		Where("id = 1").
		Pluck("finalized_slot", &finalizedSlot).
		Error; err != nil {
		return nil, err
	}

	resp := &transactionResp{
		baseTransaction: newBaseTransaction(tx),
		Position:        tx.Position,
		Finalized:       tx.Block.Slot <= finalizedSlot,
	}

	return resp, txDetail(tx.Raw, resp)
}

func txDetail(rawTxBytes []byte, resp *transactionResp) error {
	rawTx := &gateway.Tx{}
	if err := json.Unmarshal(rawTxBytes, rawTx); err != nil {
		return err
	}

	switch pbc.TxType_value[rawTx.Type] {
	case int32(pbc.TxType_BALANCE_TRANSFER):
		resp.BalanceTransfer = &BalanceTransfer{
			To:     rawTx.BalanceTransfer.To,
			Amount: phoAmount(rawTx.BalanceTransfer.Amount),
		}

	case int32(pbc.TxType_VALIDATOR_DEPOSIT):
		resp.ValidatorDeposit = &Deposit{
			Amount: phoAmount(rawTx.ValidatorDeposit.Amount),
		}

	case int32(pbc.TxType_AUDITOR_DEPOSIT):
		resp.AuditorDeposit = &Deposit{
			Amount: phoAmount(rawTx.AuditorDeposit.Amount),
		}

	case int32(pbc.TxType_OBJECT_COMMIT):
		oc := rawTx.ObjectCommit
		resp.ObjectCommit = &ObjectCommit{
			Owner:    oc.Owner,
			Depot:    oc.Depot,
			Hash:     oc.Hash,
			Size:     units.HumanSize(float64(oc.Size)),
			Duration: oc.Duration,
			Fee:      phoAmount(oc.Fee),
			Bond:     phoAmount(oc.Bond),
		}

	case int32(pbc.TxType_OBJECT_AUDIT):
		oa := rawTx.ObjectAudit
		resp.ObjectAudit = &ObjectAudit{
			CommitTxHash: oa.CommitTxHash,
			Auditor:      oa.Auditor,
			Depot:        oa.Depot,
			Hash:         oa.Hash,
			Size:         units.HumanSize(float64(oa.Size)),
			EncodedHash:  oa.EncodedHash,
			EncodedSize:  units.HumanSize(float64(oa.EncodedSize)),
			NumBlocks:    oa.NumBlocks,
		}

	case int32(pbc.TxType_OBJECT_POR):
		op := rawTx.ObjectPoR
		resp.ObjectPoR = &ObjectPoR{
			CommitTxHash: op.CommitTxHash,
			BlockAggs:    op.BlockAggs,
			Sigma:        op.Sigma,
		}
	}

	return nil
}

func phoAmount(amount uint64) string {
	// Note: pho decimal will get from photon node
	return fmt.Sprintf("%.2f", float64(amount)/float64(1<<9))
}
