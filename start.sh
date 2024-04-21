#!/bin/sh
/app/tailscaled --tun=userspace-networking --socks5-server=localhost:1055 &
/app/tailscale up --authkey=${TAILSCALE_AUTHKEY} --hostname=overlandreceiver
echo Tailscale started
proxy=socks5://localhost:1055/
echo "ALL_PROXY=$proxy http_proxy=$proxy https_proxy=$proxy curl --verbose 100.101.105.83:"
ALL_PROXY=$proxy http_proxy=$proxy https_proxy=$proxy curl --veroise 100.101.105.83
ALL_PROXY=$proxy http_proxy=$proxy https_proxy=$proxy /app/overlandreceiver serve
>>>>>>> 1c69abb (proxy fighting and misc)
