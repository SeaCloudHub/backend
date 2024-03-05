package common

type State string

const (
	ActiveState   State = "active"
	DeActiveState State = "deactive"
)

var (
	AvailableState = []State{ActiveState, DeActiveState}
)
