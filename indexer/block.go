package indexer

import "github.com/photon-storage/go-photon/chain/gateway"

func (e *EventProcessor) processBlock(block *gateway.BlockResp) error {
	// TODO(doris)
	e.currentSlot = block.Slot
	e.currentHash = block.BlockHash
	return nil
}

func (e *EventProcessor) rollbackBlock(block *gateway.BlockResp) error {
	// TODO(doris)
	return nil
}
