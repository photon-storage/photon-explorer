package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/photon-storage/go-photon/crypto/bls"

	"github.com/photon-storage/go-photon/chain/gateway"
)

const (
	chainStatusPath     = "chain-status"
	blockPath           = "block"
	accountPath         = "account"
	storageContractPath = "storage-contract"
	validatorPath       = "validator"
	validatorsPath      = "validators"
	auditorPath         = "auditor"
	auditorsPath        = "auditors"
	committeesPath      = "committees"
)

// NodeClient gets the required data according to the HTTP request
// from the photon node.
type NodeClient struct {
	endpoint string
}

// NewNodeClient returns a new node instance.
func NewNodeClient(endpoint string) *NodeClient {
	return &NodeClient{endpoint: endpoint}
}

// ChainStatus requests chain status of photon node.
func (n *NodeClient) ChainStatus(ctx context.Context) (*gateway.ChainStatusResp, error) {
	url := fmt.Sprintf("%s/%s", n.endpoint, chainStatusPath)
	cs := &gateway.ChainStatusResp{}
	if err := httpGet(ctx, url, cs); err != nil {
		return nil, err
	}

	return cs, nil
}

// BlockBySlot requests chain block by the given slot.
func (n *NodeClient) BlockBySlot(ctx context.Context, slot uint64) (*gateway.BlockResp, error) {
	url := fmt.Sprintf("%s/%s?slot=%d", n.endpoint, blockPath, slot)
	b := &gateway.BlockResp{}
	return b, httpGet(ctx, url, b)
}

// BlockByHash requests chain block by the given hash.
func (n *NodeClient) BlockByHash(ctx context.Context, hash string) (*gateway.BlockResp, error) {
	url := fmt.Sprintf("%s/%s?hash=%s", n.endpoint, blockPath, hash)
	b := &gateway.BlockResp{}
	return b, httpGet(ctx, url, b)
}

// Account gets account detail by account public key.
func (n *NodeClient) Account(ctx context.Context, pk string) (*gateway.AccountResp, error) {
	if _, err := bls.PublicKeyFromHex(strings.ToLower(pk)); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/%s?public_key=%s", n.endpoint, accountPath, pk)
	a := &gateway.AccountResp{}
	return a, httpGet(ctx, url, a)
}

// Validator gets validator by account public key.
func (n *NodeClient) Validator(
	ctx context.Context,
	pk string,
) (*gateway.ValidatorResp, error) {
	url := fmt.Sprintf(
		"%s/%s?public_key=%s",
		n.endpoint,
		validatorPath,
		pk,
	)
	v := &gateway.ValidatorResp{}
	return v, httpGet(ctx, url, v)
}

// Validators gets validators by pagination params.
func (n *NodeClient) Validators(
	ctx context.Context,
	pageToken string,
	pageSize uint64,
) (*gateway.ValidatorsResp, error) {
	url := fmt.Sprintf(
		"%s/%s?page_token=%s&page_size=%d",
		n.endpoint,
		validatorsPath,
		pageToken,
		pageSize,
	)
	v := &gateway.ValidatorsResp{}
	return v, httpGet(ctx, url, v)
}

// Auditor gets auditor by account public key.
func (n *NodeClient) Auditor(
	ctx context.Context,
	pk string,
) (*gateway.ValidatorResp, error) {
	url := fmt.Sprintf(
		"%s/%s?public_key=%s",
		n.endpoint,
		auditorPath,
		pk,
	)
	v := &gateway.ValidatorResp{}
	return v, httpGet(ctx, url, v)
}

// Auditors gets auditors by pagination params.
func (n *NodeClient) Auditors(
	ctx context.Context,
	pageToken string,
	pageSize uint64,
) (*gateway.AuditorsResp, error) {
	url := fmt.Sprintf(
		"%s/%s?page_token=%s&page_size=%d",
		n.endpoint,
		auditorsPath,
		pageToken,
		pageSize,
	)
	a := &gateway.AuditorsResp{}
	return a, httpGet(ctx, url, a)
}

// StorageContract gets storage contract detail by tx hash.
func (n *NodeClient) StorageContract(
	ctx context.Context,
	txHash string,
	blockHash string,
) (*gateway.StorageResp, error) {
	url := fmt.Sprintf(
		"%s/%s?storage_hash=%s&block_hash=%s",
		n.endpoint,
		storageContractPath,
		txHash,
		blockHash,
	)
	s := &gateway.StorageResp{}
	return s, httpGet(ctx, url, s)
}

// TODO(doris): set committees response exported in go-photon repo.
type Committee struct {
	CommitteeIndex   uint64
	ValidatorIndexes []uint64
}

// Committees returns committees info by the given slot.
func (n *NodeClient) Committees(
	ctx context.Context,
	slot uint64,
) ([]*Committee, error) {
	url := fmt.Sprintf(
		"%s/%s?slot=%d",
		n.endpoint,
		committeesPath,
		slot,
	)
	c := []*Committee{}
	return c, httpGet(ctx, url, &c)
}

type photonResponse struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data,omitempty"`
}

func httpGet(ctx context.Context, url string, result interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pr := &photonResponse{}
	if err := json.Unmarshal(body, pr); err != nil {
		return err
	}

	if pr.Code != http.StatusOK {
		return fmt.Errorf("request photon node failed, err: %s", pr.Msg)
	}

	return json.Unmarshal(pr.Data, result)
}
