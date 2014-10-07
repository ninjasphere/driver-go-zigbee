#!/usr/bin/env bash
name=$1

tail +2 testdata/${name}.json | mosquitto_pub -t "$(head -1 testdata/${name}.json)" -s