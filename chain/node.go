package chain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/photon-storage/go-photon/crypto/bls"

	"github.com/photon-storage/go-photon/chain/gateway"
)

const (
	chainStatusPath = "chain-status"
	blockPath       = "block"
	account         = "account"
)

var errPublicKey = errors.New("invalid public key string")

// NodeClient gets the required data according to the HTTP request
// from the photon node.
type NodeClient struct {
	endpoint string
}

// NewNodeClient returns a new node instance.
func NewNodeClient(endpoint string) *NodeClient {
	return &NodeClient{endpoint: endpoint}
}

// HeadSlot requests chain head slot.
func (n *NodeClient) HeadSlot(ctx context.Context) (uint64, error) {
	url := fmt.Sprintf("%s/%s", n.endpoint, chainStatusPath)
	cs := &gateway.ChainStatusResp{}
	if err := httpGet(ctx, url, cs); err != nil {
		return 0, err
	}

	return cs.Best.Slot, nil
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

// Account gets account detail by account public key
func (n *NodeClient) Account(ctx context.Context, pk string) (*gateway.AccountResp, error) {
	if !isValidPublicKey(pk) {
		return nil, errPublicKey
	}

	url := fmt.Sprintf("%s/%s?public_key=%s", n.endpoint, account, pk)
	a := &gateway.AccountResp{}
	return a, httpGet(ctx, url, a)
}

func isValidPublicKey(pk string) bool {
	if strings.HasPrefix("0x", strings.ToLower(pk)) {
		pk = pk[2:]
	}
	_, err := hex.DecodeString(pk)
	return len(pk) == bls.PublicKeyLength*2 && err == nil
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
