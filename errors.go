package topo

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrCycleDetected = Error("cycle detected")
)
