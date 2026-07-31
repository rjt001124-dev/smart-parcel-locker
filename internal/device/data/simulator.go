package data

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	devicebiz "github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
)

type Scenario string

const (
	ScenarioOnline       Scenario = "ONLINE"
	ScenarioOffline      Scenario = "OFFLINE"
	ScenarioTimeout      Scenario = "TIMEOUT"
	ScenarioFail         Scenario = "FAIL"
	ScenarioDoorLeftOpen Scenario = "DOOR_LEFT_OPEN"
)

var ErrUnknownScenario = errors.New("unknown simulator scenario")

type Simulator struct {
	mu        sync.RWMutex
	scenarios map[string]Scenario
}

func NewSimulator() *Simulator {
	return &Simulator{scenarios: make(map[string]Scenario)}
}

func (s *Simulator) SetScenario(deviceNo string, scenario Scenario) error {
	if !knownScenario(scenario) {
		return fmt.Errorf("%w: %s", ErrUnknownScenario, scenario)
	}
	s.mu.Lock()
	s.scenarios[deviceNo] = scenario
	s.mu.Unlock()
	return nil
}

func (s *Simulator) Execute(ctx context.Context, command devicebiz.GatewayCommand) (devicebiz.GatewayResult, error) {
	s.mu.RLock()
	scenario := s.scenarios[command.DeviceNo]
	s.mu.RUnlock()
	if scenario == "" {
		scenario = ScenarioOnline
	}
	switch scenario {
	case ScenarioOnline:
		return devicebiz.GatewayResult{Opened: command.Action == devicebiz.ActionOpenDoor, DoorClosed: true}, nil
	case ScenarioDoorLeftOpen:
		return devicebiz.GatewayResult{Opened: true, DoorClosed: false}, nil
	case ScenarioOffline:
		return devicebiz.GatewayResult{}, devicebiz.ErrGatewayOffline
	case ScenarioFail:
		return devicebiz.GatewayResult{}, devicebiz.ErrGatewayFailure
	case ScenarioTimeout:
		timer := time.NewTimer(time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return devicebiz.GatewayResult{Retryable: true}, fmt.Errorf("%w: %v", devicebiz.ErrGatewayTimeout, ctx.Err())
		case <-timer.C:
			return devicebiz.GatewayResult{Retryable: true}, devicebiz.ErrGatewayTimeout
		}
	default:
		return devicebiz.GatewayResult{}, ErrUnknownScenario
	}
}

func knownScenario(scenario Scenario) bool {
	switch scenario {
	case ScenarioOnline, ScenarioOffline, ScenarioTimeout, ScenarioFail, ScenarioDoorLeftOpen:
		return true
	default:
		return false
	}
}
