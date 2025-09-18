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

RUN apk add --update iptables wireguard-tools tcpdump dnsmasq iproute2 libcap sudo \
  && ln -s /usr/bin/resolvectl /usr/local/bin/resolvconf \
  && addgroup -g 1000 wiredoor \
  && adduser -S -u 1000 -G wiredoor -H -s /sbin/nologin wiredoor

COPY --chown=wiredoor:wiredoor /etc/wiredoor/config.ini.example /etc/wiredoor/config.ini

COPY connect-wiredoor /usr/bin/connect-wiredoor

COPY --chown=wiredoor:wiredoor --from=builder /app/bin/wiredoor /usr/bin/

RUN setcap 'cap_net_bind_service=+ep' /usr/sbin/dnsmasq \
  && echo 'wiredoor ALL=(root) NOPASSWD: /usr/bin/wiredoor, /usr/bin/wg-quick, /usr/bin/wg, /sbin/ip, /usr/sbin/iptables, /usr/sbin/ip6tables, /usr/bin/tcpdump, /usr/sbin/xtables-nft-multi' > /etc/sudoers.d/wiredoor \
  && chmod +x /usr/bin/connect-wiredoor

USER wiredoor

CMD [ "/usr/bin/connect-wiredoor" ]
