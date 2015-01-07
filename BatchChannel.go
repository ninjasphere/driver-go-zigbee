package main

import "github.com/ninjasphere/go-ninja/devices"

type BatchChannel struct {
	Channel
	onOff      *OnOffChannel
	brightness *BrightnessChannel
	color      *ColorChannel
}

func (c *BatchChannel) init() error {
	log.Debugf("Initialising batch channel of device %d", *c.device.deviceInfo.IeeeAddress)

	err := c.device.driver.Conn.ExportChannel(c.device, c, c.ID)
	if err != nil {
		log.Fatalf("Failed to announce batch channel: %s", err)
	}

	return nil
}

func (c *BatchChannel) SetBatch(state *devices.LightDeviceState) error {

	if c.onOff != nil && state.OnOff != nil {
		c.onOff.SetOnOff(*state.OnOff)
	}

	if c.brightness != nil && state.Brightness != nil {
		c.brightness.SetBrightness(*state.Brightness)
	}

	if c.color != nil && state.Color != nil {
		c.color.SetColor(state.Color)
	}

	return nil
}

func (c *BatchChannel) GetProtocol() string {
	return "core/batching"
}

func (c *BatchChannel) SetEventHandler(_ func(event string, payload ...interface{}) error) {
}
