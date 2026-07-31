//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func TestSiteDeviceFlow(t *testing.T) {
	baseURL := envOr("TEST_BASE_URL", "http://127.0.0.1:8000")
	token := envOr("TEST_INTERNAL_TOKEN", "change-this-local-token")
	api := apiClient{baseURL: baseURL, token: token, client: &http.Client{Timeout: 10 * time.Second}}

	var cities struct {
		Cities []struct {
			Code string `json:"code"`
		} `json:"cities"`
	}
	api.getJSON(t, "/v1/cities", false, &cities)
	if len(cities.Cities) == 0 || cities.Cities[0].Code == "" {
		t.Fatalf("cities = %+v", cities)
	}

	var sites struct {
		Sites []struct {
			ID string `json:"id"`
		} `json:"sites"`
	}
	api.getJSON(t, "/v1/sites?city_code=310100&latitude=31.2304&longitude=121.4737&radius_m=5000", false, &sites)
	if len(sites.Sites) == 0 {
		t.Fatal("nearby sites is empty")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	api.sendJSON(t, http.MethodPost, "/v1/internal/devices/DEV-SH-001/heartbeat", true, map[string]any{
		"deviceNo": "DEV-SH-001", "firmwareVersion": "integration", "reportedAt": now, "online": true,
	}, http.StatusOK, nil)

	key := fmt.Sprintf("integration-reserve-%d", time.Now().UnixNano())
	reserveBody := map[string]any{"siteId": sites.Sites[0].ID, "size": "CELL_SIZE_MEDIUM", "ttlSeconds": 120, "idempotencyKey": key}
	var firstReserve struct {
		CellID   string `json:"cellId"`
		DeviceNo string `json:"deviceNo"`
		CellNo   string `json:"cellNo"`
	}
	api.sendJSON(t, http.MethodPost, "/v1/internal/cells/reserve", true, reserveBody, http.StatusOK, &firstReserve)
	var replayReserve struct {
		CellID string `json:"cellId"`
	}
	api.sendJSON(t, http.MethodPost, "/v1/internal/cells/reserve", true, reserveBody, http.StatusOK, &replayReserve)
	if firstReserve.CellID == "" || replayReserve.CellID != firstReserve.CellID {
		t.Fatalf("reserve replay = first %+v replay %+v", firstReserve, replayReserve)
	}

	var command struct {
		Status     string `json:"status"`
		DoorOpened bool   `json:"doorOpened"`
		DoorClosed bool   `json:"doorClosed"`
	}
	api.sendJSON(t, http.MethodPost, "/v1/internal/device-commands", true, map[string]any{
		"deviceNo": firstReserve.DeviceNo, "action": "DEVICE_ACTION_OPEN_DOOR", "cellNo": firstReserve.CellNo,
		"ttlSeconds": 30, "idempotencyKey": fmt.Sprintf("integration-command-%d", time.Now().UnixNano()),
	}, http.StatusOK, &command)
	if command.Status != "DEVICE_COMMAND_STATUS_SUCCEEDED" || !command.DoorOpened || !command.DoorClosed {
		t.Fatalf("online command = %+v", command)
	}

	api.sendJSON(t, http.MethodPut, "/v1/internal/simulator/devices/"+firstReserve.DeviceNo+"/scenario", true, map[string]any{
		"deviceNo": firstReserve.DeviceNo, "scenario": "SIMULATOR_SCENARIO_DOOR_LEFT_OPEN",
	}, http.StatusOK, nil)
	api.sendJSON(t, http.MethodPost, "/v1/internal/device-commands", true, map[string]any{
		"deviceNo": firstReserve.DeviceNo, "action": "DEVICE_ACTION_OPEN_DOOR", "cellNo": firstReserve.CellNo,
		"ttlSeconds": 30, "idempotencyKey": fmt.Sprintf("integration-door-open-%d", time.Now().UnixNano()),
	}, http.StatusOK, &command)
	if !command.DoorOpened || command.DoorClosed {
		t.Fatalf("door-left-open command = %+v", command)
	}

	api.sendJSON(t, http.MethodPost, "/v1/internal/cells/reserve", false, reserveBody, http.StatusUnauthorized, nil)
}

func (c apiClient) getJSON(t *testing.T, path string, internal bool, target any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if internal {
		req.Header.Set("X-Internal-Token", c.token)
	}
	c.do(t, req, http.StatusOK, target)
}

func (c apiClient) sendJSON(t *testing.T, method, path string, internal bool, payload any, targetStatus int, target any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if internal {
		req.Header.Set("X-Internal-Token", c.token)
	}
	c.do(t, req, targetStatus, target)
}

func (c apiClient) do(t *testing.T, req *http.Request, wantStatus int, target any) {
	t.Helper()
	response, err := c.client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d body=%s, want=%d", req.Method, req.URL.Path, response.StatusCode, body, wantStatus)
	}
	if target != nil && len(body) > 0 {
		if err := json.Unmarshal(body, target); err != nil {
			t.Fatalf("decode response %s: %v", body, err)
		}
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
