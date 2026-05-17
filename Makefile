## Variables for Tailwind CSS and UPX
TAILWIND_VERSION := v4.2.0
TAILWIND_BIN     := /usr/local/bin/tailwindcss
TAILWIND_URL     := https://github.com/tailwindlabs/tailwindcss/releases/download/$(TAILWIND_VERSION)/tailwindcss-linux-x64
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

.PHONY: all build build-prod run clean css watch-css migrate tidy deps \
        install-tailwind install-upx certs help

all: clean css build-prod

build: clean css
	@echo "Building Go application..."
	CGO_ENABLED=1 go build -o $(BIN) $(LDFLAGS) .

build-prod:
	@echo "Building Go application (production)..."
	CGO_ENABLED=1 go build -o $(BIN) $(LDFLAGS) .
	upx --best --lzma $(BIN)

run:
	@echo "Starting application..."
	./$(BIN) serve

css:
	@echo "Building CSS with Tailwind..."
	tailwindcss -i ./web/static/css/input.css -o ./web/static/css/style.css --minify

watch-css:
	@echo "Watching CSS changes..."
	tailwindcss -i ./web/static/css/input.css -o ./web/static/css/style.css --watch

migrate:
	@echo "Running database migrations..."
	./$(BIN) migrate

clean:
	@echo "Cleaning up..."
	rm -f $(BIN)
	rm -f web/static/css/style.css

tidy:
	@echo "Tidying go modules..."
	go mod tidy

deps:
	@echo "Installing Go dependencies..."
	go mod download

certs:
	@echo "Generating self-signed SSL certificates..."
	mkdir -p ssl
	openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
		-keyout ssl/server.key -out ssl/server.crt \
		-subj "/C=BR/ST=SP/L=Sao Paulo/O=Development/CN=localhost"

install-tailwind:
	@echo "Installing Tailwind CSS binary $(TAILWIND_VERSION)..."
	curl -ksSL "$(TAILWIND_URL)" -o tailwindcss-linux-x64
	chmod +x tailwindcss-linux-x64
	mv tailwindcss-linux-x64 "$(TAILWIND_BIN)"

install-upx:
	@echo "Installing UPX $(UPX_VERSION)..."
	curl -ksSL "$(UPX_URL)" -o "$(UPX_ARCHIVE)"
	tar -xf "$(UPX_ARCHIVE)"
	chmod +x "$(UPX_DIR)/upx"
	mv "$(UPX_DIR)/upx" "$(UPX_BIN)"
	rm -rf "$(UPX_DIR)" "$(UPX_ARCHIVE)"

help:
	@echo "Makefile commands:"
	@echo "  build            - Build the Go application (with CSS)"
	@echo "  build-prod       - Build + compress with UPX (production)"
	@echo "  run              - Run the application"
	@echo "  css              - Build CSS using Tailwind binary"
	@echo "  watch-css        - Watch for CSS changes"
	@echo "  migrate          - Run database migrations"
	@echo "  clean            - Remove binary and generated CSS"
	@echo "  tidy             - Run go mod tidy"
	@echo "  deps             - Download Go dependencies"
	@echo "  certs            - Generate self-signed SSL certificates"
	@echo "  install-tailwind - Download and install Tailwind CSS binary"
	@echo "  install-upx      - Download and install UPX binary"
