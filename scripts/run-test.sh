#!/usr/bin/env bash

topic() {
    local dtype=$1
    local device=$2
    local ctype=$3
    local channel=$4
    echo "\$${dtype}/${device}/${ctype}/${channel}"
}

payload() {
    method=$1

cat <<EOF
{"params":[],"method":"$method","jsonrpc":"2.0","time":1411702470066,"id":6132576}
EOF

}

main()

{
       _channel() {
	   local channel=$1
	   local device=${2:-${TEST_DEVICE}}

	   if test -n "$channel" 
	   then
	       if test -z "$device"
	       then
		   echo "export TEST_DEVICE={ninja-device-id}" 1>&2
		   exit 1
	       fi

	       mosquitto_pub -h ${TEST_HOST:-localhost} -t "$(topic device $device channel $channel)" -s
	   else
	       echo "usage: run-test.sh channel {channel} [{device}]" 1>&2
	       exit 1
	   fi
       }

       _driver() {
	   local node=$1
	   local driver=$2

	   mosquitto_pub -h ${TEST_HOST:-localhost} -t "$(topic node $node driver $driver)" -s
       }

       _payload() {
	   simple() {
	       local method=$1

cat <<EOF
{"params":[],"method":"$method","jsonrpc":"2.0","time":1411702470066,"id":6132576}
EOF
           }
	   
	   startPairing() {
	       local timeout=$1

cat <<EOF
{"params":[$timeout],"method":"startPairing","jsonrpc":"2.0","time":1411702470066,"id":6132576}
EOF
	   }
	   "$@"
       }

       cmd=$1
       shift 1
       _$cmd "$@"
}

main "$@"
