package data

import "context"

// DoorSimulator is a development-grade DoorGateway. The milestone build has no
// real device gateway wired in, so doors are reported as opened immediately
// and unconditionally. A production build would route OpenDepositDoor /
// OpenPickupDoor through the device command domain (internal/device) instead.
type DoorSimulator struct{}

var _ interface {
	OpenDepositDoor(context.Context, string) error
	OpenPickupDoor(context.Context, string) error
} = (*DoorSimulator)(nil)

func NewDoorSimulator() *DoorSimulator {
	return &DoorSimulator{}
}

func (s *DoorSimulator) OpenDepositDoor(_ context.Context, _ string) error {
	return nil
}

func (s *DoorSimulator) OpenPickupDoor(_ context.Context, _ string) error {
	return nil
}
