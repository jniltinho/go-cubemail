## Variables for UPX
UPX_VERSION      := 5.1.1
UPX_ARCHIVE      := upx-$(UPX_VERSION)-amd64_linux.tar.xz
UPX_DIR          := upx-$(UPX_VERSION)-amd64_linux
UPX_BIN          := /usr/local/bin/upx
UPX_URL          := https://github.com/upx/upx/releases/download/v$(UPX_VERSION)/$(UPX_ARCHIVE)

## Variables for Go application
APP        := go-cubemail
BIN        := bin/$(APP)
PREFIX     := go-cubemail/cmd
VERSION    := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS    := -ldflags "-s -w \
	-X $(PREFIX).Version=$(VERSION) \
	-X $(PREFIX).BuildDate=$(BUILD_TIME) \
	-X $(PREFIX).GitCommit=$(GIT_COMMIT)"

.PHONY: all build build-prod run clean frontend frontend-dev dev \
        migrate tidy deps install-upx certs swagger help

## Default: build frontend + go binary
all: clean frontend build

## Build frontend (Vue 3 + Vite) to web/dist/
frontend:
	@echo "Building Vue 3 frontend..."
	cd frontend && npm run build

## Run Vite dev server (proxy to :8080)
frontend-dev:
	@echo "Starting Vite dev server on :5173 (proxy → :8080)..."
	cd frontend && npm run dev

## Development mode helper: starts Vite on :5173 (proxies /api to backend :8080).
## Start the Go backend separately in another terminal: go run . serve [--debug]
dev:
	@echo "Starting frontend dev server (:5173, API proxy → :8080)..."
	@echo "In another terminal, start backend: go run . serve --debug"
	cd frontend && npm run dev

## Build Go binary (requires web/dist/ to exist)
build:
	@echo "Building Go application..."
	CGO_ENABLED=1 go build -o $(BIN) $(LDFLAGS) .

## Full production build: frontend + Go + UPX compression
build-prod: clean frontend
	@echo "Building Go application (production)..."
	CGO_ENABLED=1 go build -o $(BIN) $(LDFLAGS) .
	upx --best --lzma $(BIN)

## Run the application binary
run:
	@echo "Starting application..."
	./$(BIN) serve

## Database migration
migrate:
	@echo "Running database migrations..."
	./$(BIN) migrate

## Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -f $(BIN)
	rm -rf web/dist

## Go module tidy
tidy:
	@echo "Tidying go modules..."
	go mod tidy

## Install Go dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download

## Install Node dependencies for frontend
deps-frontend:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

## Generate self-signed SSL certificates
certs:
	@echo "Generating self-signed SSL certificates..."
	mkdir -p ssl
	openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
		-keyout ssl/server.key -out ssl/server.crt \
		-subj "/C=BR/ST=SP/L=Sao Paulo/O=Development/CN=localhost"

install-upx:
	@echo "Installing UPX $(UPX_VERSION)..."
	curl -ksSL "$(UPX_URL)" -o "$(UPX_ARCHIVE)"
	tar -xf "$(UPX_ARCHIVE)"
	chmod +x "$(UPX_DIR)/upx"
	mv "$(UPX_DIR)/upx" "$(UPX_BIN)"
	rm -rf "$(UPX_DIR)" "$(UPX_ARCHIVE)"

## Generate Swagger documentation (swag init)
swagger:
	@echo "Generating Swagger documentation..."
	go run github.com/swaggo/swag/cmd/swag@latest init -g main.go --parseDependency --parseInternal
	@echo "Swagger docs generated in docs/"
	@echo "Don't forget to run 'make build' afterwards so the docs are embedded."

help:
	@echo "Makefile commands:"
	@echo "  all              - Clean, build frontend + Go binary"
	@echo "  frontend         - Build Vue 3 SPA to web/dist/"
	@echo "  frontend-dev     - Start Vite dev server (:5173)"
	@echo "  dev              - Start Vite dev server (:5173) + backend hint"
	@echo "  build            - Build Go binary (requires web/dist/)"
	@echo "  build-prod       - Full prod build with UPX compression"
	@echo "  run              - Run the application"
	@echo "  migrate          - Run database migrations"
	@echo "  clean            - Remove binary and web/dist"
	@echo "  tidy             - Run go mod tidy"
	@echo "  deps             - Download Go dependencies"
	@echo "  deps-frontend    - Install frontend npm packages"
	@echo "  certs            - Generate self-signed SSL certificates"
	@echo "  swagger          - Generate Swagger API documentation"
	@echo "  install-upx      - Download and install UPX binary"
