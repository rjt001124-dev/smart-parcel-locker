package conf

import (
	"strings"
	"testing"
	"time"
)

func TestLoadUsesDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTPAddr != "0.0.0.0:8000" {
		t.Errorf("HTTPAddr = %q, want %q", cfg.HTTPAddr, "0.0.0.0:8000")
	}
	if cfg.AppEnv != "development" {
		t.Errorf("AppEnv = %q, want %q", cfg.AppEnv, "development")
	}
	if cfg.MySQL.Port != 3306 {
		t.Errorf("MySQL.Port = %d, want %d", cfg.MySQL.Port, 3306)
	}
	if cfg.Redis.Port != 6379 {
		t.Errorf("Redis.Port = %d, want %d", cfg.Redis.Port, 6379)
	}
	if cfg.DeviceOfflineThreshold != 30*time.Second {
		t.Errorf("DeviceOfflineThreshold = %s, want %s", cfg.DeviceOfflineThreshold, 30*time.Second)
	}
}

func TestLoadUsesConfiguredValues(t *testing.T) {
	env := map[string]string{
		"APP_ENV":                  "test",
		"HTTP_ADDR":                "127.0.0.1:9000",
		"MYSQL_HOST":               "mysql",
		"MYSQL_PORT":               "3307",
		"MYSQL_DATABASE":           "locker_test",
		"MYSQL_USER":               "locker",
		"MYSQL_PASSWORD":           "local-only",
		"REDIS_HOST":               "redis",
		"REDIS_PORT":               "6380",
		"REDIS_USERNAME":           "redis-user",
		"REDIS_PASSWORD":           "redis-local-only",
		"REDIS_DATABASE":           "2",
		"INTERNAL_API_TOKEN":       "integration-token",
		"DEVICE_OFFLINE_THRESHOLD": "45s",
	}

	cfg, err := Load(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := Config{
		HTTPAddr:               "127.0.0.1:9000",
		AppEnv:                 "test",
		InternalAPIToken:       "integration-token",
		DeviceOfflineThreshold: 45 * time.Second,
		MySQL: MySQLConfig{
			Host:     "mysql",
			Port:     3307,
			Database: "locker_test",
			User:     "locker",
			Password: "local-only",
		},
		Redis: RedisConfig{
			Host:     "redis",
			Port:     6380,
			Username: "redis-user",
			Password: "redis-local-only",
			Database: 2,
		},
	}
	if cfg != want {
		t.Fatal("Load() did not load configured values exactly")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "MySQL port", key: "MYSQL_PORT", value: "not-a-number"},
		{name: "Redis port", key: "REDIS_PORT", value: "not-a-number"},
		{name: "Redis database", key: "REDIS_DATABASE", value: "not-a-number"},
		{name: "device offline threshold", key: "DEVICE_OFFLINE_THRESHOLD", value: "not-a-duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(func(key string) string {
				if key == tt.key {
					return tt.value
				}
				return ""
			})
			if err == nil {
				t.Fatalf("Load() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("Load() error = %q, want it to name %s", err, tt.key)
			}
		})
	}
}

func TestLoadRedactsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "MySQL port", key: "MYSQL_PORT", value: "leak-marker-int"},
		{name: "Redis database", key: "REDIS_DATABASE", value: "leak-marker-db"},
		{name: "device offline threshold", key: "DEVICE_OFFLINE_THRESHOLD", value: "leak-marker-duration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(func(key string) string {
				if key == tt.key {
					return tt.value
				}
				return ""
			})
			if err == nil {
				t.Fatal("Load() error = nil, want an error")
			}

			message := err.Error()
			if !strings.Contains(message, tt.key) {
				t.Fatalf("Load() error does not name %s", tt.key)
			}
			if strings.Contains(message, tt.value) {
				t.Fatalf("Load() error disclosed the raw value for %s", tt.key)
			}
		})
	}
}

func TestLoadRejectsOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "zero MySQL port", key: "MYSQL_PORT", value: "0"},
		{name: "negative MySQL port", key: "MYSQL_PORT", value: "-1"},
		{name: "MySQL port above maximum", key: "MYSQL_PORT", value: "65536"},
		{name: "zero Redis port", key: "REDIS_PORT", value: "0"},
		{name: "negative Redis port", key: "REDIS_PORT", value: "-1"},
		{name: "Redis port above maximum", key: "REDIS_PORT", value: "65536"},
		{name: "negative Redis database", key: "REDIS_DATABASE", value: "-1"},
		{name: "zero device offline threshold", key: "DEVICE_OFFLINE_THRESHOLD", value: "0s"},
		{name: "negative device offline threshold", key: "DEVICE_OFFLINE_THRESHOLD", value: "-1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(func(key string) string {
				if key == tt.key {
					return tt.value
				}
				return ""
			})
			if err == nil {
				t.Fatal("Load() error = nil, want an error")
			}
			if !strings.Contains(err.Error(), tt.key) {
				t.Fatalf("Load() error = %q, want it to name %s", err, tt.key)
			}
		})
	}
}
