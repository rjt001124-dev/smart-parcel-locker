package biz

import "context"

type GatewayCommand struct {
	DeviceNo string
	Action   Action
	CellNo   string
	Payload  []byte
}

type GatewayResult struct {
	Opened     bool
	DoorClosed bool
	Retryable  bool
}

type Gateway interface {
	Execute(context.Context, GatewayCommand) (GatewayResult, error)
}
