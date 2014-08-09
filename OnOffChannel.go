package main

import (
	"fmt"
	"log"
	"time"

	"github.com/ninjasphere/go-zigbee/gateway"
)

type OnOffChannel struct {
	Channel
}

func (c *OnOffChannel) init() error {
	log.Printf("Initialising on/off channel of device %d", *c.device.deviceInfo.IeeeAddress)

	clusterID := uint32(0x06)
	attributeID := uint32(0)
	minReportInterval := uint32(1)
	maxReportInterval := uint32(120)

	request := &gateway.GwSetAttributeReportingReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
		ClusterId: &clusterID,
		AttributeReportList: []*gateway.GwAttributeReportT{&gateway.GwAttributeReportT{
			AttributeId:       &attributeID,
			AttributeType:     gateway.GwZclAttributeDataTypesT_ZCL_DATATYPE_BOOLEAN.Enum(),
			MinReportInterval: &minReportInterval,
			MaxReportInterval: &maxReportInterval,
		}},
	}

	response := &gateway.GwSetAttributeReportingRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Error enabling on/off reporting: %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to enable on/off reporting. status: %s", response.Status.String())
	}

	return nil
}

func (c *OnOffChannel) turnOn() error {
	return c.setState(gateway.GwOnOffStateT_ON_STATE.Enum())
}

func (c *OnOffChannel) turnOff() error {
	return c.setState(gateway.GwOnOffStateT_OFF_STATE.Enum())
}

func (c *OnOffChannel) toggle() error {
	return c.setState(gateway.GwOnOffStateT_TOGGLE_STATE.Enum())
}

func (c *OnOffChannel) setState(state *gateway.GwOnOffStateT) error {
	request := &gateway.DevSetOnOffStateReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
		State: gateway.GwOnOffStateT_TOGGLE_STATE.Enum(),
	}

	response := &gateway.GwZigbeeGenericRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Error setting on/off state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to set on/off state. status: %s", response.Status.String())
	}

	return nil
}
