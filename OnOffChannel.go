package main

import (
	"fmt"
	"log"

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

	reportingRequest := &gateway.GwSetAttributeReportingReq{
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

	confirmation := &gateway.GwZigbeeGenericCnf{}

	err := c.device.driver.gatewayConn.SendCommand(reportingRequest, confirmation)
	if err != nil {
		log.Fatalf("Failed to enable reporting on on/off device: %s", err)
	}
	if confirmation.Status.String() != "STATUS_SUCCESS" {
		log.Fatalf("Failed to enable reporting on on/off device. Status:%s", confirmation.Status.String())
	}

	reportingResponse := &gateway.GwSetAttributeReportingRspInd{}
	err = c.device.driver.gatewayConn.WaitForSequenceResponse(confirmation.SequenceNumber, reportingResponse)
	if err != nil {
		log.Fatalf("Failed to get reporting response: %s", err)
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
	toggleRequest := &gateway.DevSetOnOffStateReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
		State: gateway.GwOnOffStateT_TOGGLE_STATE.Enum(),
	}

	confirmation := &gateway.GwZigbeeGenericCnf{}

	err := c.device.driver.gatewayConn.SendCommand(toggleRequest, confirmation)
	if err != nil {
		return fmt.Errorf("Failed to send on/off state request: %e", err)
	}

	if confirmation.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to send state request. Status:%s", confirmation.Status.String())
	}

	response := &gateway.GwZigbeeGenericRspInd{}
	err = c.device.driver.gatewayConn.WaitForSequenceResponse(confirmation.SequenceNumber, response)
	if err != nil {
		return fmt.Errorf("Failed to get on/off state response: %e", err)
	}

	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("On/off state request failed Status:%s", confirmation.Status.String())
	}

	return nil
}
