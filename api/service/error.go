package service

import "github.com/pkg/errors"

var errSystem = errors.New("system error")

var ErrorCode = map[error]int{
	errSystem: 1000,
}
