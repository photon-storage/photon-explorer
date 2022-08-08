package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/photon-storage/go-photon/chain/gateway"
)

const (
	chainStatusPath = "chain-status"
	blockPath       = "block"
)

// Node gets the required data according to the HTTP request
// from the photon node.
type Node struct {
	BaseURL string
}

// NewNode returns a new node instance.
func NewNode(url string) *Node {
	return &Node{BaseURL: url}
}

// HeadSlot requests chain head slot.
func (n *Node) HeadSlot(ctx context.Context) (uint64, error) {
	url := fmt.Sprintf("%s/%s", n.BaseURL, chainStatusPath)
	cs := &gateway.ChainStatusResp{}
	if err := httpGet(ctx, url, cs); err != nil {
		return 0, err
	}

	return cs.Best.Slot, nil
}

// BlockBySlot requests chain block by the given slot.
func (n *Node) BlockBySlot(ctx context.Context, slot uint64) (*gateway.BlockResp, error) {
	url := fmt.Sprintf("%s/%s?slot=%d", n.BaseURL, blockPath, slot)
	b := &gateway.BlockResp{}
	return b, httpGet(ctx, url, b)
}

// BlockByHash requests chain block by the given hash.
func (n *Node) BlockByHash(ctx context.Context, hash string) (*gateway.BlockResp, error) {
	url := fmt.Sprintf("%s/%s?hash=%s", n.BaseURL, blockPath, hash)
	b := &gateway.BlockResp{}
	return b, httpGet(ctx, url, b)
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
