package service

import "github.com/pkg/errors"

var (
	errSystem              = errors.New("system error")
	errMissingTxHash       = errors.New("missing transaction hash")
	errMissingQueryValue   = errors.New("missing query type value")
	errMissingPublicKey    = errors.New("missing public key")
	errMissingContractHash = errors.New("missing contract hash")
	errApiNotReady         = errors.New("api isn't ready")
)

var ErrorCode = map[error]int{
	errSystem:              1000,
	errMissingTxHash:       1001,
	errMissingQueryValue:   1002,
	errMissingPublicKey:    1003,
	errMissingContractHash: 1004,
	errApiNotReady:         1005,
}
