FROM golang:latest AS builder
WORKDIR /root
COPY . .
RUN go build main.go

FROM alpine:3.23 AS xray
ARG TARGETARCH
RUN apk add --no-cache curl unzip
COPY scripts/bundle-xray.sh /tmp/bundle-xray.sh
RUN mkdir /bundle && sh /tmp/bundle-xray.sh "$TARGETARCH" "/bundle/xray-linux-$TARGETARCH"

FROM debian:12-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates iproute2 \
    && apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
WORKDIR /root
COPY --from=builder  /root/main /root/x-ui
COPY bin/geoip.dat bin/geosite.dat /root/bin/
COPY --from=xray /bundle/ /root/bin/
VOLUME [ "/etc/x-ui" ]
CMD [ "./x-ui" ]