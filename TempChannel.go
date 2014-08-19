package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type TempChannel struct {
	Channel
}

func (c *TempChannel) init() error {
	log.Printf("Initialising Temp channel of device %d", *c.device.deviceInfo.IeeeAddress)

	clusterID := ClusterIDTemp
	instantaneousDemandAttributeID := uint32(0x0400)
	minReportInterval := uint32(10)
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
		log.Printf("Error enabling Temp reporting: %s", err)
	} else if response.Status.String() != "STATUS_SUCCESS" {
		log.Printf("Failed to enable Temp reporting. status: %s", response.Status.String())
	}

	c.bus, err = c.device.bus.AnnounceChannel("temperature", "temperature", []string{}, []string{"state"}, func(method string, payload *simplejson.Json) {
		log.Printf("Temp got an unknown method %s", method)
	})

	if err != nil {
		log.Fatalf("Failed to announce temperature channel: %s", err)
	}

	go func() {
		for {
			err := c.fetchState()
			if err != nil {
				log.Printf("Failed to poll for Temperature %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

func (c *TempChannel) fetchState() error {

	request := &gateway.DevGetTempReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
	}

	response := &gateway.DevGetTempRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting Temp level : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get Temp level. status: %s", response.Status.String())
	}

	log.Printf("Got Temp value %d", *response.TemperatureValue)

	payload := simplejson.New()
	payload.Set("value", float64(*response.TemperatureValue)/100)

	c.bus.SendEvent("state", payload)

	return nil
}
