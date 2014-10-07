#!/usr/bin/env bash

channel() {
    local device=$1
    local channel=$2
    echo "\$device/${device}/channel/${channel}"
}

payload() {
    method=$1

cat <<EOF
{"params":[],"method":"$method","jsonrpc":"2.0","time":1411702470066,"id":6132576}
EOF

}

run-test() {
    local channel=$1
    local method=$2
    local device=${3:-${TEST_DEVICE}}

    if test -n "$channel" 
    then
	if test -z "$device"
	then
	    echo "export TEST_DEVICE={ninja-device-id}" 1>&2
	    exit 1
	fi

	payload $method | mosquitto_pub -h ${TEST_HOST:-localhost} -t "$(channel $device $channel)" -s
    else
	echo "usage: run-test.sh channel method [device]" 1>&2
	exit 1
    fi
}

run-test "$@"
