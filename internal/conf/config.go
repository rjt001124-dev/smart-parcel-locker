package conf

import (
	"fmt"
	"strconv"
	"time"
)

type MySQLConfig struct {
	Host     string
	Database string
	User     string
	Password string
	Port     int
}

type RedisConfig struct {
	Host     string
	Username string
	Password string
	Port     int
	Database int
}

type Config struct {
	HTTPAddr               string
	AppEnv                 string
	InternalAPIToken       string
	DeviceOfflineThreshold time.Duration
	MySQL                  MySQLConfig
	Redis                  RedisConfig
}

func Load(getenv func(string) string) (Config, error) {
	mysqlPort, err := envInt(getenv, "MYSQL_PORT", 3306)
	if err != nil {
		return Config{}, err
	}
	if mysqlPort < 1 || mysqlPort > 65535 {
		return Config{}, fmt.Errorf("MYSQL_PORT must be between 1 and 65535")
	}

	redisPort, err := envInt(getenv, "REDIS_PORT", 6379)
	if err != nil {
		return Config{}, err
	}
	if redisPort < 1 || redisPort > 65535 {
		return Config{}, fmt.Errorf("REDIS_PORT must be between 1 and 65535")
	}

	redisDatabase, err := envInt(getenv, "REDIS_DATABASE", 0)
	if err != nil {
		return Config{}, err
	}
	if redisDatabase < 0 {
		return Config{}, fmt.Errorf("REDIS_DATABASE must be non-negative")
	}

	deviceOfflineThreshold, err := envDuration(getenv, "DEVICE_OFFLINE_THRESHOLD", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	if deviceOfflineThreshold <= 0 {
		return Config{}, fmt.Errorf("DEVICE_OFFLINE_THRESHOLD must be positive")
	}

	return Config{
		HTTPAddr:               envString(getenv, "HTTP_ADDR", "0.0.0.0:8000"),
		AppEnv:                 envString(getenv, "APP_ENV", "development"),
		InternalAPIToken:       envString(getenv, "INTERNAL_API_TOKEN", ""),
		DeviceOfflineThreshold: deviceOfflineThreshold,
		MySQL: MySQLConfig{
			Host:     envString(getenv, "MYSQL_HOST", "127.0.0.1"),
			Port:     mysqlPort,
			Database: envString(getenv, "MYSQL_DATABASE", "smart_parcel_locker"),
			User:     envString(getenv, "MYSQL_USER", "locker"),
			Password: envString(getenv, "MYSQL_PASSWORD", ""),
		},
		Redis: RedisConfig{
			Host:     envString(getenv, "REDIS_HOST", "127.0.0.1"),
			Port:     redisPort,
			Username: envString(getenv, "REDIS_USERNAME", ""),
			Password: envString(getenv, "REDIS_PASSWORD", ""),
			Database: redisDatabase,
		},
	}, nil
}

func envString(getenv func(string) string, key, defaultValue string) string {
	value := getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func envInt(getenv func(string) string, key string, defaultValue int) (int, error) {
	value := getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}

func envDuration(getenv func(string) string, key string, defaultValue time.Duration) (time.Duration, error) {
	value := getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return parsed, nil
}
