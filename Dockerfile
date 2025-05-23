# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

WORKDIR /app

ARG TARGETARCH
ENV GOARCH=$TARGETARCH

COPY . .

RUN go mod download && \
  CGO_ENABLED=0 GOOS=linux GOARCH=$GOARCH go build -o bin/wiredoor

FROM alpine:3.21 AS production

WORKDIR /app

ENV WIREDOOR_URL="" \
  TOKEN=""

COPY /etc/wiredoor/config.ini.example /etc/wiredoor/config.ini
COPY connect-wiredoor /usr/bin/connect-wiredoor

RUN apk add --update iptables wireguard-tools tcpdump dnsmasq iproute2 \
  && ln -s /usr/bin/resolvectl /usr/local/bin/resolvconf \
  && chmod +x /usr/bin/connect-wiredoor

COPY --from=builder /app/bin/wiredoor /usr/bin/

CMD [ "/usr/bin/connect-wiredoor" ]
