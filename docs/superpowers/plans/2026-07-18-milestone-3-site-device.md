# Site and Device Milestone Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the MySQL/Redis-backed city, site, locker, cell reservation, and simulated device foundation required by the future smart-locker order flow.

**Architecture:** Keep one Go Kratos modular monolith with `service -> biz -> data` dependencies inside `site`, `locker`, and `device` domains. MySQL 8.4 is the source of truth, Redis 7 is an optional acceleration layer, public and internal APIs are defined in Proto, and an isolated worker reclaims expired reservations.

**Tech Stack:** Go 1.23, Kratos 2.9.2, MySQL 8.4, Redis 7, `database/sql`, `go-sql-driver/mysql`, `go-redis/v9`, Protobuf, Kratos HTTP generation, Docker Compose, Python repository validators, GitHub Actions.

---

## File map

- `api/locker/v1/site.proto`: public city, site, and cell availability contracts.
- `api/locker/v1/internal.proto`: heartbeat, reservation, device command, and simulator contracts.
- `api/locker/v1/*.pb.go`: generated messages; never edit manually.
- `api/locker/v1/*_http.pb.go`: generated Kratos HTTP bindings; never edit manually.
- `internal/conf/config.go`: process, MySQL, Redis, internal-token, and device timing configuration.
- `internal/platform/data/clients.go`: MySQL and Redis construction and shutdown.
- `internal/platform/data/readiness.go`: dependency readiness state.
- `internal/site/biz`: city/site entities, validation, use cases, and repository interfaces.
- `internal/site/data`: MySQL repository and Redis query cache.
- `internal/site/service`: Proto request/response mapping.
- `internal/locker/biz`: cell state and reservation rules.
- `internal/locker/data`: MySQL reservation transaction and Redis marker.
- `internal/locker/service`: public cell and internal reservation API mapping.
- `internal/device/biz`: heartbeat and command rules plus gateway interface.
- `internal/device/data`: MySQL command repository and deterministic simulator.
- `internal/device/service`: internal heartbeat, command, and simulator API mapping.
- `internal/server`: route registration, internal-token middleware, health, and readiness.
- `app/api`: dependency wiring for the HTTP process.
- `app/worker`: expired reservation reclamation process.
- `migrations/mysql`: reversible schema and development seed SQL.
- `.agents/skills/developing-site-device-domain`: project-specific AI rules and evidence.
- `deploy/docker-compose.yml`: API, worker, MySQL, and Redis development services.
- `.github/workflows/ci.yml`: unit, race, migration, integration, rule, and image checks.

## Task 1: Add typed MySQL, Redis, and internal API configuration

**Files:**
- Modify: `internal/conf/config.go`
- Modify: `internal/conf/config_test.go`
- Modify: `app/api/main.go`
- Create: `.env.example`

- [ ] **Step 1: Write failing configuration tests**

Replace `internal/conf/config_test.go` with tests that require defaults, explicit values, integer validation, and secret-safe configuration:

```go
package conf

import "testing"

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "0.0.0.0:8000" || cfg.MySQL.Port != 3306 || cfg.Redis.Port != 6379 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	if cfg.AppEnv != "development" || cfg.DeviceOfflineThreshold.String() != "30s" {
		t.Fatalf("unexpected runtime defaults: %+v", cfg)
	}
}

func TestLoadRejectsInvalidRedisDatabase(t *testing.T) {
	_, err := Load(func(key string) string {
		if key == "REDIS_DATABASE" {
			return "not-a-number"
		}
		return ""
	})
	if err == nil {
		t.Fatal("expected REDIS_DATABASE validation error")
	}
}

func TestLoadExplicitValues(t *testing.T) {
	values := map[string]string{
		"APP_ENV": "test", "MYSQL_HOST": "mysql", "MYSQL_PORT": "3307",
		"MYSQL_DATABASE": "locker_test", "MYSQL_USER": "locker",
		"MYSQL_PASSWORD": "local-only", "REDIS_HOST": "redis",
		"REDIS_PORT": "6380", "REDIS_DATABASE": "2",
		"INTERNAL_API_TOKEN": "integration-token", "DEVICE_OFFLINE_THRESHOLD": "45s",
	}
	cfg, err := Load(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MySQL.Host != "mysql" || cfg.MySQL.Port != 3307 || cfg.Redis.Database != 2 {
		t.Fatalf("unexpected explicit config: %+v", cfg)
	}
	if cfg.InternalAPIToken != "integration-token" || cfg.DeviceOfflineThreshold.String() != "45s" {
		t.Fatalf("unexpected security config: %+v", cfg)
	}
}
```

- [ ] **Step 2: Run the tests and verify the API mismatch**

Run: `go test ./internal/conf`

Expected: FAIL because `Load` currently returns one value and the nested MySQL/Redis configuration does not exist.

- [ ] **Step 3: Implement typed configuration**

Replace `internal/conf/config.go` with:

```go
package conf

import (
	"fmt"
	"strconv"
	"time"
)

const defaultHTTPAddr = "0.0.0.0:8000"

type MySQLConfig struct {
	Host, Database, User, Password string
	Port                           int
}

type RedisConfig struct {
	Host, Username, Password string
	Port, Database           int
}

type Config struct {
	HTTPAddr              string
	AppEnv                string
	InternalAPIToken      string
	DeviceOfflineThreshold time.Duration
	MySQL                 MySQLConfig
	Redis                 RedisConfig
}

func Load(getenv func(string) string) (Config, error) {
	mysqlPort, err := envInt(getenv, "MYSQL_PORT", 3306)
	if err != nil { return Config{}, err }
	redisPort, err := envInt(getenv, "REDIS_PORT", 6379)
	if err != nil { return Config{}, err }
	redisDB, err := envInt(getenv, "REDIS_DATABASE", 0)
	if err != nil { return Config{}, err }
	offline, err := envDuration(getenv, "DEVICE_OFFLINE_THRESHOLD", 30*time.Second)
	if err != nil { return Config{}, err }
	return Config{
		HTTPAddr: envString(getenv, "HTTP_ADDR", defaultHTTPAddr),
		AppEnv: envString(getenv, "APP_ENV", "development"),
		InternalAPIToken: getenv("INTERNAL_API_TOKEN"),
		DeviceOfflineThreshold: offline,
		MySQL: MySQLConfig{
			Host: envString(getenv, "MYSQL_HOST", "127.0.0.1"), Port: mysqlPort,
			Database: envString(getenv, "MYSQL_DATABASE", "smart_parcel_locker"),
			User: envString(getenv, "MYSQL_USER", "locker"), Password: getenv("MYSQL_PASSWORD"),
		},
		Redis: RedisConfig{
			Host: envString(getenv, "REDIS_HOST", "127.0.0.1"), Port: redisPort,
			Username: getenv("REDIS_USERNAME"), Password: getenv("REDIS_PASSWORD"), Database: redisDB,
		},
	}, nil
}

func envString(getenv func(string) string, key, fallback string) string {
	if value := getenv(key); value != "" { return value }
	return fallback
}

func envInt(getenv func(string) string, key string, fallback int) (int, error) {
	value := getenv(key)
	if value == "" { return fallback, nil }
	parsed, err := strconv.Atoi(value)
	if err != nil { return 0, fmt.Errorf("%s must be an integer: %w", key, err) }
	return parsed, nil
}

func envDuration(getenv func(string) string, key string, fallback time.Duration) (time.Duration, error) {
	value := getenv(key)
	if value == "" { return fallback, nil }
	parsed, err := time.ParseDuration(value)
	if err != nil { return 0, fmt.Errorf("%s must be a duration: %w", key, err) }
	return parsed, nil
}
```

Update `app/api/main.go` so configuration errors stop startup before any secret can be logged:

```go
cfg, err := conf.Load(os.Getenv)
if err != nil {
	log.Fatal(err)
}
```

Create `.env.example` with local-only placeholders:

```dotenv
APP_ENV=development
HTTP_ADDR=0.0.0.0:8000
MYSQL_HOST=127.0.0.1
MYSQL_PORT=3306
MYSQL_DATABASE=smart_parcel_locker
MYSQL_USER=locker
MYSQL_PASSWORD=change-this-local-password
MYSQL_ROOT_PASSWORD=change-this-root-password
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_USERNAME=
REDIS_PASSWORD=change-this-local-password
REDIS_DATABASE=0
INTERNAL_API_TOKEN=change-this-local-token
DEVICE_OFFLINE_THRESHOLD=30s
```

- [ ] **Step 4: Run focused and repository tests**

Run: `gofmt -w internal/conf app/api && go test ./internal/conf ./app/api`

Expected: PASS.

- [ ] **Step 5: Commit configuration support**

```bash
git add .env.example internal/conf app/api/main.go
git commit -m "feat: configure mysql redis and internal api"
```

## Task 2: Add local MySQL/Redis services and reversible schema migrations

**Files:**
- Modify: `deploy/docker-compose.yml`
- Create: `migrations/mysql/000001_site_device.up.sql`
- Create: `migrations/mysql/000001_site_device.down.sql`
- Create: `migrations/mysql/000002_development_seed.up.sql`
- Create: `migrations/mysql/000002_development_seed.down.sql`
- Create: `Makefile`

- [ ] **Step 1: Add migration-first schema SQL**

Create `migrations/mysql/000001_site_device.up.sql` with InnoDB tables, UTC timestamps, foreign keys, unique keys, and allocation indexes. Use these exact table names and enums:

```sql
CREATE TABLE cities (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  code VARCHAR(32) NOT NULL,
  name VARCHAR(64) NOT NULL,
  province VARCHAR(64) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id), UNIQUE KEY uk_cities_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE sites (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  site_no VARCHAR(32) NOT NULL, city_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL, address VARCHAR(255) NOT NULL,
  latitude DECIMAL(10,7) NOT NULL, longitude DECIMAL(10,7) NOT NULL,
  open_time TIME NOT NULL, close_time TIME NOT NULL,
  contact_phone VARCHAR(32) NOT NULL,
  service_status ENUM('ACTIVE','SUSPENDED','CLOSED') NOT NULL DEFAULT 'ACTIVE',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id), UNIQUE KEY uk_sites_site_no (site_no),
  KEY idx_sites_city_status (city_id, service_status),
  KEY idx_sites_coordinates (latitude, longitude),
  CONSTRAINT fk_sites_city FOREIGN KEY (city_id) REFERENCES cities(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE locker_devices (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_no VARCHAR(64) NOT NULL, site_id BIGINT UNSIGNED NOT NULL,
  protocol_type VARCHAR(32) NOT NULL DEFAULT 'SIMULATOR',
  network_status ENUM('ONLINE','OFFLINE') NOT NULL DEFAULT 'OFFLINE',
  operational_status ENUM('ACTIVE','MAINTENANCE','DISABLED') NOT NULL DEFAULT 'ACTIVE',
  last_heartbeat_at DATETIME(6) NULL, firmware_version VARCHAR(64) NOT NULL DEFAULT '',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id), UNIQUE KEY uk_locker_devices_device_no (device_no),
  KEY idx_locker_devices_site_status (site_id, network_status, operational_status),
  CONSTRAINT fk_locker_devices_site FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE locker_cells (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  device_id BIGINT UNSIGNED NOT NULL, cell_no VARCHAR(32) NOT NULL,
  size ENUM('SMALL','MEDIUM','LARGE') NOT NULL,
  occupancy_status ENUM('IDLE','LOCKED','OCCUPIED','DISABLED') NOT NULL DEFAULT 'IDLE',
  door_status ENUM('CLOSED','OPEN','UNKNOWN') NOT NULL DEFAULT 'CLOSED',
  reservation_key VARCHAR(128) NULL, lock_expires_at DATETIME(6) NULL,
  current_order_id BIGINT UNSIGNED NULL, version BIGINT UNSIGNED NOT NULL DEFAULT 0,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id), UNIQUE KEY uk_locker_cells_device_cell (device_id, cell_no),
  UNIQUE KEY uk_locker_cells_reservation_key (reservation_key),
  KEY idx_locker_cells_allocation (device_id, size, occupancy_status, lock_expires_at),
  CONSTRAINT fk_locker_cells_device FOREIGN KEY (device_id) REFERENCES locker_devices(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE device_commands (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  command_no CHAR(26) NOT NULL, device_id BIGINT UNSIGNED NOT NULL,
  action ENUM('OPEN_DOOR','QUERY_STATUS') NOT NULL,
  payload_json JSON NOT NULL, idempotency_key VARCHAR(128) NOT NULL,
  status ENUM('PENDING','RUNNING','SUCCEEDED','FAILED','TIMED_OUT','EXPIRED') NOT NULL DEFAULT 'PENDING',
  expires_at DATETIME(6) NOT NULL, attempt_count INT UNSIGNED NOT NULL DEFAULT 0,
  result_json JSON NULL, error_code VARCHAR(64) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id), UNIQUE KEY uk_device_commands_command_no (command_no),
  UNIQUE KEY uk_device_commands_idempotency_key (idempotency_key),
  KEY idx_device_commands_device_status (device_id, status, created_at),
  CONSTRAINT fk_device_commands_device FOREIGN KEY (device_id) REFERENCES locker_devices(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
```

Create `migrations/mysql/000001_site_device.down.sql` with the exact reverse dependency order:

```sql
DROP TABLE IF EXISTS device_commands;
DROP TABLE IF EXISTS locker_cells;
DROP TABLE IF EXISTS locker_devices;
DROP TABLE IF EXISTS sites;
DROP TABLE IF EXISTS cities;
```

Create `migrations/mysql/000002_development_seed.up.sql` with deterministic local data:

```sql
INSERT INTO cities (id, code, name, province, enabled)
VALUES (1, '310100', '上海市', '上海市', TRUE);

INSERT INTO sites (id, site_no, city_id, name, address, latitude, longitude, open_time, close_time, contact_phone, service_status)
VALUES
  (1, 'SITE-SH-001', 1, '人民广场寄存点', '上海市黄浦区人民大道 100 号', 31.2304000, 121.4737000, '08:00:00', '22:00:00', '13800000001', 'ACTIVE'),
  (2, 'SITE-SH-002', 1, '南京东路寄存点', '上海市黄浦区南京东路 200 号', 31.2361000, 121.4802000, '09:00:00', '21:00:00', '13800000002', 'ACTIVE');

INSERT INTO locker_devices (id, device_no, site_id, protocol_type, network_status, operational_status, last_heartbeat_at, firmware_version)
VALUES
  (1, 'DEV-SH-001', 1, 'SIMULATOR', 'ONLINE', 'ACTIVE', UTC_TIMESTAMP(6), 'sim-1.0.0'),
  (2, 'DEV-SH-002', 2, 'SIMULATOR', 'ONLINE', 'ACTIVE', UTC_TIMESTAMP(6), 'sim-1.0.0');

INSERT INTO locker_cells (device_id, cell_no, size, occupancy_status, door_status)
VALUES
  (1, 'A01', 'SMALL', 'IDLE', 'CLOSED'), (1, 'A02', 'SMALL', 'IDLE', 'CLOSED'),
  (1, 'B01', 'MEDIUM', 'IDLE', 'CLOSED'), (1, 'C01', 'LARGE', 'IDLE', 'CLOSED'),
  (2, 'A01', 'SMALL', 'IDLE', 'CLOSED'), (2, 'B01', 'MEDIUM', 'IDLE', 'CLOSED'),
  (2, 'B02', 'MEDIUM', 'IDLE', 'CLOSED'), (2, 'C01', 'LARGE', 'IDLE', 'CLOSED');
```

Create `migrations/mysql/000002_development_seed.down.sql`:

```sql
DELETE FROM locker_cells WHERE device_id IN (1, 2);
DELETE FROM locker_devices WHERE id IN (1, 2);
DELETE FROM sites WHERE id IN (1, 2);
DELETE FROM cities WHERE id = 1;
```

- [ ] **Step 2: Add Docker Compose services**

Replace `deploy/docker-compose.yml` with the following baseline; Task 13 adds the fully wired worker dependencies after the binary exists:

```yaml
services:
  mysql:
    image: mysql:8.4
    environment:
      TZ: UTC
      MYSQL_DATABASE: ${MYSQL_DATABASE:-smart_parcel_locker}
      MYSQL_USER: ${MYSQL_USER:-locker}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD:-change-this-local-password}
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD:-change-this-root-password}
    command: ["--default-time-zone=+00:00", "--character-set-server=utf8mb4", "--collation-server=utf8mb4_0900_ai_ci"]
    ports: ["127.0.0.1:3306:3306"]
    volumes: ["mysql_data:/var/lib/mysql"]
    healthcheck:
      test: ["CMD-SHELL", "mysqladmin ping -h 127.0.0.1 -uroot -p$${MYSQL_ROOT_PASSWORD} --silent"]
      interval: 5s
      timeout: 3s
      retries: 30
  redis:
    image: redis:7.4-alpine
    environment:
      REDIS_PASSWORD: ${REDIS_PASSWORD:-change-this-local-password}
    command: ["sh", "-c", "exec redis-server --appendonly yes --requirepass \"$${REDIS_PASSWORD}\""]
    ports: ["127.0.0.1:6379:6379"]
    volumes: ["redis_data:/data"]
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a \"$${REDIS_PASSWORD}\" ping | grep PONG"]
      interval: 5s
      timeout: 3s
      retries: 30
  api:
    build:
      context: ..
      dockerfile: deploy/Dockerfile.api
      args: { VERSION: local }
    environment:
      APP_ENV: development
      HTTP_ADDR: 0.0.0.0:8000
      MYSQL_HOST: mysql
      MYSQL_PORT: 3306
      MYSQL_DATABASE: ${MYSQL_DATABASE:-smart_parcel_locker}
      MYSQL_USER: ${MYSQL_USER:-locker}
      MYSQL_PASSWORD: ${MYSQL_PASSWORD:-change-this-local-password}
      REDIS_HOST: redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: ${REDIS_PASSWORD:-change-this-local-password}
      INTERNAL_API_TOKEN: ${INTERNAL_API_TOKEN:-change-this-local-token}
    ports: ["127.0.0.1:8000:8000"]
    depends_on:
      mysql: { condition: service_healthy }
      redis: { condition: service_healthy }
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://127.0.0.1:8000/healthz"]
      interval: 5s
      timeout: 2s
      retries: 10
volumes:
  mysql_data:
  redis_data:
```

- [ ] **Step 3: Add repeatable migration commands**

Create a `Makefile` with commands that use the official migrate container:

```make
MIGRATIONS := $(CURDIR)/migrations/mysql
MYSQL_URL ?= mysql://locker:change-this-local-password@tcp(127.0.0.1:3306)/smart_parcel_locker?multiStatements=true&parseTime=true&loc=UTC

.PHONY: db-up db-down migrate-up migrate-down migrate-cycle
db-up:
	docker compose -f deploy/docker-compose.yml up -d mysql redis
db-down:
	docker compose -f deploy/docker-compose.yml down
migrate-up:
	go run -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 -path="$(MIGRATIONS)" -database "$(MYSQL_URL)" up
migrate-down:
	go run -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 -path="$(MIGRATIONS)" -database "$(MYSQL_URL)" down -all
migrate-cycle: migrate-down migrate-up
```

- [ ] **Step 4: Verify schema creation and reversal**

Run:

```bash
docker compose -f deploy/docker-compose.yml config
docker compose -f deploy/docker-compose.yml up -d mysql redis
make migrate-up
docker compose -f deploy/docker-compose.yml exec -T mysql mysql -ulocker -pchange-this-local-password smart_parcel_locker -e "SHOW TABLES;"
make migrate-cycle
```

Expected: five business tables appear after each up migration, and the cycle exits 0.

- [ ] **Step 5: Commit local persistence infrastructure**

```bash
git add deploy/docker-compose.yml migrations/mysql Makefile
git commit -m "feat: add mysql redis development persistence"
```

## Task 3: Create platform data clients and readiness checks

**Files:**
- Create: `internal/platform/data/clients.go`
- Create: `internal/platform/data/clients_test.go`
- Create: `internal/platform/data/readiness.go`
- Create: `internal/platform/data/readiness_test.go`
- Modify: `internal/server/health.go`
- Modify: `internal/server/health_test.go`

- [ ] **Step 1: Write failing DSN and readiness tests**

Test that `MySQLDSN` includes `parseTime=true`, `loc=UTC`, and does not appear in readiness responses. Test a readiness checker with healthy MySQL/Redis, failed MySQL, and failed Redis; failed MySQL returns HTTP 503, while failed Redis returns HTTP 200 with status `degraded`.

```go
func TestMySQLDSNUsesUTC(t *testing.T) {
	cfg := conf.MySQLConfig{Host: "mysql", Port: 3306, Database: "locker", User: "app", Password: "secret"}
	dsn := MySQLDSN(cfg)
	if !strings.Contains(dsn, "parseTime=true") || !strings.Contains(dsn, "loc=UTC") {
		t.Fatalf("DSN missing UTC settings: %s", dsn)
	}
}
```

- [ ] **Step 2: Run focused tests and verify missing packages**

Run: `go test ./internal/platform/data ./internal/server`

Expected: FAIL because the platform data package and readiness registration do not exist.

- [ ] **Step 3: Implement clients and readiness**

Use `github.com/go-sql-driver/mysql` and `github.com/redis/go-redis/v9`. `Open` must set MySQL connection limits, ping both dependencies with a five-second context, and return a `Clients` value with a single idempotent `Close` method. `MySQLDSN` must use `mysql.NewConfig()` rather than string concatenation.

```go
func MySQLDSN(cfg conf.MySQLConfig) string {
	parsed := mysql.NewConfig()
	parsed.User, parsed.Passwd = cfg.User, cfg.Password
	parsed.Net = "tcp"
	parsed.Addr = net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))
	parsed.DBName = cfg.Database
	parsed.ParseTime = true
	parsed.Loc = time.UTC
	parsed.MultiStatements = false
	return parsed.FormatDSN()
}
```

Define a small readiness interface so server tests use fakes:

```go
type Readiness interface {
	Check(context.Context) Status
}

type Status struct {
	Status string `json:"status"`
	MySQL string `json:"mysql"`
	Redis string `json:"redis"`
}
```

Register `/readyz` separately from `/healthz`; never include hostnames, usernames, DSNs, or errors containing credentials in the JSON response.

- [ ] **Step 4: Run tests and dependency verification**

Run: `go mod tidy && gofmt -w internal/platform internal/server && go test ./internal/platform/data ./internal/server`

Expected: PASS.

- [ ] **Step 5: Commit shared data infrastructure**

```bash
git add go.mod go.sum internal/platform internal/server
git commit -m "feat: add data clients and readiness checks"
```

## Task 4: Define and test the site domain

**Files:**
- Create: `internal/site/biz/model.go`
- Create: `internal/site/biz/repository.go`
- Create: `internal/site/biz/usecase.go`
- Create: `internal/site/biz/usecase_test.go`

- [ ] **Step 1: Write failing site use-case tests**

Use an in-memory repository fake and cover disabled cities, coordinate validation, radius default/max, active-site filtering, and ascending distance order.

```go
func TestListNearbySitesSortsByDistance(t *testing.T) {
	repo := &fakeRepository{
		cities: []City{{ID: 1, Code: "310100", Enabled: true}},
		sites: []Site{
			{ID: 1, CityID: 1, Name: "far", Latitude: 31.2500, Longitude: 121.5000, Status: SiteActive},
			{ID: 2, CityID: 1, Name: "near", Latitude: 31.2305, Longitude: 121.4738, Status: SiteActive},
		},
	}
	uc := NewUseCase(repo)
	got, err := uc.ListNearby(context.Background(), NearbyQuery{CityCode: "310100", Latitude: 31.2304, Longitude: 121.4737, RadiusM: 50000})
	if err != nil { t.Fatal(err) }
	if len(got) != 2 || got[0].Site.Name != "near" || got[0].DistanceM > got[1].DistanceM {
		t.Fatalf("unexpected ordering: %+v", got)
	}
}
```

- [ ] **Step 2: Run tests and verify missing domain**

Run: `go test ./internal/site/biz`

Expected: FAIL because `City`, `Site`, `NearbyQuery`, `Repository`, and `UseCase` do not exist.

- [ ] **Step 3: Implement site entities and rules**

Define `City`, `Site`, `SiteStatus`, `NearbyQuery`, and `NearbySite`. Define repository methods `ListEnabledCities`, `FindCityByCode`, `ListCandidateSites`, `GetSite`, and `GetAvailabilitySummary`. Keep Haversine distance and input validation in biz so it can be tested without MySQL.

```go
func validateCoordinates(latitude, longitude float64) error {
	if latitude < -90 || latitude > 90 || longitude < -180 || longitude > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}

func normalizedRadius(value int32) (int32, error) {
	if value == 0 { return 5000, nil }
	if value < 1 || value > 50000 { return 0, ErrInvalidRadius }
	return value, nil
}
```

Filter inactive sites before sorting. Return typed sentinel errors that the service layer can map to stable business codes.

- [ ] **Step 4: Run site unit tests**

Run: `gofmt -w internal/site/biz && go test ./internal/site/biz`

Expected: PASS.

- [ ] **Step 5: Commit the site domain**

```bash
git add internal/site/biz
git commit -m "feat: define city and site use cases"
```

## Task 5: Implement the MySQL site repository and Redis cache

**Files:**
- Create: `internal/site/data/repository.go`
- Create: `internal/site/data/repository_integration_test.go`
- Create: `internal/site/data/cache.go`
- Create: `internal/site/data/cache_test.go`

- [ ] **Step 1: Write failing repository integration tests**

Create tests behind build tag `integration` that read `TEST_MYSQL_DSN`, apply migrations before the package run, and verify enabled-city filtering, city lookup, candidate-site filtering, and availability counts. A separate cache unit test uses `miniredis` to verify a 30-second key and JSON round trip without storing contact phone or credentials.

- [ ] **Step 2: Run unit and integration tests to verify missing repository**

Run:

```bash
go test ./internal/site/data
go test -tags=integration ./internal/site/data
```

Expected: both commands fail until the repository and test database environment are added.

- [ ] **Step 3: Implement explicit MySQL queries**

Use `database/sql`; do not introduce an ORM. Candidate queries must first narrow by city and active status. Availability counts join `locker_devices` and `locker_cells`, exclude disabled devices/cells, and treat stale heartbeat devices as unavailable.

```sql
SELECT s.id, s.site_no, s.city_id, s.name, s.address, s.latitude, s.longitude,
       s.open_time, s.close_time, s.contact_phone, s.service_status
FROM sites s
JOIN cities c ON c.id = s.city_id
WHERE c.code = ? AND c.enabled = TRUE AND s.service_status = 'ACTIVE'
  AND s.latitude BETWEEN ? AND ? AND s.longitude BETWEEN ? AND ?;
```

The data layer returns domain entities and wraps errors without embedding SQL arguments that may contain sensitive values.

- [ ] **Step 4: Implement optional Redis caching**

Cache only public nearby-site results under a SHA-256-derived key. Redis read/write errors are recorded as degraded metrics/log events and treated as cache misses. MySQL errors are returned.

- [ ] **Step 5: Run repository tests with local services**

Run:

```bash
go test ./internal/site/data
TEST_MYSQL_DSN='locker:change-this-local-password@tcp(127.0.0.1:3306)/smart_parcel_locker?parseTime=true&loc=UTC' go test -tags=integration ./internal/site/data
```

Expected: PASS.

- [ ] **Step 6: Commit site persistence**

```bash
git add internal/site/data go.mod go.sum
git commit -m "feat: persist and cache site queries"
```

## Task 6: Add public Proto contracts and site HTTP service

**Files:**
- Create: `api/locker/v1/site.proto`
- Create: `buf.yaml`
- Create: `buf.gen.yaml`
- Modify: `Makefile`
- Create: `internal/site/service/service.go`
- Create: `internal/site/service/service_test.go`

- [ ] **Step 1: Write the public Proto contract**

Define `SiteService` with `ListCities`, `ListSites`, `GetSite`, and `ListCells`. Use `google.api.http` annotations for the exact public paths approved in the design. Money fields are absent from this milestone. Latitude/longitude are request doubles; database conversion remains inside data/biz.

```proto
syntax = "proto3";
package locker.v1;
option go_package = "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1;v1";

import "google/api/annotations.proto";

service SiteService {
  rpc ListCities(ListCitiesRequest) returns (ListCitiesReply) { option (google.api.http) = { get: "/v1/cities" }; }
  rpc ListSites(ListSitesRequest) returns (ListSitesReply) { option (google.api.http) = { get: "/v1/sites" }; }
  rpc GetSite(GetSiteRequest) returns (GetSiteReply) { option (google.api.http) = { get: "/v1/sites/{site_id}" }; }
  rpc ListCells(ListCellsRequest) returns (ListCellsReply) { option (google.api.http) = { get: "/v1/sites/{site_id}/cells" }; }
}

enum CellSize {
  CELL_SIZE_UNSPECIFIED = 0;
  CELL_SIZE_SMALL = 1;
  CELL_SIZE_MEDIUM = 2;
  CELL_SIZE_LARGE = 3;
}

enum CellStatus {
  CELL_STATUS_UNSPECIFIED = 0;
  CELL_STATUS_IDLE = 1;
  CELL_STATUS_LOCKED = 2;
  CELL_STATUS_OCCUPIED = 3;
  CELL_STATUS_DISABLED = 4;
}

message City { string id = 1; string code = 2; string name = 3; string province = 4; }
message ListCitiesRequest {}
message ListCitiesReply { repeated City cities = 1; }

message CellAvailability { CellSize size = 1; int32 available_count = 2; }
message SiteSummary {
  string id = 1; string site_no = 2; string name = 3; string address = 4;
  double latitude = 5; double longitude = 6; int32 distance_m = 7;
  repeated CellAvailability availability = 8;
}
message ListSitesRequest {
  string city_code = 1; double latitude = 2; double longitude = 3; int32 radius_m = 4;
}
message ListSitesReply { repeated SiteSummary sites = 1; }

message GetSiteRequest { string site_id = 1; }
message GetSiteReply {
  string id = 1; string site_no = 2; string name = 3; string address = 4;
  double latitude = 5; double longitude = 6; string open_time = 7; string close_time = 8;
  int32 online_device_count = 9; repeated CellAvailability availability = 10;
}

message ListCellsRequest { string site_id = 1; CellSize size = 2; CellStatus status = 3; }
message CellView { string id = 1; string cell_no = 2; CellSize size = 3; CellStatus status = 4; }
message ListCellsReply { repeated CellView cells = 1; }
```

Keep these field numbers stable. Use strings for public IDs and enums for site/cell state. Do not reuse field numbers after generation begins.

- [ ] **Step 2: Add deterministic generation**

Create `buf.yaml`:

```yaml
version: v2
modules:
  - path: api
deps:
  - buf.build/googleapis/googleapis
lint:
  use: [STANDARD]
breaking:
  use: [FILE]
```

Create `buf.gen.yaml`:

```yaml
version: v2
plugins:
  - local: protoc-gen-go
    out: api
    opt: [paths=source_relative]
  - local: protoc-gen-go-http
    out: api
    opt: [paths=source_relative]
```

Append these targets to `Makefile`:

```make
.PHONY: tools api
tools:
	go install github.com/bufbuild/buf/cmd/buf@v1.50.0
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@v2.9.2
api:
	buf lint
	buf generate
```

- [ ] **Step 3: Generate code and write failing service tests**

Run: `make tools api`

Then write service tests with a fake use case that verify query mapping, public-ID parsing, error-code mapping, and that responses omit `reservation_key`, `contact_phone`, and internal device information.

- [ ] **Step 4: Implement the service adapter**

`internal/site/service.Service` embeds `v1.UnimplementedSiteServiceServer`, accepts a small use-case interface, converts Proto requests to biz queries, and maps typed errors with `errors.NotFound`, `errors.BadRequest`, or `errors.ServiceUnavailable` using stable reason strings such as `CITY_NOT_FOUND`.

- [ ] **Step 5: Run generation checks and service tests**

Run:

```bash
make api
git diff --exit-code api/locker/v1
go test ./internal/site/service
```

Expected: generated files are reproducible and tests pass.

- [ ] **Step 6: Commit public API support**

```bash
git add api buf.yaml buf.gen.yaml Makefile internal/site/service go.mod go.sum
git commit -m "feat: expose city and site query api"
```

## Task 7: Define cell reservation rules with test-first fakes

**Files:**
- Create: `internal/locker/biz/model.go`
- Create: `internal/locker/biz/repository.go`
- Create: `internal/locker/biz/usecase.go`
- Create: `internal/locker/biz/usecase_test.go`

- [ ] **Step 1: Write failing reservation tests**

Cover default/max TTL, empty idempotency keys, offline devices, no available cell, same-key replay, mismatched release key, and expiration reclaim.

```go
func TestReserveReturnsExistingReservationForSameKey(t *testing.T) {
	expires := time.Date(2026, 7, 18, 1, 5, 0, 0, time.UTC)
	repo := &fakeRepository{existing: &Reservation{CellID: 9, ReservationKey: "req-1", ExpiresAt: expires}}
	uc := NewUseCase(repo, nil, func() time.Time { return expires.Add(-time.Minute) })
	got, err := uc.Reserve(context.Background(), ReserveRequest{SiteID: 1, Size: SizeMedium, ReservationKey: "req-1", TTL: time.Minute})
	if err != nil { t.Fatal(err) }
	if got.CellID != 9 || repo.reserveCalls != 0 { t.Fatalf("unexpected replay: %+v calls=%d", got, repo.reserveCalls) }
}
```

- [ ] **Step 2: Run tests and verify missing reservation domain**

Run: `go test ./internal/locker/biz`

Expected: FAIL because the reservation model and use case do not exist.

- [ ] **Step 3: Implement reservation use cases**

Define cell size/status enums, `Reservation`, `ReserveRequest`, repository methods `FindReservation`, `ReserveAvailable`, `Release`, `ReclaimExpired`, and optional `Marker` methods. Enforce TTL between 30 seconds and 15 minutes. MySQL success is returned even if Redis marker creation fails.

```go
func (uc *UseCase) Reserve(ctx context.Context, req ReserveRequest) (Reservation, error) {
	if req.ReservationKey == "" { return Reservation{}, ErrIdempotencyKeyRequired }
	if req.TTL < 30*time.Second || req.TTL > 15*time.Minute { return Reservation{}, ErrInvalidReservationTTL }
	if existing, err := uc.repo.FindReservation(ctx, req.ReservationKey); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrReservationNotFound) {
		return Reservation{}, err
	}
	reservation, err := uc.repo.ReserveAvailable(ctx, req, uc.now().Add(req.TTL))
	if err != nil { return Reservation{}, err }
	if uc.marker != nil { _ = uc.marker.Mark(ctx, reservation) }
	return reservation, nil
}
```

- [ ] **Step 4: Run locker domain tests**

Run: `gofmt -w internal/locker/biz && go test ./internal/locker/biz`

Expected: PASS.

- [ ] **Step 5: Commit reservation rules**

```bash
git add internal/locker/biz
git commit -m "feat: define idempotent cell reservations"
```

## Task 8: Implement atomic MySQL cell reservation and Redis markers

**Files:**
- Create: `internal/locker/data/repository.go`
- Create: `internal/locker/data/repository_integration_test.go`
- Create: `internal/locker/data/marker.go`
- Create: `internal/locker/data/marker_test.go`

- [ ] **Step 1: Write failing concurrent integration tests**

Create one online device with exactly one idle medium cell. Start 20 goroutines with distinct idempotency keys and assert exactly one succeeds. Start 20 goroutines with the same idempotency key and assert every successful response has the same cell and expiry.

- [ ] **Step 2: Run integration tests and capture the missing implementation failure**

Run: `go test -tags=integration -run 'TestReserveAvailableConcurrent|TestReserveSameKeyConcurrent' ./internal/locker/data`

Expected: FAIL because no MySQL locker repository exists.

- [ ] **Step 3: Implement the transaction**

Inside one transaction:

```sql
SELECT lc.id, ld.id
FROM locker_cells lc
JOIN locker_devices ld ON ld.id = lc.device_id
WHERE ld.site_id = ?
  AND ld.network_status = 'ONLINE'
  AND ld.operational_status = 'ACTIVE'
  AND ld.last_heartbeat_at >= ?
  AND lc.size = ?
  AND (lc.occupancy_status = 'IDLE' OR (lc.occupancy_status = 'LOCKED' AND lc.lock_expires_at < ?))
  AND lc.current_order_id IS NULL
ORDER BY lc.id
LIMIT 1
FOR UPDATE SKIP LOCKED;
```

Then conditionally update the selected row, increment `version`, commit, and return the reservation. If the unique `reservation_key` constraint races, roll back and read the existing reservation. Release and reclaim updates include reservation/current-order conditions.

- [ ] **Step 4: Implement Redis markers as non-authoritative state**

Use key `locker:reservation:{hex_sha256_reservation_key}` with expiration equal to the remaining reservation TTL. `Mark` and `Delete` return errors to logging/metrics callers but never decide MySQL allocation correctness.

- [ ] **Step 5: Run concurrent and Redis-loss tests**

Run:

```bash
go test ./internal/locker/data
go test -tags=integration -race ./internal/locker/data
```

Expected: PASS; one distinct-key winner, same-key replay consistency, and correct MySQL state after Redis flush.

- [ ] **Step 6: Commit reservation persistence**

```bash
git add internal/locker/data go.mod go.sum
git commit -m "feat: reserve locker cells atomically"
```

## Task 9: Define device command rules and deterministic simulator

**Files:**
- Create: `internal/device/biz/model.go`
- Create: `internal/device/biz/repository.go`
- Create: `internal/device/biz/gateway.go`
- Create: `internal/device/biz/usecase.go`
- Create: `internal/device/biz/usecase_test.go`
- Create: `internal/device/data/simulator.go`
- Create: `internal/device/data/simulator_test.go`

- [ ] **Step 1: Write failing command and simulator tests**

Test heartbeat freshness, expired commands, idempotent command replay, maximum attempts, and each scenario: online success, offline error, deadline timeout, explicit failure, and door-left-open result.

```go
func TestSimulatorDoorLeftOpen(t *testing.T) {
	sim := NewSimulator()
	sim.SetScenario("DEV-001", ScenarioDoorLeftOpen)
	result, err := sim.Execute(context.Background(), devicebiz.GatewayCommand{DeviceNo: "DEV-001", Action: devicebiz.ActionOpenDoor})
	if err != nil { t.Fatal(err) }
	if !result.Opened || result.DoorClosed { t.Fatalf("unexpected result: %+v", result) }
}
```

- [ ] **Step 2: Run focused tests and verify missing device domain**

Run: `go test ./internal/device/...`

Expected: FAIL because device command models, use cases, and simulator do not exist.

- [ ] **Step 3: Implement the device domain**

Define actions `OPEN_DOOR` and `QUERY_STATUS`, command states, gateway result, repository interface, and a use case that checks device freshness and command expiry before gateway execution. Persist `PENDING`, then terminal status. Same idempotency key returns the original command and never executes the gateway twice.

- [ ] **Step 4: Implement deterministic simulator scenarios**

Store scenario state in a mutex-protected map. `TIMEOUT` waits on the request context rather than sleeping beyond the deadline. `FAIL` returns a typed gateway error. `DOOR_LEFT_OPEN` returns an opened-but-not-closed result. No simulator state is written to production tables.

- [ ] **Step 5: Run unit tests with the race detector**

Run: `gofmt -w internal/device && go test -race ./internal/device/...`

Expected: PASS.

- [ ] **Step 6: Commit device behavior**

```bash
git add internal/device
git commit -m "feat: add simulated locker device commands"
```

## Task 10: Persist heartbeats and device commands in MySQL

**Files:**
- Create: `internal/device/data/repository.go`
- Create: `internal/device/data/repository_integration_test.go`

- [ ] **Step 1: Write failing MySQL command tests**

Verify heartbeat updates only the addressed device, command idempotency is unique, terminal results persist as JSON, expired commands cannot return to running, and no raw payload is included in repository errors.

- [ ] **Step 2: Run integration tests to verify missing repository**

Run: `go test -tags=integration ./internal/device/data`

Expected: FAIL because `NewRepository` and persistence methods do not exist.

- [ ] **Step 3: Implement explicit MySQL persistence**

Use one repository with `UpdateHeartbeat`, `FindDevice`, `FindCommandByIdempotencyKey`, `CreateCommand`, `MarkRunning`, and `CompleteCommand`. `CompleteCommand` uses a conditional status update so a timed-out/expired command cannot be overwritten by a late simulator result.

- [ ] **Step 4: Run integration and race tests**

Run: `go test -tags=integration -race ./internal/device/data`

Expected: PASS.

- [ ] **Step 5: Commit device persistence**

```bash
git add internal/device/data
git commit -m "feat: persist device heartbeats and commands"
```

## Task 11: Add internal Proto APIs, token middleware, and service adapters

**Files:**
- Create: `api/locker/v1/internal.proto`
- Create: `internal/server/internal_auth.go`
- Create: `internal/server/internal_auth_test.go`
- Create: `internal/locker/service/service.go`
- Create: `internal/locker/service/service_test.go`
- Create: `internal/device/service/service.go`
- Create: `internal/device/service/service_test.go`

- [ ] **Step 1: Define the internal contract**

Create `api/locker/v1/internal.proto` with this contract:

```proto
syntax = "proto3";
package locker.v1;
option go_package = "github.com/rjt001124-dev/smart-parcel-locker/api/locker/v1;v1";

import "google/api/annotations.proto";
import "locker/v1/site.proto";

service InternalLockerService {
  rpc ReserveCell(ReserveCellRequest) returns (ReserveCellReply) {
    option (google.api.http) = { post: "/v1/internal/cells/reserve" body: "*" };
  }
  rpc ReleaseCell(ReleaseCellRequest) returns (ReleaseCellReply) {
    option (google.api.http) = { post: "/v1/internal/cells/{cell_id}/release" body: "*" };
  }
}

service InternalDeviceService {
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatReply) {
    option (google.api.http) = { post: "/v1/internal/devices/{device_no}/heartbeat" body: "*" };
  }
  rpc CreateDeviceCommand(CreateDeviceCommandRequest) returns (DeviceCommandReply) {
    option (google.api.http) = { post: "/v1/internal/device-commands" body: "*" };
  }
  rpc GetDeviceCommand(GetDeviceCommandRequest) returns (DeviceCommandReply) {
    option (google.api.http) = { get: "/v1/internal/device-commands/{command_no}" };
  }
  rpc SetSimulatorScenario(SetSimulatorScenarioRequest) returns (SetSimulatorScenarioReply) {
    option (google.api.http) = { put: "/v1/internal/simulator/devices/{device_no}/scenario" body: "*" };
  }
}

message ReserveCellRequest {
  string site_id = 1; CellSize size = 2; int32 ttl_seconds = 3; string idempotency_key = 4;
}
message ReserveCellReply { string cell_id = 1; string device_no = 2; string cell_no = 3; string expires_at = 4; }
message ReleaseCellRequest { string cell_id = 1; string idempotency_key = 2; }
message ReleaseCellReply { bool released = 1; }

message HeartbeatRequest {
  string device_no = 1; string firmware_version = 2; string reported_at = 3; bool online = 4;
}
message HeartbeatReply { string accepted_at = 1; }

enum DeviceAction {
  DEVICE_ACTION_UNSPECIFIED = 0;
  DEVICE_ACTION_OPEN_DOOR = 1;
  DEVICE_ACTION_QUERY_STATUS = 2;
}
enum DeviceCommandStatus {
  DEVICE_COMMAND_STATUS_UNSPECIFIED = 0;
  DEVICE_COMMAND_STATUS_PENDING = 1;
  DEVICE_COMMAND_STATUS_RUNNING = 2;
  DEVICE_COMMAND_STATUS_SUCCEEDED = 3;
  DEVICE_COMMAND_STATUS_FAILED = 4;
  DEVICE_COMMAND_STATUS_TIMED_OUT = 5;
  DEVICE_COMMAND_STATUS_EXPIRED = 6;
}
message CreateDeviceCommandRequest {
  string device_no = 1; DeviceAction action = 2; string cell_no = 3;
  int32 ttl_seconds = 4; string idempotency_key = 5;
}
message GetDeviceCommandRequest { string command_no = 1; }
message DeviceCommandReply {
  string command_no = 1; DeviceCommandStatus status = 2; bool retryable = 3;
  bool door_opened = 4; bool door_closed = 5; string error_code = 6;
}

enum SimulatorScenario {
  SIMULATOR_SCENARIO_UNSPECIFIED = 0;
  SIMULATOR_SCENARIO_ONLINE = 1;
  SIMULATOR_SCENARIO_OFFLINE = 2;
  SIMULATOR_SCENARIO_TIMEOUT = 3;
  SIMULATOR_SCENARIO_FAIL = 4;
  SIMULATOR_SCENARIO_DOOR_LEFT_OPEN = 5;
}
message SetSimulatorScenarioRequest { string device_no = 1; SimulatorScenario scenario = 2; }
message SetSimulatorScenarioReply { bool updated = 1; }
```

Requests include idempotency keys where required; responses never include database IDs, DSNs, or credentials.

- [ ] **Step 2: Write failing token middleware tests**

Use constant-time comparison and test missing, wrong, and correct `X-Internal-Token` headers. An empty configured token must fail startup in non-test environments instead of disabling authentication.

```go
func TestInternalAuthRejectsWrongToken(t *testing.T) {
	nextCalled := false
	handler := InternalAuth("expected")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }))
	req := httptest.NewRequest(http.MethodPost, "/v1/internal/cells/reserve", nil)
	req.Header.Set("X-Internal-Token", "wrong")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized || nextCalled { t.Fatalf("unexpected auth result: %d", recorder.Code) }
}
```

- [ ] **Step 3: Generate APIs and write service tests**

Run: `make api`

Write adapter tests for stable error reasons, TTL conversion, idempotency forwarding, command result mapping, and simulator route rejection when `APP_ENV=production`.

- [ ] **Step 4: Implement middleware and services**

Register public services normally. Register internal services behind token middleware. Construct the simulator service only outside production; the route must not exist in production server tests.

- [ ] **Step 5: Run API and service tests**

Run:

```bash
make api
git diff --exit-code api/locker/v1
go test ./internal/server ./internal/locker/service ./internal/device/service
```

Expected: PASS.

- [ ] **Step 6: Commit secured internal APIs**

```bash
git add api/locker/v1 internal/server internal/locker/service internal/device/service
git commit -m "feat: expose secured locker device api"
```

## Task 12: Wire the API process and expired-reservation worker

**Files:**
- Modify: `internal/server/http.go`
- Modify: `app/api/main.go`
- Create: `app/worker/main.go`
- Create: `internal/locker/biz/reclaimer.go`
- Create: `internal/locker/biz/reclaimer_test.go`
- Create: `deploy/Dockerfile.worker`

- [ ] **Step 1: Write failing reclaimer tests**

Use a fake repository and fake clock to verify that one run calls `ReclaimExpired(now, batchSize)` and that context cancellation stops the loop without another call.

- [ ] **Step 2: Run tests and verify missing worker behavior**

Run: `go test ./internal/locker/biz -run Reclaimer`

Expected: FAIL because `Reclaimer` does not exist.

- [ ] **Step 3: Implement bounded reclamation**

The reclaimer runs every 30 seconds, processes at most 100 expired rows per transaction, and continues until a batch returns fewer than 100. It releases only `LOCKED`, expired, unbound cells using reservation-key conditions. It logs counts, never reservation keys.

- [ ] **Step 4: Wire the API dependencies**

In `app/api/main.go`, load config, open clients, construct repositories/use cases/services, create the HTTP server, and defer client closure. Do not use a global service locator. `internal/server.NewHTTPServer` receives explicit service interfaces and readiness.

- [ ] **Step 5: Create the worker binary and image**

`app/worker/main.go` opens the same safe clients, creates only locker repository/reclaimer dependencies, and exits cleanly on SIGINT/SIGTERM. `deploy/Dockerfile.worker` mirrors the API multi-stage build but builds `./app/worker`.

- [ ] **Step 6: Run process tests and builds**

Run:

```bash
go test ./internal/locker/biz ./internal/server ./app/...
go build ./app/api ./app/worker
docker build -f deploy/Dockerfile.api -t smart-parcel-locker-api:m3 .
docker build -f deploy/Dockerfile.worker -t smart-parcel-locker-worker:m3 .
```

Expected: PASS.

- [ ] **Step 7: Commit process wiring**

```bash
git add app internal/server internal/locker/biz deploy/Dockerfile.worker
git commit -m "feat: wire api and reservation worker"
```

## Task 13: Complete Compose, integration smoke tests, and CI

**Files:**
- Modify: `deploy/docker-compose.yml`
- Modify: `.github/workflows/ci.yml`
- Create: `tests/integration/site_device_test.go`
- Create: `scripts/smoke_site_device.ps1`

- [ ] **Step 1: Write a failing end-to-end integration test**

The test starts from migrated seed data and verifies: list cities, query nearby sites, reserve a medium cell, replay the same key, send an online open-door command, switch the simulator to door-left-open, and reject an internal request without a token.

- [ ] **Step 2: Run the integration test before final wiring**

Run: `go test -tags=integration ./tests/integration`

Expected: FAIL until the compose environment and complete route wiring are active.

- [ ] **Step 3: Finish Compose dependencies**

Add API and worker environment variables using compose interpolation, health-gated MySQL/Redis dependencies, named volumes, restart policy, and localhost port publishing. Do not embed the production host, username, password, database, or Redis address.

- [ ] **Step 4: Add CI service containers and migration gate**

Update GitHub Actions to start MySQL 8.4 and Redis 7 services with CI-only credentials, apply up/down/up migrations, run unit/race tests, run tagged integration tests, validate Skills, and build both API and worker images.

- [ ] **Step 5: Add a Windows smoke script**

`scripts/smoke_site_device.ps1` loads only local `.env`, starts compose, waits on `/readyz`, runs the public and internal requests with a generated idempotency key, prints business IDs but not tokens/passwords, and exits nonzero on any unexpected status.

- [ ] **Step 6: Run the complete local smoke test**

Run:

```powershell
Copy-Item .env.example .env
docker compose -f deploy/docker-compose.yml up -d --build
make migrate-up
powershell -ExecutionPolicy Bypass -File scripts/smoke_site_device.ps1
go test -tags=integration -race ./...
```

Expected: PASS; API is ready, public queries work, internal auth is enforced, and reservation replay is stable.

- [ ] **Step 7: Commit integration delivery**

```bash
git add deploy/docker-compose.yml .github/workflows/ci.yml tests/integration scripts/smoke_site_device.ps1
git commit -m "test: verify site device integration flow"
```

## Task 14: Add and verify the project-specific site/device Skill

**Files:**
- Create: `.agents/skills/developing-site-device-domain/SKILL.md`
- Create: `.agents/skills/developing-site-device-domain/agents/openai.yaml`
- Create: `.agents/skills/developing-site-device-domain/references/invariants.md`
- Create: `docs/skill-tests/developing-site-device-domain/baseline.md`
- Create: `docs/skill-tests/developing-site-device-domain/with-skill.md`
- Modify: `scripts/test_validate_skills.py`

- [ ] **Step 1: Add a failing repository validation assertion**

Extend `scripts/test_validate_skills.py` to require the new Skill, its metadata, and references. Require rule text for MySQL source of truth, Redis non-authority, idempotency, internal token protection, production simulator exclusion, UTC, and secret redaction.

- [ ] **Step 2: Run validation and verify failure**

Run: `python scripts/test_validate_skills.py`

Expected: FAIL because the Skill package does not exist.

- [ ] **Step 3: Write the Skill and evidence**

The Skill instructs future agents to inspect schema/use cases before editing, preserve MySQL conditional allocation, treat Redis failure as degradation, require idempotency keys, never register simulator routes in production, avoid logging connection values, and run the exact domain/integration tests. Add baseline evidence showing an unsafe Redis-only proposal and with-Skill evidence showing the corrected MySQL transaction and security boundaries.

- [ ] **Step 4: Run all Skill validators**

Run:

```bash
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
```

Expected: PASS.

- [ ] **Step 5: Commit the project knowledge package**

```bash
git add .agents/skills/developing-site-device-domain docs/skill-tests/developing-site-device-domain scripts/test_validate_skills.py
git commit -m "feat: add site device development skill"
```

## Task 15: Document Navicat, local operations, and production isolation

**Files:**
- Modify: `README.md`
- Create: `docs/operations/local-mysql-redis.md`
- Create: `docs/operations/site-device-api.md`

- [ ] **Step 1: Write operational documentation**

Document local compose startup, migration commands, Navicat connection fields (`127.0.0.1`, port `3306`, values from local `.env`), named volume behavior, the destructive effect of `docker compose down -v`, Redis flush recovery expectations, public/internal curl examples, and simulator scenarios. State explicitly that the desktop production credential file is never used for development.

- [ ] **Step 2: Add a production safety checklist**

Require backup, migration preflight, manual approval, restricted MySQL account, Redis ACL/private network, TLS or SSH tunnel, IP allowlist, password rotation, and removal of plaintext desktop credentials before a future production deployment.

- [ ] **Step 3: Verify documentation contains no secrets or production endpoints**

Run a secret scanner over tracked files and manually inspect `git diff`. Expected: only placeholders and localhost examples are present.

- [ ] **Step 4: Commit operations documentation**

```bash
git add README.md docs/operations
git commit -m "docs: explain local site device operations"
```

## Task 16: Run final verification and prepare the milestone PR

**Files:**
- Modify only files required by verification findings.

- [ ] **Step 1: Regenerate and verify generated files**

Run:

```bash
make api
git diff --exit-code api/locker/v1
```

Expected: no generated-code diff.

- [ ] **Step 2: Run formatting, static analysis, unit, race, and integration tests**

Run:

```bash
test -z "$(gofmt -l app api internal tests)"
go mod verify
go vet ./...
go test ./...
go test -race ./...
go test -tags=integration -race ./...
```

Expected: every command exits 0.

- [ ] **Step 3: Validate rules, Skills, migrations, and images**

Run:

```bash
python scripts/test_agents_rules.py
python scripts/test_validate_skills.py
python scripts/validate_skills.py
make migrate-cycle
docker build -f deploy/Dockerfile.api -t smart-parcel-locker-api:m3 .
docker build -f deploy/Dockerfile.worker -t smart-parcel-locker-worker:m3 .
git diff --check
```

Expected: every command exits 0.

- [ ] **Step 4: Verify production secrets are absent**

Inspect `git status`, `git diff`, tracked environment files, test output, and built configuration. Confirm `.env` and the desktop credential file are untracked and that no production host, username, password, token, or Redis address appears in the branch.

- [ ] **Step 5: Commit any verification-only corrections**

If verification changes a tracked file, return to the task that owns that file, repeat its focused test, stage that task's explicit file list, and commit with the task's commit message. When verification produces no tracked changes, do not create an empty commit.

- [ ] **Step 6: Push and open a draft PR**

Push `codex/milestone-3-site-device` and create a draft pull request targeting `main`. The PR body lists MySQL/Redis isolation, public APIs, cell concurrency evidence, simulator scenarios, worker behavior, tests, and the explicit absence of production credentials.
