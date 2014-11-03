package main

import (
	"reflect"

	"github.com/ninjasphere/go-ninja/channels"
)

// TODO: This should check if this is a motion or presence sensor, and output channels accordingly.
// TODO: This should expose battery alarm
// TODO: This should expose tamper alarm
// BUG: Multiple status events are being received for a single actual event sent from the device
type IASZoneCluster struct {
	Channel
	presence *channels.PresenceChannel
}

type IASZoneStatus struct {
	Alarm1             bool
	Alarm2             bool
	Tamper             bool
	Battery            bool
	SupervisionReports bool
	RestoreReports     bool
	Trouble            bool
	AC                 bool
	Reserved1          bool
	Reserved2          bool
	Reserved3          bool
	Reserved4          bool
	Reserved5          bool
	Reserved6          bool
	Reserved7          bool
	Reserved8          bool
}

func (c *IASZoneCluster) init() error {
	log.Debugf("Initialising IAS Zone cluster of device % X", *c.device.deviceInfo.IeeeAddress)

	stateChange := c.device.driver.gatewayConn.OnZoneState(*c.device.deviceInfo.IeeeAddress)

	go func() {
		for {
			state := <-stateChange

			if state.SrcAddress.EndpointId == c.endpoint.EndpointId {
				log.Infof("IAS Zone change. Device:%X State:%v", *c.device.deviceInfo.IeeeAddress, state)
			}

			status := &IASZoneStatus{}

			readMask(int(*state.ZoneStatus), status)

			c.presence.SendState(status.Alarm1)
		}
	}()

	c.presence = channels.NewPresenceChannel()
	err = c.device.driver.Conn.ExportChannel(c.device, c.presence, c.ID+"presence")
	if err != nil {
		log.Fatalf("Failed to announce presence channel: %s", err)
	}

	return nil
}

func readMask(mask int, target interface{}) {

	targetValue := reflect.Indirect(reflect.ValueOf(target))

	for i := 0; i < targetValue.NumField(); i++ {
		value := (mask >> uint(i) & 1) > 0
		f := targetValue.Field(i)
		if f.CanSet() {
			targetValue.Field(i).SetBool(value)
		}
	}
}
