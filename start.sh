#!/bin/sh
DBHOST=100.71.190.118
/app/tailscaled --tun=userspace-networking &
/app/tailscale up --authkey=${TAILSCALE_AUTHKEY} --hostname=overlandreceiver
echo Tailscale started
nc -lk -p 8086 -e /app/tailscale nc "${DBHOST}" 8086 &
nc -lk -p 12317 -e /app/tailscale nc "${DBHOST}" 12317 &
/app/overlandreceiver serve
