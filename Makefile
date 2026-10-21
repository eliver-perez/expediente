GO ?= go

.PHONY: deps frontend build build-dev test check test-e2e installer-macos installer-ubuntu
deps:
	$(GO) mod download
	npm --prefix web ci

frontend:
	npm --prefix web run build

build: frontend
	$(GO) build -trimpath -o build/aidex-server ./cmd/gestor-documental

build-dev: frontend
	$(GO) build -tags development -trimpath -o build/aidex-server-dev ./cmd/gestor-documental

test:
	$(GO) test -race ./...
	$(GO) test -race -tags development ./...

check: frontend
	python3 scripts/check_format.py
	$(GO) vet ./...
	$(GO) test -race ./...
	$(GO) test -race -tags development ./...
	python3 scripts/check_design.py

test-e2e: frontend
	npm --prefix web run test:e2e

# AIDEX installers: production identity/licensing; release signing is separate.
installer-macos: frontend
	python3 scripts/build_installers.py --target macos

installer-ubuntu: frontend
	python3 scripts/build_installers.py --target ubuntu
