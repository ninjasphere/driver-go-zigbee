package main

import (
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-zigbee/nwkmgr"
)

type OnOffSwitchCluster struct {
	Channel
	SendEvent func(event string, payload interface{}) error
}

func (c *OnOffSwitchCluster) SetEventHandler(handler func(event string, payload interface{}) error) {
	c.SendEvent = handler
}

func (c *OnOffSwitchCluster) GetProtocol() string {
	return "button-momentary"
}

func (c *OnOffSwitchCluster) init() error {
	log.Debugf("Initialising on/off button cluster of device %d", *c.device.deviceInfo.IeeeAddress)

	clusterID := uint32(0x06)

	dstEndpoint := uint32(5)

	bindReq := &nwkmgr.NwkSetBindingEntryReq{
		SrcAddr: &nwkmgr.NwkAddressStructT{
			AddressType: nwkmgr.NwkAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
			EndpointId:  c.endpoint.EndpointId,
		},
		ClusterId: &clusterID,
		DstAddr: &nwkmgr.NwkAddressStructT{
			AddressType: nwkmgr.NwkAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.driver.localDevice.IeeeAddress,
			EndpointId:  &dstEndpoint,
		},
		BindingMode: nwkmgr.NwkBindingModeT_BIND.Enum(),
	}

	log.Infof("Binding on-off cluster %v", bindReq)

	bindRes := &nwkmgr.NwkSetBindingEntryRspInd{}

	err := c.device.driver.nwkmgrConn.SendAsyncCommand(bindReq, bindRes, time.Second*10)
	if err != nil {
		log.Errorf("Error binding on/off cluster: %s", err)
	} else if bindRes.Status.String() != "STATUS_SUCCESS" {
		log.Errorf("Failed to bind on/off cluster. status: %s", bindRes.Status.String())
	}

	update := c.device.driver.gatewayConn.OnBoundCluster(*c.device.deviceInfo.IeeeAddress, *c.endpoint.EndpointId, clusterID)

	go func() {
		for {
			state := <-update

			spew.Dump("Incoming on/off state:", state)

			c.SendEvent("pressed", true)
		}
	}()

	err = c.device.driver.Conn.ExportChannel(c.device, c, c.ID)
	if err != nil {
		log.Fatalf("Failed to announce on/off switch channel: %s", err)
	}

	return nil

}
