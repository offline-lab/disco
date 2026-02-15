FROM golang:1.24-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o nss-daemon cmd/daemon/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o nss-query cmd/query/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o nss-status cmd/status/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o nss-ping cmd/ping/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o nss-announce cmd/announce/main.go

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y \
    build-essential \
    libc6-dev \
    iputils-ping \
    curl \
    netcat-openbsd \
    tcpdump \
    dnsutils \
    iproute2 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /run /etc/nss-daemon

WORKDIR /app

COPY --from=builder /build/nss-daemon .
COPY --from=builder /build/nss-query .
COPY --from=builder /build/nss-status .
COPY --from=builder /build/nss-ping .
COPY --from=builder /build/nss-announce .
COPY libnss/nss_daemon.c /build/

RUN gcc -fPIC -shared -o libnss_daemon.so.2 -Wl,-soname,libnss_daemon.so.2 /build/nss_daemon.c && \
    cp libnss_daemon.so.2 /lib/x86_64-linux-gnu/ && \
    ln -sf /lib/x86_64-linux-gnu/libnss_daemon.so.2 /lib/x86_64-linux-gnu/libnss_daemon.so && \
    ldconfig

COPY test/docker/config.yaml /etc/nss-daemon/config.yaml

RUN echo "hosts: files daemon dns" > /etc/nsswitch.conf

EXPOSE 5354/udp

CMD ["./nss-daemon", "-config", "/etc/nss-daemon/config.yaml"]
