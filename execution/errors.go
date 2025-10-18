package execution

import "errors"

var (
	// ErrExecutionNotFound is returned when an execution ID doesn't exist
	ErrExecutionNotFound = errors.New("execution not found")

	// ErrExecutionAlreadyRunning is returned when trying to start an already-running execution
	ErrExecutionAlreadyRunning = errors.New("execution already running")

	// ErrExecutionCancelled is returned when an execution is cancelled
	ErrExecutionCancelled = errors.New("execution cancelled")

	// ErrInvalidSpec is returned when the agent specification is invalid
	ErrInvalidSpec = errors.New("invalid agent specification")

	// ErrStorageUnavailable is returned when storage operations fail
	ErrStorageUnavailable = errors.New("storage unavailable")
)
