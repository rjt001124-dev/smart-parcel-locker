package service

import (
	"context"
	"errors"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
)

type deviceUseCaseStub struct {
	heartbeat biz.DeviceHeartbeat
	execute   biz.CommandRequest
	command   biz.Command
	err       error
}

func (s *deviceUseCaseStub) RecordHeartbeat(_ context.Context, heartbeat biz.DeviceHeartbeat) error {
	s.heartbeat = heartbeat
	return s.err
}

func (s *deviceUseCaseStub) Execute(_ context.Context, req biz.CommandRequest) (biz.Command, error) {
	s.execute = req
	return s.command, s.err
}

func (s *deviceUseCaseStub) GetCommand(context.Context, string) (biz.Command, error) {
	return s.command, s.err
}

func TestHeartbeatParsesUTCAndFirmware(t *testing.T) {
	now := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	uc := &deviceUseCaseStub{}
	service := NewService(uc, nil, true, func() time.Time { return now })
	reply, err := service.Heartbeat(context.Background(), &v1.HeartbeatRequest{DeviceNo: "DEV-1", FirmwareVersion: "1.2.3", ReportedAt: "2026-07-24T19:59:59+08:00", Online: true})
	if err != nil {
		t.Fatal(err)
	}
	if uc.heartbeat.DeviceNo != "DEV-1" || uc.heartbeat.FirmwareVersion != "1.2.3" || !uc.heartbeat.ReportedAt.Equal(time.Date(2026, 7, 24, 11, 59, 59, 0, time.UTC)) || !uc.heartbeat.Online {
		t.Fatalf("heartbeat = %+v", uc.heartbeat)
	}
	if reply.AcceptedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("accepted_at = %q", reply.AcceptedAt)
	}
}

func TestCreateDeviceCommandMapsTTLAndResult(t *testing.T) {
	uc := &deviceUseCaseStub{command: biz.Command{CommandNo: "CMD-1", Status: biz.CommandSucceeded, Result: biz.GatewayResult{Opened: true, DoorClosed: false}}}
	service := NewService(uc, nil, true, nil)
	reply, err := service.CreateDeviceCommand(context.Background(), &v1.CreateDeviceCommandRequest{DeviceNo: "DEV-1", Action: v1.DeviceAction_DEVICE_ACTION_OPEN_DOOR, CellNo: "A01", TtlSeconds: 45, IdempotencyKey: "cmd-key"})
	if err != nil {
		t.Fatal(err)
	}
	if uc.execute.DeviceNo != "DEV-1" || uc.execute.Action != biz.ActionOpenDoor || uc.execute.CellNo != "A01" || uc.execute.TTL != 45*time.Second || uc.execute.IdempotencyKey != "cmd-key" {
		t.Fatalf("command request = %+v", uc.execute)
	}
	if reply.CommandNo != "CMD-1" || reply.Status != v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_SUCCEEDED || !reply.DoorOpened || reply.DoorClosed {
		t.Fatalf("reply = %+v", reply)
	}
}

func TestSetSimulatorScenarioIsHiddenInProduction(t *testing.T) {
	service := NewService(&deviceUseCaseStub{}, nil, false, nil)
	_, err := service.SetSimulatorScenario(context.Background(), &v1.SetSimulatorScenarioRequest{DeviceNo: "DEV-1", Scenario: v1.SimulatorScenario_SIMULATOR_SCENARIO_ONLINE})
	public := kratoserrors.FromError(err)
	if public.Code != 404 || public.Reason != "SIMULATOR_NOT_FOUND" {
		t.Fatalf("error = %+v", public)
	}
}

func TestCreateDeviceCommandMapsSanitizedError(t *testing.T) {
	service := NewService(&deviceUseCaseStub{err: errors.New("secret database dsn")}, nil, true, nil)
	_, err := service.CreateDeviceCommand(context.Background(), &v1.CreateDeviceCommandRequest{DeviceNo: "DEV-1", Action: v1.DeviceAction_DEVICE_ACTION_OPEN_DOOR, IdempotencyKey: "cmd-key"})
	public := kratoserrors.FromError(err)
	if public.Code != 503 || public.Reason != "DEVICE_DATA_UNAVAILABLE" {
		t.Fatalf("error = %+v", public)
	}
}
