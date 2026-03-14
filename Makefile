.PHONY: all clean install uninstall man-pages install-man-pages uninstall-man-pages test help cross-compile

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

BUILDDIR := build
BINDIR := $(BUILDDIR)/bin
LIBDIR := $(BUILDDIR)/lib

BINARIES := disco disco-daemon
TOOLS := disco-gps-broadcaster
MANPAGES := disco.1 disco-daemon.1

PREFIX ?= /usr/local
INSTALL_BINDIR := $(PREFIX)/bin
INSTALL_MANDIR := $(PREFIX)/share/man/man1
INSTALL_LIBDIR := /lib/x86_64-linux-gnu
CONFIGDIR := /etc/disco

all: $(BINDIR) $(LIBDIR) $(BINARIES) $(TOOLS)

$(BINDIR):
	@mkdir -p $(BINDIR)

$(LIBDIR):
	@mkdir -p $(LIBDIR)

$(BINARIES) $(TOOLS): | $(BINDIR)

disco:
	go build $(LDFLAGS) -o $(BINDIR)/$@ cmd/disco/main.go

disco-daemon:
	go build $(LDFLAGS) -o $(BINDIR)/$@ cmd/daemon/main.go

disco-gps-broadcaster:
	go build $(LDFLAGS) -o $(BINDIR)/$@ cmd/gps-broadcaster/main.go

libnss: | $(LIBDIR)
	@echo "Building NSS module (libnss_disco)..."
	@if command -v gcc >/dev/null 2>&1; then \
		if [ -f /usr/include/nss.h ]; then \
			gcc -fPIC -shared -o $(LIBDIR)/libnss_disco.so.2 \
				-Wl,-soname,libnss_disco.so.2 \
				-Wall -Wextra -Werror \
				-D_GNU_SOURCE \
				-I/usr/include \
				libnss/nss_disco.c && \
			echo "✓ NSS module built: $(LIBDIR)/libnss_disco.so.2"; \
		else \
			echo "✗ NSS headers not found (requires Linux with glibc)"; \
		fi; \
	else \
		echo "✗ gcc not found"; \
	fi

man-pages: $(MANPAGES)

.1.md.1:
	@command -v pandoc >/dev/null 2>&1 || (echo "✗ pandoc not found" && exit 1)
	pandoc -s -t man $< -o $@

clean:
	rm -rf $(BUILDDIR)
	rm -f $(MANPAGES)

install: all libnss install-man-pages
	@echo "Installing binaries to $(INSTALL_BINDIR)..."
	@mkdir -p $(INSTALL_BINDIR)
	@for bin in $(BINARIES) $(TOOLS); do \
		if [ -f $(BINDIR)/$$bin ]; then \
			install -m 755 $(BINDIR)/$$bin $(INSTALL_BINDIR)/ && echo "  ✓ $$bin"; \
		fi; \
	done
	@if [ -f $(LIBDIR)/libnss_disco.so.2 ]; then \
		echo "Installing NSS module..."; \
		install -m 755 $(LIBDIR)/libnss_disco.so.2 $(INSTALL_LIBDIR)/ 2>/dev/null || \
		install -m 755 $(LIBDIR)/libnss_disco.so.2 /lib64/ 2>/dev/null || \
		echo "  ✗ Could not install NSS module (run as root)"; \
		ldconfig 2>/dev/null || true; \
	fi

uninstall:
	@echo "Removing binaries from $(INSTALL_BINDIR)..."
	@for bin in $(BINARIES) $(TOOLS); do \
		if [ -f $(INSTALL_BINDIR)/$$bin ]; then \
			rm -f $(INSTALL_BINDIR)/$$bin && echo "  ✓ $$bin"; \
		fi; \
	done
	@rm -f $(INSTALL_LIBDIR)/libnss_disco.so.2 /lib64/libnss_disco.so.2 2>/dev/null
	@$(MAKE) uninstall-man-pages

install-man-pages: man-pages
	@echo "Installing man pages to $(INSTALL_MANDIR)..."
	@mkdir -p $(INSTALL_MANDIR)
	@for man in $(MANPAGES); do \
		if [ -f $$man ]; then \
			install -m 644 $$man $(INSTALL_MANDIR)/ && echo "  ✓ $$man"; \
		fi; \
	done

uninstall-man-pages:
	@echo "Removing man pages from $(INSTALL_MANDIR)..."
	@for man in $(MANPAGES); do \
		if [ -f $(INSTALL_MANDIR)/$$man ]; then \
			rm -f $(INSTALL_MANDIR)/$$man && echo "  ✓ $$man"; \
		fi; \
	done

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	@command -v golangci-lint >/dev/null 2>&1 || (echo "✗ golangci-lint not found" && exit 1)
	golangci-lint run ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BINDIR)
	@echo "  linux/amd64..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/disco-daemon-linux-amd64 cmd/daemon/main.go
	@echo "  linux/arm64 (Pi 4, Pi Zero 2W)..."
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/disco-daemon-linux-arm64 cmd/daemon/main.go
	@echo "  linux/arm (Pi Zero)..."
	@GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/disco-daemon-linux-arm cmd/daemon/main.go
	@echo "  GPS broadcaster for ARM..."
	@GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/disco-gps-broadcaster-linux-arm cmd/gps-broadcaster/main.go
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINDIR)/disco-gps-broadcaster-linux-arm64 cmd/gps-broadcaster/main.go
	@echo "✓ Cross-compilation complete. Binaries in $(BINDIR)/"

help:
	@echo "Disco - Service Discovery Daemon"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build all binaries (default)"
	@echo "  libnss           Build NSS module (Linux only)"
	@echo "  man-pages        Generate man pages"
	@echo "  clean            Remove build artifacts"
	@echo "  install          Install all components"
	@echo "  uninstall        Remove all installed components"
	@echo "  test             Run Go tests with race detection"
	@echo "  test-coverage    Run tests and generate coverage report"
	@echo "  lint             Run golangci-lint"
	@echo "  fmt              Format Go code"
	@echo "  vet              Run go vet"
	@echo "  cross-compile    Build for multiple platforms"
	@echo "  help             Show this help"
	@echo ""
	@echo "Build output:"
	@echo "  Binaries:  $(BINDIR)/"
	@echo "  Libraries: $(LIBDIR)/"
	@echo ""
	@echo "Binaries:"
	@echo "  disco                    Unified CLI tool"
	@echo "  disco-daemon             Main daemon"
	@echo "  disco-gps-broadcaster    GPS time broadcaster"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION    Build version (default: git tag or 'dev')"
	@echo "  COMMIT     Git commit (default: git HEAD)"
	@echo "  PREFIX     Installation prefix (default: /usr/local)"
	@echo ""
	@echo "Examples:"
	@echo "  make                           # Build all binaries"
	@echo "  make VERSION=v1.0.0 all        # Build with specific version"
	@echo "  make PREFIX=/usr install       # Install to /usr/bin"
	@echo "  make cross-compile             # Build for all platforms"
	@echo "  ./build/bin/disco-daemon       # Run built binary"
