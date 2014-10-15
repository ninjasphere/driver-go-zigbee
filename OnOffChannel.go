package main

import (
	"fmt"
	"time"

	"github.com/ninjasphere/go-ninja/channels"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type OnOffChannel struct {
	Channel
	channel *channels.OnOffChannel
}

// -------- On/Off Protocol --------

func (c *OnOffChannel) TurnOn() error {
	return c.setState(gateway.GwOnOffStateT_ON_STATE.Enum())
}

func (c *OnOffChannel) TurnOff() error {
	return c.setState(gateway.GwOnOffStateT_OFF_STATE.Enum())
}

func (c *OnOffChannel) ToggleOnOff() error {
	return c.setState(gateway.GwOnOffStateT_TOGGLE_STATE.Enum())
}

func (c *OnOffChannel) SetOnOff(state bool) error {
	if state {
		return c.TurnOn()
	}

	return c.TurnOff()
}

func (c *OnOffChannel) init() error {
	log.Debugf("Initialising on/off channel of device %d", *c.device.deviceInfo.IeeeAddress)

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
		AttributeReportList: []*gateway.GwAttributeReportT{{
			AttributeId:       &attributeID,
			AttributeType:     gateway.GwZclAttributeDataTypesT_ZCL_DATATYPE_BOOLEAN.Enum(),
			MinReportInterval: &minReportInterval,
			MaxReportInterval: &maxReportInterval,
		}},
	}

	response := &gateway.GwSetAttributeReportingRspInd{}

	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 20*time.Second)
	if err != nil {
		log.Errorf("Error enabling on/off reporting: %s", err)
	} else if response.Status.String() != "STATUS_SUCCESS" {
		log.Errorf("Failed to enable on/off reporting. status: %s", response.Status.String())
	}

	c.channel = channels.NewOnOffChannel(c)
	err = c.device.driver.Conn.ExportChannel(c.device, c.channel, "on-off")
	if err != nil {
		log.Fatalf("Failed to announce on/off channel: %s", err)
	}

	go func() {
		for {
			log.Debugf("Polling for on/off")
			err := c.fetchState()
			if err != nil {
				log.Errorf("Failed to poll for on/off state %s", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	return nil

}

func (c *OnOffChannel) setState(state *gateway.GwOnOffStateT) error {
	request := &gateway.DevSetOnOffStateReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
		State: state,
	}

	response := &gateway.GwZigbeeGenericRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 2*time.Second)
	if err != nil {
		return fmt.Errorf("Error setting on/off state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to set on/off state. status: %s", response.Status.String())
	}

	return c.fetchState()
}

func (c *OnOffChannel) fetchState() error {
	request := &gateway.DevGetOnOffStateReq{
		DstAddress: &gateway.GwAddressStructT{
			AddressType: gateway.GwAddressTypeT_UNICAST.Enum(),
			IeeeAddr:    c.device.deviceInfo.IeeeAddress,
		},
	}

	response := &gateway.DevGetOnOffStateRspInd{}
	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 10*time.Second)
	if err != nil {
		return fmt.Errorf("Error getting on/off state : %s", err)
	}
	if response.Status.String() != "STATUS_SUCCESS" {
		return fmt.Errorf("Failed to get on/off state. status: %s", response.Status.String())
	}

	c.channel.SendState(*response.StateValue == gateway.GwOnOffStateValueT_ON)

	return nil
}
