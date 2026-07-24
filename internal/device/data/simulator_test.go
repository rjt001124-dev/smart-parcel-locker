package data

import (
	"context"
	"errors"
	"testing"
	"time"

	devicebiz "github.com/rjt001124-dev/smart-parcel-locker/internal/device/biz"
)

func TestSimulatorOnlineSuccess(t *testing.T) {
	sim := NewSimulator()
	result, err := sim.Execute(context.Background(), devicebiz.GatewayCommand{DeviceNo: "DEV-001", Action: devicebiz.ActionOpenDoor})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Opened || !result.DoorClosed {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSimulatorOffline(t *testing.T) {
	sim := NewSimulator()
	if err := sim.SetScenario("DEV-001", ScenarioOffline); err != nil {
		t.Fatal(err)
	}
	_, err := sim.Execute(context.Background(), devicebiz.GatewayCommand{DeviceNo: "DEV-001", Action: devicebiz.ActionOpenDoor})
	if !errors.Is(err, devicebiz.ErrGatewayOffline) {
		t.Fatalf("error = %v, want ErrGatewayOffline", err)
	}
}

func TestSimulatorTimeoutHonorsContextDeadline(t *testing.T) {
	sim := NewSimulator()
	if err := sim.SetScenario("DEV-001", ScenarioTimeout); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := sim.Execute(ctx, devicebiz.GatewayCommand{DeviceNo: "DEV-001", Action: devicebiz.ActionOpenDoor})
	if !errors.Is(err, devicebiz.ErrGatewayTimeout) {
		t.Fatalf("error = %v, want ErrGatewayTimeout", err)
	}
	if elapsed := time.Since(started); elapsed > 200*time.Millisecond {
		t.Fatalf("timeout took %s, want it to honor deadline", elapsed)
	}
}

func TestSimulatorFailure(t *testing.T) {
	sim := NewSimulator()
	if err := sim.SetScenario("DEV-001", ScenarioFail); err != nil {
		t.Fatal(err)
	}
	_, err := sim.Execute(context.Background(), devicebiz.GatewayCommand{DeviceNo: "DEV-001", Action: devicebiz.ActionOpenDoor})
	if !errors.Is(err, devicebiz.ErrGatewayFailure) {
		t.Fatalf("error = %v, want ErrGatewayFailure", err)
	}
}

func TestSimulatorDoorLeftOpen(t *testing.T) {
	sim := NewSimulator()
	if err := sim.SetScenario("DEV-001", ScenarioDoorLeftOpen); err != nil {
		t.Fatal(err)
	}
	result, err := sim.Execute(context.Background(), devicebiz.GatewayCommand{DeviceNo: "DEV-001", Action: devicebiz.ActionOpenDoor})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Opened || result.DoorClosed {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestSimulatorRejectsUnknownScenario(t *testing.T) {
	sim := NewSimulator()
	if err := sim.SetScenario("DEV-001", Scenario("UNKNOWN")); !errors.Is(err, ErrUnknownScenario) {
		t.Fatalf("error = %v, want ErrUnknownScenario", err)
	}
}
