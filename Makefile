.PHONY: all clean install uninstall man-pages install-man-pages uninstall-man-pages

BINARIES = nss-daemon nss-status nss-query nss-key nss-config-validate nss-ping nss-dns nss-announce
MANPAGES = nss-daemon.1 nss-query.1 nss-status.1 nss-key.1 nss-config-validate.1 nss-ping.1 nss-dns.1
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin
MANDIR = $(PREFIX)/share/man/man1
CONFIGDIR = /etc/nss-daemon

# Build all Go binaries
all: $(BINARIES)

nss-daemon:
	go build -ldflags="-s -w" -o $@ cmd/daemon/main.go

nss-status:
	go build -ldflags="-s -w" -o $@ cmd/status/main.go

nss-query:
	go build -ldflags="-s -w" -o $@ cmd/query/main.go

nss-key:
	go build -ldflags="-s -w" -o $@ cmd/key/main.go

nss-config-validate:
	go build -ldflags="-s -w" -o $@ cmd/config-validate/main.go

nss-dns:
	go build -ldflags="-s -w" -o $@ cmd/dns/main.go

nss-ping:
	go build -ldflags="-s -w" -o $@ cmd/ping/main.go

nss-announce:
	go build -ldflags="-s -w" -o $@ cmd/announce/main.go

# Build NSS module (Linux only with glibc)
libnss:
	@echo "Building NSS module..."
	@if command -v gcc >/dev/null 2>&1; then \
		if [ -f /usr/include/nss.h ]; then \
			gcc -fPIC -shared -o libnss_daemon.so.2 \
				-Wl,-soname,libnss_daemon.so.2 \
				-Wall -Wextra -Werror \
				-D_GNU_SOURCE \
				-I/usr/include \
				libnss/nss_daemon.c && \
			echo "✓ NSS module built successfully"; \
		else \
			echo "✗ NSS headers not found (requires Linux with glibc)"; \
		fi; \
	else \
		echo "✗ gcc not found"; \
	fi

man-pages: $(MANPAGES)

.1.md.1:
	@command -v pandoc >/dev/null 2>&1 || (echo "✗ pandoc not found. Install with: brew install pandoc (macOS) or apt install pandoc (Linux)" && exit 1)
	pandoc -s -t man $< -o $@

# Clean build artifacts
clean:
	rm -f $(BINARIES) libnss_daemon.so.2 libnss_daemon.so

# Install all components
install: all libnss install-man-pages
	@echo "Installing binaries to $(BINDIR)..."
	@mkdir -p $(BINDIR)
	@for bin in $(BINARIES); do \
		install -m 755 $$bin $(BINDIR)/ && echo "  ✓ $$bin"; \
	done

# Uninstall all components
uninstall:
	@echo "Removing binaries from $(BINDIR)..."
	@for bin in $(BINARIES); do \
		if [ -f $(BINDIR)/$$bin ]; then \
			rm -f $(BINDIR)/$$bin && echo "  ✓ $$bin"; \
		fi; \
	done
	@$(MAKE) uninstall-man-pages

# Install man pages
install-man-pages: man-pages
	@echo "Installing man pages to $(MANDIR)..."
	@mkdir -p $(MANDIR)
	@for man in $(MANPAGES); do \
		if [ -f $$man ]; then \
			install -m 644 $$man $(MANDIR)/ && echo "  ✓ $$man"; \
		fi; \
	done

# Uninstall man pages
uninstall-man-pages:
	@echo "Removing man pages from $(MANDIR)..."
	@for man in $(MANPAGES); do \
		if [ -f $(MANDIR)/$$man ]; then \
			rm -f $(MANDIR)/$$man && echo "  ✓ $$man"; \
		fi; \
	done

# Run tests
test:
	go test -v ./...

# Show help
help:
	@echo "NSS Daemon Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  all              Build all binaries (default)"
	@echo "  nss-daemon       Build daemon binary"
	@echo "  nss-status       Build status tool"
	@echo "  nss-query         Build query tool"
	@echo "  nss-key          Build key management tool"
	@echo "  nss-config-validate Build config validator"
	@echo "  libnss           Build NSS module (Linux only)"
	@echo "  man-pages        Generate man pages"
	@echo "  clean            Remove build artifacts"
	@echo "  install          Install all components (binaries + man pages)"
	@echo "  uninstall        Remove all installed components"
	@echo "  test             Run Go tests"
	@echo "  help             Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX          Installation prefix (default: /usr/local)"
	@echo "  BINDIR          Binary installation directory (default: PREFIX/bin)"
	@echo "  MANDIR          Man page installation directory (default: PREFIX/share/man/man1)"
	@echo ""
	@echo "Examples:"
	@echo "  make                     # Build all binaries"
	@echo "  make PREFIX=/usr install  # Install to /usr/bin"
	@echo "  make clean && make all     # Clean rebuild"

.PHONY: help
