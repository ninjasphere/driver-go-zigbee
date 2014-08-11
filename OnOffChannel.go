package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/ninjasphere/go-zigbee/gateway"
)

type OnOffChannel struct {
	Channel
}

// -------- On/Off Protocol --------

func (c *OnOffChannel) TurnOn() error {
	return c.setState(gateway.GwOnOffStateT_ON_STATE.Enum())
}

func (c *OnOffChannel) TurnOff() error {
	return c.setState(gateway.GwOnOffStateT_OFF_STATE.Enum())
}

func (c *OnOffChannel) Toggle() error {
	return c.setState(gateway.GwOnOffStateT_TOGGLE_STATE.Enum())
}

func (c *OnOffChannel) Set(state bool) error {
	if state {
		return c.TurnOn()
	}

	return c.TurnOff()
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

	err := c.device.driver.gatewayConn.SendAsyncCommand(request, response, 20*time.Second)
	if err != nil {
		log.Printf("Error enabling on/off reporting: %s", err)
	} else if response.Status.String() != "STATUS_SUCCESS" {
		log.Printf("Failed to enable on/off reporting. status: %s", response.Status.String())
	}

	methods := []string{"turnOn", "turnOff", "set", "toggle"}
	events := []string{"state"}
	bus, _ := c.device.bus.AnnounceChannel("on-off", "on-off", methods, events, func(method string, payload *simplejson.Json) {
		log.Printf("INCOMING ON/OFF : %s", method)

		switch method {
		case "turnOn":
			c.TurnOn()
		case "turnOff":
			c.TurnOff()
		case "toggle":
			c.Toggle()
		case "set":
			state, _ := payload.GetIndex(0).Bool()
			c.Set(state)
		default:
			log.Printf("On-off got an unknown method %s", method)
			return
		}
	})

	c.bus = bus

	go func() {
		for {
			err := c.fetchState()
			if err != nil {
				log.Printf("Failed to poll for on/off state %s", err)
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

	payload, _ := simplejson.NewJson([]byte("false"))
	if *response.StateValue == gateway.GwOnOffStateValueT_ON {
		payload, _ = simplejson.NewJson([]byte("true"))
	}

	c.bus.SendEvent("state", payload)

	return nil
}
