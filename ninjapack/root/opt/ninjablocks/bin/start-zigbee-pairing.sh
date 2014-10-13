#!/usr/bin/env bash

die() {
    echo "$*" 1>&2
    exit 1
}

main()
{
    local pairingTime=$1
    local node=${2:-${DEVKIT_NODE:-$(sphere-config | grep nodeId | sed "s/.*://" | tr -d ' ,"')}}
    local sphere=${3:-${DEVKIT_HOST:-localhost}}

    test -n "$node" || die "usage: $0 {join-time-in-secs} {node} [sphere-ip]"
    (
	cat <<EOF
{"params":[$pairingTime],"method":"startPairing","jsonrpc":"2.0","time":1411702470066,"id":6132576}
EOF
    ) | mosquitto_pub -s -h "${sphere}" -t "\$node/$node/driver/com.ninjablocks.zigbee"
}

main "$@"