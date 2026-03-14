FROM golang:1.24-bookworm AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

RUN mkdir -p build/bin build/lib && \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o build/bin/disco-daemon cmd/daemon/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-query cmd/query/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-status cmd/status/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-ping cmd/ping/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-announce cmd/announce/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-time cmd/time/main.go && \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o build/bin/disco-timeset cmd/timeset/main.go

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
    && mkdir -p /run /etc/disco

WORKDIR /app

COPY --from=builder /build/build/bin/disco-daemon .
COPY --from=builder /build/build/bin/disco-query .
COPY --from=builder /build/build/bin/disco-status .
COPY --from=builder /build/build/bin/disco-ping .
COPY --from=builder /build/build/bin/disco-announce .
COPY --from=builder /build/build/bin/disco-time .
COPY --from=builder /build/build/bin/disco-timeset .
COPY libnss/nss_disco.c /build/

RUN gcc -fPIC -shared -o libnss_disco.so.2 -Wl,-soname,libnss_disco.so.2 /build/nss_disco.c && \
    cp libnss_disco.so.2 /lib/x86_64-linux-gnu/ && \
    ln -sf /lib/x86_64-linux-gnu/libnss_disco.so.2 /lib/x86_64-linux-gnu/libnss_disco.so && \
    ldconfig

COPY test/docker/config.yaml /etc/disco/config.yaml

RUN echo "hosts: files disco dns" > /etc/nsswitch.conf

EXPOSE 5354/udp

CMD ["./disco-daemon", "-config", "/etc/disco/config.yaml"]
