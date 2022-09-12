package service

import "github.com/pkg/errors"

var (
	errSystem            = errors.New("system error")
	errMissingTxHash     = errors.New("missing transaction hash")
	errMissingQueryValue = errors.New("missing query type value")
)

var ErrorCode = map[error]int{
	errSystem:            1000,
	errMissingTxHash:     1001,
	errMissingQueryValue: 1002,
}
