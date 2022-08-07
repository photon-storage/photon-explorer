package chain

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"time"

	"github.com/photon-storage/go-photon/chain/gateway"
)

const (
	chainStatusURL = "chain-status"
	blockURL       = "block"
)

// Node requests chain data according to http
type Node struct {
	BaseURL string
}

// NewNode returns a new node instance
func NewNode(url string) *Node {
	return &Node{BaseURL: url}
}

// HeadSlot requests chain head slot
func (n *Node) HeadSlot() (uint64, error) {
	url := fmt.Sprintf("%s/%s", n.BaseURL, chainStatusURL)
	head := &struct {
		Best struct {
			Slot uint64 `json:"slot"`
		} `json:"best"`
	}{}
	if err := httpGet(url, head); err != nil {
		return 0, err
	}

	return head.Best.Slot, nil
}

// BlockBySlot requests chain block by the given slot
func (n *Node) BlockBySlot(slot uint64) (*gateway.BlockResp, error) {
	url := fmt.Sprintf("%s/%s?slot=%d", n.BaseURL, blockURL, slot)
	b := &gateway.BlockResp{}
	return b, httpGet(url, b)
}

// BlockByHash requests chain block by the given hash
func (n *Node) BlockByHash(hash string) (*gateway.BlockResp, error) {
	url := fmt.Sprintf("%s/%s?hash=%s", n.BaseURL, blockURL, hash)
	b := &gateway.BlockResp{}
	return b, httpGet(url, b)
}

type photonResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

func httpGet(url string, result interface{}) error {
	if reflect.TypeOf(result).Kind() != reflect.Ptr {
		return fmt.Errorf("http result parameter must be pointer interface: %v", result)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
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
		return fmt.Errorf("request photon node failed,err:%s", pr.Msg)
	}

	data, err := json.Marshal(pr.Data)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, result)
}
