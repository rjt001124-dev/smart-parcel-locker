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
