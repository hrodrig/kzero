# kzero — build, test, release (GNU Make). On FreeBSD use gmake (pkg install gmake).
# GNU Make prefers GNUmakefile over Makefile; the Makefile stub forwards to gmake.

BINARY   := kzero
DIST     := dist
# BSD dist tarball arch (cross-compile). Examples: make dist-freebsd FREEBSD_ARCH=arm64
FREEBSD_ARCH ?= amd64
OPENBSD_ARCH ?= amd64
check-docker = @docker info >/dev/null 2>&1 || { echo "Error: Docker is not running. Start Docker and try again."; exit 1; }
GRYPE_FAIL_ON ?= high
# Minimum total statement coverage for `make cover-check` (see .cursor/rules/release-tests.mdc).
COVERAGE_MIN ?= 80
VERSION  ?= $(shell v=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); [ -n "$$v" ] && echo "v$$v" || echo "v0.2.0")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BRANCH   := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILDDATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -ldflags "-s -w -X github.com/hrodrig/kzero/internal/cli.Version=$(VERSION) -X github.com/hrodrig/kzero/internal/notify.AppVersion=$(VERSION) -X github.com/hrodrig/kzero/internal/cli.Commit=$(COMMIT) -X github.com/hrodrig/kzero/internal/cli.BuildDate=$(BUILDDATE) -X github.com/hrodrig/kzero/internal/cli.Branch=$(BRANCH)"
PORT_VERSION := $(shell cat VERSION 2>/dev/null | tr -d '\n\r' | sed 's/^v//')

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo "kzero — Kubernetes pipeline CLI"
	@echo ""
	@echo "  build           Build ./bin/kzero for current platform"
	@echo "  build-all       Cross-compile to $(DIST)/ (linux, darwin, windows, freebsd, openbsd)"
	@echo "  install         go install to \$$GOBIN"
	@echo "  install-man     Install man page to \$$MANDIR/man1 (default /usr/local/share/man)"
	@echo "  clean           Remove ./bin/kzero, coverage.out, and $(DIST)/"
	@echo "  test            Unit tests (go test ./...)"
	@echo "  cover           Unit tests with coverage.out"
	@echo "  cover-check     Fail if total statement coverage < $(COVERAGE_MIN)% (override: COVERAGE_MIN=70)"
	@echo "  lint            gofmt -s, go vet, gocyclo (<=14)"
	@echo "  lint-fix        gofmt -s -w"
	@echo "  tools           Install govulncheck and gocyclo to \$$GOBIN"
	@echo "  security        govulncheck ./..."
	@echo "  docker-build    Build container image kzero:local"
	@echo "  docker-scan     Build kzero:scan and run Grype (needs Docker)"
	@echo "  release-check   VERSION semver + lint + test + cover-check + security + docker-scan"
	@echo "  release         release-check then goreleaser (only from main)"
	@echo "  snapshot        Goreleaser snapshot to $(DIST)/ (no tag; includes .deb/.rpm/.tar.gz)"
	@echo "  dist-freebsd    Tarball for FreeBSD ports (default FREEBSD_ARCH=amd64)"
	@echo "  dist-openbsd    Tarball for OpenBSD ports (default OPENBSD_ARCH=amd64)"
	@echo "  port-freebsd-sync   Set PORTVERSION in contrib/freebsd/Makefile from VERSION"
	@echo "  port-openbsd-sync   Set DISTNAME/PKGNAME/MASTER_SITES/DISTFILES in contrib/openbsd/port/Makefile"
	@echo ""
	@echo "Current VERSION file: $$(cat VERSION 2>/dev/null | tr -d '\n\r' || echo '?')  (ldflags $(VERSION))"

.PHONY: build build-all install install-man clean test cover cover-check lint lint-fix tools security docker-build docker-scan release-check release snapshot dist-freebsd dist-openbsd port-freebsd-sync port-openbsd-sync

build:
	@mkdir -p bin
	go build -trimpath $(LDFLAGS) -o bin/$(BINARY) ./cmd/kzero

build-all:
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/kzero
	GOOS=linux GOARCH=arm64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 ./cmd/kzero
	GOOS=darwin GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64 ./cmd/kzero
	GOOS=darwin GOARCH=arm64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 ./cmd/kzero
	GOOS=windows GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-windows-amd64.exe ./cmd/kzero
	GOOS=freebsd GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-freebsd-amd64 ./cmd/kzero
	GOOS=freebsd GOARCH=arm64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-freebsd-arm64 ./cmd/kzero
	GOOS=openbsd GOARCH=amd64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-openbsd-amd64 ./cmd/kzero
	GOOS=openbsd GOARCH=arm64 go build -trimpath $(LDFLAGS) -o $(DIST)/$(BINARY)-openbsd-arm64 ./cmd/kzero

install:
	go install -trimpath $(LDFLAGS) ./cmd/kzero

MANDIR ?= /usr/local/share/man
.PHONY: install-man
install-man:
	@mkdir -p $(MANDIR)/man1
	@cp contrib/man/man1/kzero.1 $(MANDIR)/man1/
	@echo "Installed man page to $(MANDIR)/man1/kzero.1"

clean:
	rm -f bin/$(BINARY) coverage.out
	rm -rf $(DIST)

test:
	go test ./...

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -func=coverage.out | tail -1

cover-check:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@pct=$$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
	echo "Total statement coverage: $$pct% (minimum $(COVERAGE_MIN)%)"; \
	awk -v p="$$pct" -v m="$(COVERAGE_MIN)" 'BEGIN { if (p+0 < m+0) { print "Error: coverage is below " m "% — add tests or set COVERAGE_MIN="; exit 1 } }'

lint:
	@echo "Checking gofmt -s..."
	@unformatted=$$(gofmt -s -l .); [ -z "$$unformatted" ] || { echo "Files not formatted (run: make lint-fix):"; echo "$$unformatted"; exit 1; }
	@echo "Running go vet..."
	@go vet ./...
	@echo "Running gocyclo (complexity <= 14)..."
	@go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	@"$(shell go env GOPATH)/bin/gocyclo" -over 14 .

lint-fix:
	gofmt -s -w .

tools:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

security:
	@echo "Running govulncheck..."
	@tmp=$$(mktemp); \
	go run golang.org/x/vuln/cmd/govulncheck@latest ./... >"$$tmp" 2>&1 || true; \
	output=$$(grep -v '^exit status [0-9]*$$' "$$tmp" || true); \
	rm -f "$$tmp"; \
	ignored_ids=$$(grep 'id:' .govulncheck-ignore.yaml 2>/dev/null | cut -d'"' -f2); \
	total_vulns=$$(echo "$$output" | grep -c 'Vulnerability #' || true); \
	matching=0; \
	for id in $$ignored_ids; do \
		c=$$(echo "$$output" | grep -c "^Vulnerability.*$$id" || true); \
		matching=$$((matching + c)); \
	done; \
	if [ $$total_vulns -eq 0 ]; then \
		echo "$$output"; \
		echo ""; \
		echo "=== security: PASS (govulncheck clean) ==="; \
	elif [ $$total_vulns -eq $$matching ]; then \
		echo "$$output"; \
		echo ""; \
		echo "=== security: PASS (known false positives only) ==="; \
		echo "govulncheck reported $$matching advisories filtered — see .govulncheck-ignore.yaml (containerd v2-only CRI, Helm openpgp)."; \
		echo "Pending upstream: Go vulndb module-path correction; no kzero release action until vulndb or helm/containerd graph changes."; \
		echo "Policy: .govulncheck-ignore.yaml"; \
	else \
		echo "$$output"; \
		echo ""; \
		echo "ERROR: $$((total_vulns - matching)) unfiltered govulncheck finding(s)."; \
		echo "Add to .govulncheck-ignore.yaml only with documented false-positive rationale."; \
		exit 1; \
	fi

docker-build:
	$(check-docker)
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILDDATE=$(BUILDDATE) --build-arg BRANCH=$(BRANCH) -t kzero:local .

docker-scan:
	$(check-docker)
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILDDATE=$(BUILDDATE) --build-arg BRANCH=$(BRANCH) -t kzero:scan .
	@if command -v grype >/dev/null 2>&1; then \
		grype kzero:scan -c .grype.yaml --fail-on $(GRYPE_FAIL_ON); \
	else \
		echo "grype not on PATH; using anchore/grype container..."; \
		docker run --rm --pull=always -v /var/run/docker.sock:/var/run/docker.sock -v "$(CURDIR)/.grype.yaml:/.grype.yaml:ro" anchore/grype:latest \
			kzero:scan -c /.grype.yaml --fail-on $(GRYPE_FAIL_ON); \
	fi

.PHONY: release-check
release-check:
	$(check-docker)
	@set -e; \
	test -f VERSION || { echo "Error: VERSION file is required"; exit 1; }; \
	ver_raw=$$(cat VERSION | tr -d '\n\r'); 	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	echo "Release version: $$ver (tag v$$ver)"; \
	man_ver=$$(sed -n 's/^\.TH KZERO 1 "[^"]*" "\([^"]*\)".*/\1/p' contrib/man/man1/kzero.1 | head -1); \
	expect_ver="kzero v$$ver"; \
	test "$$man_ver" = "$$expect_ver" || { echo "Error: contrib/man/man1/kzero.1 .TH version ($$man_ver) must match VERSION ($$expect_ver)"; exit 1; };
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) cover-check
	@$(MAKE) security
	@$(MAKE) docker-scan
	@echo "All release checks passed."

release: release-check
	$(check-docker)
	@branch=$$(git branch --show-current 2>/dev/null); \
	if [ "$$branch" != "main" ]; then \
	  echo "Error: release only from main (current: $$branch). Merge develop → main first."; \
	  exit 1; \
	fi; \
	goreleaser release --clean

snapshot:
	@ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required for snapshot"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	KZERO_SNAPSHOT_VERSION="$$ver-next" goreleaser release --snapshot --clean

.PHONY: port-freebsd-sync
port-freebsd-sync:
	@[ -n "$(PORT_VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@sed -i.bak "s/^PORTVERSION=.*/PORTVERSION=\t$(PORT_VERSION)/" contrib/freebsd/Makefile
	@rm -f contrib/freebsd/Makefile.bak
	@echo "Updated contrib/freebsd/Makefile PORTVERSION to $(PORT_VERSION)"

.PHONY: port-openbsd-sync
port-openbsd-sync:
	@[ -n "$(PORT_VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@test -f contrib/openbsd/port/Makefile || { echo "Error: contrib/openbsd/port/Makefile not found"; exit 1; }
	@sed -i.bak \
	  -e 's#^DISTNAME =.*#DISTNAME =	kzero_v$(PORT_VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}#' \
	  -e 's#^PKGNAME =.*#PKGNAME =	kzero-$(PORT_VERSION)#' \
	  -e 's#^MASTER_SITES =.*#MASTER_SITES =	https://github.com/hrodrig/kzero/releases/download/v$(PORT_VERSION)/#' \
	  -e 's#^DISTFILES =.*#DISTFILES =	kzero_v$(PORT_VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz#' \
	  contrib/openbsd/port/Makefile
	@rm -f contrib/openbsd/port/Makefile.bak
	@echo "Updated contrib/openbsd/port/Makefile to $(PORT_VERSION)"

.PHONY: dist-freebsd
dist-freebsd:
	@set -e; \
	ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	echo "$(FREEBSD_ARCH)" | grep -qE '^(amd64|arm64)$$' || { echo "Error: FREEBSD_ARCH must be amd64 or arm64"; exit 1; }; \
	arch="$(FREEBSD_ARCH)"; \
	out="$(DIST)/kzero_v$${ver}_freebsd_$$arch.tar.gz"; \
	stage="/tmp/kzero-dist-root-$$PPID"; \
	tmpbin="$(DIST)/kzero-freebsd-$$arch-$$PPID"; \
	echo "Building kzero for FreeBSD $$arch with VERSION=v$$ver..."; \
	mkdir -p "$(DIST)"; \
	GOOS=freebsd GOARCH="$$arch" go build -trimpath $(LDFLAGS) -o "$$tmpbin" ./cmd/kzero; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/man/man1" "$$stage/share/doc/kzero" "$$stage/share/examples/kzero"; \
	cp "$$tmpbin" "$$stage/kzero"; \
	rm -f "$$tmpbin"; \
	cp LICENSE "$$stage/share/doc/kzero/LICENSE"; \
	cp configs/kzero.sample.yml "$$stage/share/examples/kzero/kzero.sample.yml"; \
	cp contrib/man/man1/kzero.1 "$$stage/share/man/man1/kzero.1"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"

.PHONY: dist-openbsd
dist-openbsd:
	@set -e; \
	ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	echo "$(OPENBSD_ARCH)" | grep -qE '^(amd64|arm64)$$' || { echo "Error: OPENBSD_ARCH must be amd64 or arm64"; exit 1; }; \
	arch="$(OPENBSD_ARCH)"; \
	out="$(DIST)/kzero_v$${ver}_openbsd_$$arch.tar.gz"; \
	stage="/tmp/kzero-openbsd-dist-root-$$PPID"; \
	tmpbin="$(DIST)/kzero-openbsd-$$arch-$$PPID"; \
	echo "Building kzero for OpenBSD $$arch with VERSION=v$$ver..."; \
	mkdir -p "$(DIST)"; \
	GOOS=openbsd GOARCH="$$arch" go build -trimpath $(LDFLAGS) -o "$$tmpbin" ./cmd/kzero; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/man/man1" "$$stage/share/doc/kzero" "$$stage/share/examples/kzero"; \
	cp "$$tmpbin" "$$stage/kzero"; \
	rm -f "$$tmpbin"; \
	cp LICENSE "$$stage/share/doc/kzero/LICENSE"; \
	cp configs/kzero.sample.yml "$$stage/share/examples/kzero/kzero.sample.yml"; \
	cp contrib/man/man1/kzero.1 "$$stage/share/man/man1/kzero.1"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"
