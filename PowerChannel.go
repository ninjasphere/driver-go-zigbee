package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/davecgh/go-spew/spew"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type PowerChannel struct {
	Channel
}

func (c *PowerChannel) init() error {
	log.Printf("Initialising power channel of device %d", *c.device.deviceInfo.IeeeAddress)

	clusterID := ClusterIDPower
	instantaneousDemandAttributeID := uint32(0x0400)
	minReportInterval := uint32(1)
	maxReportInterval := uint32(120)
	reportableChange := uint32(1)

	request := &gateway.GwSetAttributeReportingReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
			EndpointId:  c.endpoint.EndpointId,
		},
		ClusterId: &clusterID,
		AttributeReportList: []*gateway.GwAttributeReportT{&gateway.GwAttributeReportT{
			AttributeId:       &instantaneousDemandAttributeID,
			AttributeType:     gateway.GwZclAttributeDataTypesT_ZCL_DATATYPE_INT24.Enum(),
			MinReportInterval: &minReportInterval,
			MaxReportInterval: &maxReportInterval,
			ReportableChange:  &reportableChange,
		}},
	}

	response := &gateway.GwSetAttributeReportingRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 20*time.Second)
	if err != nil {
		log.Printf("Error enabling power reporting: %s", err)
	} else if response.Status.String() != "STATUS_SUCCESS" {
		log.Printf("Failed to enable power reporting. status: %s", response.Status.String())
	}

	c.bus, _ = c.device.bus.AnnounceChannel("power", "power", []string{}, []string{"state"}, func(method string, payload *simplejson.Json) {
		log.Printf("Power got an unknown method %s", method)
	})

	if err != nil {
		log.Fatalf("Failed to announce power channel: %s", err)
	}

	go func() {
		for {
			log.Printf("Polling for power")
			err := c.fetchState()
			if err != nil {
				log.Printf("Failed to poll for power level %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

func (c *PowerChannel) fetchState() error {

	request := &gateway.DevGetPowerReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
	}

	response := &gateway.DevGetPowerRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting power level : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get power level. status: %s", response.Status.String())
	}

	log.Printf("Got power value %d", *response.PowerValue)

	payload := simplejson.New()
	payload.Set("value", *response.PowerValue)

	spew.Dump("THE BUS 2", c.bus)

	c.bus.SendEvent("state", payload)

	return nil
}
