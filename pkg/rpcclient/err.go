package rpcclient

import "errors"

type RpsError struct {
	err error
}

func (e *RpsError) Error() string {
	return e.err.Error()
}

func NewRpsError(msg string) *RpsError {
	return &RpsError{err: errors.New(msg)}
}
