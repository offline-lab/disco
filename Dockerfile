FROM golang:1.24-trixie AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown

RUN mkdir -p build/bin && \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o build/bin/disco-daemon ./cmd/daemon && \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o build/bin/disco ./cmd/disco

FROM debian:trixie-slim

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
COPY --from=builder /build/build/bin/disco .
COPY libnss/nss_disco.c /build/

RUN gcc -fPIC -shared -o libnss_disco.so.2 -Wl,-soname,libnss_disco.so.2 /build/nss_disco.c && \
    cp libnss_disco.so.2 /lib/ && \
    ln -sf /lib/libnss_disco.so.2 /lib/libnss_disco.so && \
    ldconfig

COPY test/docker/config.yaml /etc/disco/config.yaml

RUN echo "hosts: files disco dns" > /etc/nsswitch.conf

EXPOSE 5354/udp

CMD ["./disco-daemon", "-config", "/etc/disco/config.yaml"]
