package worker

import "fmt"

type ErrPermanent struct{ Err error }

func (e *ErrPermanent) Error() string { return fmt.Sprintf("permanent: %v", e.Err) }
func (e *ErrPermanent) Unwrap() error { return e.Err }

type ErrRetryable struct{ Err error }

func (e *ErrRetryable) Error() string { return fmt.Sprintf("retryable: %v", e.Err) }
func (e *ErrRetryable) Unwrap() error { return e.Err }

func Permanent(err error) error { return &ErrPermanent{Err: err} }
func Retryable(err error) error { return &ErrRetryable{Err: err} }
