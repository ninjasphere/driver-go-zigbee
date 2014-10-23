package main

import "github.com/ninjasphere/go-zigbee/nwkmgr"

type Channel struct {
	device   *Device
	endpoint *nwkmgr.NwkSimpleDescriptorT
}
