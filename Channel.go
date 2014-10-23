package main

import "github.com/ninjasphere/go-zigbee/nwkmgr"

type Channel struct {
	ID       string
	device   *Device
	endpoint *nwkmgr.NwkSimpleDescriptorT
}
