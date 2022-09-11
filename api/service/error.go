package service

import "github.com/pkg/errors"

var (
	errSystem        = errors.New("system error")
	errMissingTxHash = errors.New("missing transaction hash")
)

var ErrorCode = map[error]int{
	errSystem:        1000,
	errMissingTxHash: 1001,
}
