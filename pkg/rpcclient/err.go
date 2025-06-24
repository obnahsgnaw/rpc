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

type CustomError struct {
	err error
}

func (e *CustomError) Error() string {
	return e.err.Error()
}

func (e *CustomError) Raw() error {
	return e.err
}

func NewCustomError(err error) *CustomError {
	return &CustomError{err: err}
}
