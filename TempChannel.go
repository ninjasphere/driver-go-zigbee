package main

import (
	"fmt"
	"time"

	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type TempChannel struct {
	Channel
	channel *channels.TemperatureChannel
}

func (c *TempChannel) init() error {
	log.Debugf("Initialising Temp channel of device %d", *c.device.deviceInfo.IeeeAddress)

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
		log.Errorf("Error enabling Temp reporting: %s", err)
	} else if response.Status.String() != "STATUS_SUCCESS" {
		log.Errorf("Failed to enable Temp reporting. status: %s", response.Status.String())
	}

	c.channel = channels.NewTemperatureChannel(c)
	err = c.device.driver.Conn.ExportChannel(c.device, c.channel, "temperature")
	if err != nil {
		log.Fatalf("Failed to announce temperature channel: %s", err)
	}

	go func() {
		for {
			err := c.fetchState()
			if err != nil {
				log.Errorf("Failed to poll for Temperature %s", err)
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

	log.Debugf("Got Temp value %d", *response.TemperatureValue)

	c.channel.SendState(float64(*response.TemperatureValue) / 100)

	return nil
}
