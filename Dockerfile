FROM golang:alpine as builder
WORKDIR /app
COPY . ./
RUN go build

# https://docs.docker.com/develop/develop-images/multistage-build/#use-multi-stage-builds
FROM alpine:latest
RUN apk update && apk add ca-certificates iptables ip6tables && rm -rf /var/cache/apk/*

# Copy binary to production image.
COPY --from=builder /app/start.sh /app/start.sh
COPY --from=builder /app/overlandreceiver /app/overlandreceiver

# Copy Tailscale binaries from the tailscale image on Docker Hub.
COPY --from=docker.io/tailscale/tailscale:stable /usr/local/bin/tailscaled /app/tailscaled
COPY --from=docker.io/tailscale/tailscale:stable /usr/local/bin/tailscale /app/tailscale
RUN mkdir -p /var/run/tailscale /var/cache/tailscale /var/lib/tailscale
RUN chmod 755 /app/start.sh /app/overlandreceiver

# Run on container startup.
CMD ["/app/start.sh"]
