#!/bin/sh
/app/tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
/app/tailscale up --authkey=${TAILSCALE_AUTHKEY} --hostname=overlandreceiver
echo Tailscale started
ALL_PROXY=socks5h://localhost:1055/ /app/overlandreceiver serve
