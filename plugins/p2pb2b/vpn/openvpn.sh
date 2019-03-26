#!/bin/bash
OPENVPN_LOCATION=$1
OPENVPN_OVPN=$2
OPENVPN_PORT=$3

echo ${OPENVPN_LOCATION}
echo ${OPENVPN_OVPN}
echo ${OPENVPN_PORT}

VPN_SCRATCH=/tmp/vpn_scratch

EXISTING=$(docker ps | grep ${OPENVPN_PORT} | awk '{print $1}')
[[ ! -z "$EXISTING" ]] && docker kill ${EXISTING} && docker rm ${EXISTING} || true

mkdir -p ${VPN_SCRATCH}
docker run --cap-add=NET_ADMIN --device=/dev/net/tun -d -v ${VPN_SCRATCH}:/data -e OPENVPN_PROVIDER=CUSTOM -e OPENVPN_CONFIG=${OPENVPN_LOCATION} -e OPENVPN_USERNAME=yszjar4353qszyx3gbq3xsvx -e OPENVPN_PASSWORD=v7v3ifv8swubvn6hkcdum4cb  -e WEBPROXY_ENABLED=false -e LOCAL_NETWORK=192.168.0.0/16 --log-driver json-file --log-opt max-size=10m -p ${OPENVPN_PORT}:3000 -v ${OPENVPN_OVPN}:/etc/openvpn/custom/default.ovpn ahfarrell/openvpn:0.1
sleep 5
