-include .env

BUF_VERSION := v1.50.0
PROTOC_GEN_GO_VERSION := v1.36.6
PROTOC_GEN_GO_HTTP_VERSION := v2.0.0-20260404020628-f149714c1d54

MIGRATIONS := $(CURDIR)/migrations/mysql
APP_ENV ?= development
MYSQL_HOST ?= 127.0.0.1
MYSQL_PORT ?= 3306
MYSQL_DATABASE ?= smart_parcel_locker
MYSQL_USER ?= locker
MYSQL_PASSWORD ?= change-this-local-password
override MYSQL_URL := mysql://$(MYSQL_USER):$(MYSQL_PASSWORD)@tcp($(MYSQL_HOST):$(MYSQL_PORT))/$(MYSQL_DATABASE)?multiStatements=true&parseTime=true&loc=UTC

.PHONY: tools api-lint api-generate api db-up db-down guard-local confirm-local-reset migrate-up migrate-down migrate-reset-local migrate-cycle
tools:
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@$(PROTOC_GEN_GO_HTTP_VERSION)

api-lint:
	buf lint

api-generate:
	buf generate

api: api-lint api-generate

db-up:
	docker compose -f deploy/docker-compose.yml up -d mysql redis
db-down:
	docker compose -f deploy/docker-compose.yml down

guard-local:
	@case "$(MYSQL_HOST)" in 127.0.0.1|localhost) ;; *) echo "refusing migration: MYSQL_HOST must be 127.0.0.1 or localhost" >&2; exit 1 ;; esac
	@case "$(APP_ENV)" in development|test) ;; *) echo "refusing migration: APP_ENV must be development or test" >&2; exit 1 ;; esac

confirm-local-reset: guard-local
	@if [ "$(CONFIRM_LOCAL_RESET)" != "reset-local" ]; then echo "refusing reset: set CONFIRM_LOCAL_RESET=reset-local" >&2; exit 1; fi

migrate-up: guard-local
	@go run -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 -path="$(MIGRATIONS)" -database "$(MYSQL_URL)" up

migrate-down: guard-local
	@go run -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 -path="$(MIGRATIONS)" -database "$(MYSQL_URL)" down 1

migrate-reset-local: guard-local confirm-local-reset
	@go run -tags mysql github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3 -path="$(MIGRATIONS)" -database "$(MYSQL_URL)" down -all

migrate-cycle: guard-local confirm-local-reset
	@$(MAKE) --no-print-directory migrate-reset-local
	@$(MAKE) --no-print-directory migrate-up
