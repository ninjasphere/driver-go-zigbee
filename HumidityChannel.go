package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type HumidityChannel struct {
	Channel
	channel *channels.HumidityChannel
}

func (c *HumidityChannel) init() error {
	log.Printf("Initialising Humidity channel of device %d", *c.device.deviceInfo.IeeeAddress)

	clusterID := ClusterIDHumidity
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
		AttributeReportList: []*gateway.GwAttributeReportT{{
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
		log.Printf("Error enabling Humidity reporting: %s", err)
	} else if response.Status.String() != "STATUS_SUCCESS" {
		log.Printf("Failed to enable Humidity reporting. status: %s", response.Status.String())
	}

	c.channel = channels.NewHumidityChannel(c)
	err = c.device.conn.ExportChannel(c.device, c.channel, "humidity")
	if err != nil {
		log.Printf("failed to export humidity channel")
	}

	if err != nil {
		log.Fatalf("Failed to announce Humidity channel: %s", err)
	}

	go func() {
		for {
			err := c.fetchState()
			if err != nil {
				log.Printf("Failed to poll for Humidity %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil
}

func (c *HumidityChannel) fetchState() error {

	request := &gateway.DevGetHumidityReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
	}

	response := &gateway.DevGetHumidityRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting Humidity level : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get Humidity level. status: %s", response.Status.String())
	}

	log.Printf("Got Humidity value %d", *response.HumidityValue)

	c.channel.SendState(float64(*response.HumidityValue)/0x2710)

	return nil
}
