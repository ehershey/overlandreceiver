#!/bin/sh
/app/tailscaled --tun=userspace-networking &
/app/tailscale up --authkey=${TAILSCALE_AUTHKEY} --hostname=overlandreceiver
echo Tailscale started
nc -lk -p 8086 -e /app/tailscale nc 100.101.105.83 8086 &
nc -lk -p 12317 -e /app/tailscale nc 100.101.105.83 12317 &
/app/overlandreceiver serve
