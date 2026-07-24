package service

import (
	"context"
	"errors"
	"strings"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v2/errors"
	v1 "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1"
	"github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
)

type UseCase interface {
	RecordHeartbeat(context.Context, biz.DeviceHeartbeat) error
	Execute(context.Context, biz.CommandRequest) (biz.Command, error)
	GetCommand(context.Context, string) (biz.Command, error)
}

type ScenarioSetter func(deviceNo, scenario string) error

type Service struct {
	uc               UseCase
	setScenario      ScenarioSetter
	simulatorEnabled bool
	now              func() time.Time
}

var _ v1.InternalDeviceServiceHTTPServer = (*Service)(nil)

func NewService(uc UseCase, setter ScenarioSetter, simulatorEnabled bool, now func() time.Time) *Service {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{uc: uc, setScenario: setter, simulatorEnabled: simulatorEnabled, now: now}
}

func (s *Service) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) (*v1.HeartbeatReply, error) {
	if req == nil || strings.TrimSpace(req.DeviceNo) == "" {
		return nil, kratoserrors.BadRequest("INVALID_DEVICE_NO", "invalid device number")
	}
	reportedAt := s.now().UTC()
	if req.ReportedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, req.ReportedAt)
		if err != nil {
			return nil, kratoserrors.BadRequest("INVALID_REPORTED_AT", "invalid reported time")
		}
		reportedAt = parsed.UTC()
	}
	if err := s.uc.RecordHeartbeat(ctx, biz.DeviceHeartbeat{DeviceNo: req.DeviceNo, FirmwareVersion: req.FirmwareVersion, ReportedAt: reportedAt, Online: req.Online}); err != nil {
		return nil, mapDeviceError(err)
	}
	return &v1.HeartbeatReply{AcceptedAt: s.now().UTC().Format(time.RFC3339Nano)}, nil
}

func (s *Service) CreateDeviceCommand(ctx context.Context, req *v1.CreateDeviceCommandRequest) (*v1.DeviceCommandReply, error) {
	if req == nil {
		req = &v1.CreateDeviceCommandRequest{}
	}
	action, err := toAction(req.Action)
	if err != nil {
		return nil, err
	}
	if req.TtlSeconds < 0 {
		return nil, kratoserrors.BadRequest("INVALID_COMMAND_TTL", "invalid command TTL")
	}
	command, executeErr := s.uc.Execute(ctx, biz.CommandRequest{
		DeviceNo:       req.DeviceNo,
		Action:         action,
		CellNo:         req.CellNo,
		TTL:            time.Duration(req.TtlSeconds) * time.Second,
		IdempotencyKey: req.IdempotencyKey,
	})
	if executeErr != nil && command.CommandNo == "" {
		return nil, mapDeviceError(executeErr)
	}
	if executeErr != nil && !errors.Is(executeErr, biz.ErrGatewayTimeout) && !errors.Is(executeErr, biz.ErrGatewayFailure) && !errors.Is(executeErr, context.DeadlineExceeded) && !errors.Is(executeErr, context.Canceled) {
		return nil, mapDeviceError(executeErr)
	}
	return commandReply(command), nil
}

func (s *Service) GetDeviceCommand(ctx context.Context, req *v1.GetDeviceCommandRequest) (*v1.DeviceCommandReply, error) {
	if req == nil || strings.TrimSpace(req.CommandNo) == "" {
		return nil, kratoserrors.BadRequest("INVALID_COMMAND_NO", "invalid command number")
	}
	command, err := s.uc.GetCommand(ctx, req.CommandNo)
	if err != nil {
		return nil, mapDeviceError(err)
	}
	return commandReply(command), nil
}

func (s *Service) SetSimulatorScenario(_ context.Context, req *v1.SetSimulatorScenarioRequest) (*v1.SetSimulatorScenarioReply, error) {
	if !s.simulatorEnabled {
		return nil, kratoserrors.NotFound("SIMULATOR_NOT_FOUND", "simulator route not found")
	}
	if req == nil || strings.TrimSpace(req.DeviceNo) == "" {
		return nil, kratoserrors.BadRequest("INVALID_DEVICE_NO", "invalid device number")
	}
	scenario, err := toScenario(req.Scenario)
	if err != nil {
		return nil, err
	}
	if s.setScenario == nil {
		return nil, kratoserrors.ServiceUnavailable("SIMULATOR_UNAVAILABLE", "simulator unavailable")
	}
	if err := s.setScenario(req.DeviceNo, scenario); err != nil {
		return nil, kratoserrors.BadRequest("INVALID_SIMULATOR_SCENARIO", "invalid simulator scenario")
	}
	return &v1.SetSimulatorScenarioReply{Updated: true}, nil
}

func toAction(action v1.DeviceAction) (biz.Action, error) {
	switch action {
	case v1.DeviceAction_DEVICE_ACTION_OPEN_DOOR:
		return biz.ActionOpenDoor, nil
	case v1.DeviceAction_DEVICE_ACTION_QUERY_STATUS:
		return biz.ActionQueryStatus, nil
	default:
		return "", kratoserrors.BadRequest("INVALID_DEVICE_ACTION", "invalid device action")
	}
}

func toScenario(scenario v1.SimulatorScenario) (string, error) {
	switch scenario {
	case v1.SimulatorScenario_SIMULATOR_SCENARIO_ONLINE:
		return "ONLINE", nil
	case v1.SimulatorScenario_SIMULATOR_SCENARIO_OFFLINE:
		return "OFFLINE", nil
	case v1.SimulatorScenario_SIMULATOR_SCENARIO_TIMEOUT:
		return "TIMEOUT", nil
	case v1.SimulatorScenario_SIMULATOR_SCENARIO_FAIL:
		return "FAIL", nil
	case v1.SimulatorScenario_SIMULATOR_SCENARIO_DOOR_LEFT_OPEN:
		return "DOOR_LEFT_OPEN", nil
	default:
		return "", kratoserrors.BadRequest("INVALID_SIMULATOR_SCENARIO", "invalid simulator scenario")
	}
}

func commandReply(command biz.Command) *v1.DeviceCommandReply {
	return &v1.DeviceCommandReply{
		CommandNo:  command.CommandNo,
		Status:     fromStatus(command.Status),
		Retryable:  command.Result.Retryable,
		DoorOpened: command.Result.Opened,
		DoorClosed: command.Result.DoorClosed,
		ErrorCode:  command.ErrorCode,
	}
}

func fromStatus(status biz.CommandStatus) v1.DeviceCommandStatus {
	switch status {
	case biz.CommandPending:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_PENDING
	case biz.CommandRunning:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_RUNNING
	case biz.CommandSucceeded:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_SUCCEEDED
	case biz.CommandFailed:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_FAILED
	case biz.CommandTimedOut:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_TIMED_OUT
	case biz.CommandExpired:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_EXPIRED
	default:
		return v1.DeviceCommandStatus_DEVICE_COMMAND_STATUS_UNSPECIFIED
	}
}

func mapDeviceError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return kratoserrors.ClientClosed("REQUEST_CANCELED", "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return kratoserrors.GatewayTimeout("REQUEST_DEADLINE_EXCEEDED", "request deadline exceeded")
	case errors.Is(err, biz.ErrDeviceNotFound):
		return kratoserrors.NotFound("DEVICE_NOT_FOUND", "device not found")
	case errors.Is(err, biz.ErrCommandNotFound):
		return kratoserrors.NotFound("DEVICE_COMMAND_NOT_FOUND", "device command not found")
	case errors.Is(err, biz.ErrDeviceOffline):
		return kratoserrors.ServiceUnavailable("DEVICE_OFFLINE", "device offline")
	case errors.Is(err, biz.ErrDeviceMaintenance):
		return kratoserrors.Conflict("DEVICE_MAINTENANCE", "device in maintenance")
	case errors.Is(err, biz.ErrDeviceDisabled):
		return kratoserrors.Conflict("DEVICE_DISABLED", "device disabled")
	case errors.Is(err, biz.ErrInvalidAction):
		return kratoserrors.BadRequest("INVALID_DEVICE_ACTION", "invalid device action")
	case errors.Is(err, biz.ErrIdempotencyRequired):
		return kratoserrors.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "idempotency key required")
	case errors.Is(err, biz.ErrIdempotencyConflict):
		return kratoserrors.Conflict("IDEMPOTENCY_CONFLICT", "idempotency conflict")
	case errors.Is(err, biz.ErrCommandExpired):
		return kratoserrors.Conflict("COMMAND_EXPIRED", "device command expired")
	case errors.Is(err, biz.ErrMaxAttempts):
		return kratoserrors.Conflict("COMMAND_MAX_ATTEMPTS", "device command retry limit reached")
	default:
		return kratoserrors.ServiceUnavailable("DEVICE_DATA_UNAVAILABLE", "device data unavailable")
	}
}
