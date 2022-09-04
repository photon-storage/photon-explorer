package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/photon-storage/go-photon/crypto/bls"

	"github.com/photon-storage/go-photon/chain/gateway"
)

const (
	chainStatusPath = "chain-status"
	blockPath       = "block"
	accountPath     = "account"
	storagePath     = "storage"
	validatorsPath  = "validators"
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

// Validators gets validators by pagination params.
func (n *NodeClient) Validators(
	ctx context.Context,
	pageToken int,
	pageSize uint64,
) (*gateway.ValidatorsResp, error) {
	url := fmt.Sprintf(
		"%s/%s?page_token=%d&page_size=%d",
		n.endpoint,
		validatorsPath,
		pageToken,
		pageSize,
	)
	v := &gateway.ValidatorsResp{}
	return v, httpGet(ctx, url, v)
}

// StorageContract gets storage contract detail by tx hash.
func (n *NodeClient) StorageContract(ctx context.Context, hash string) (*gateway.StorageResp, error) {
	url := fmt.Sprintf("%s/%s?hash=%s", n.endpoint, storagePath, hash)
	s := &gateway.StorageResp{}
	return s, httpGet(ctx, url, s)
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
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	pr := &photonResponse{}
	if err := json.Unmarshal(body, pr); err != nil {
		return err
	}

	if pr.Code != http.StatusOK {
		return fmt.Errorf("request photon node failed, err:%s", pr.Msg)
	}

	return json.Unmarshal(pr.Data, result)
}
