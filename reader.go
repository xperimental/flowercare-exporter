package main

import (
	"fmt"
	"time"

	"github.com/barnybug/miflora"
)

type sensorData struct {
	Time     time.Time
	Firmware miflora.Firmware
	Sensors  miflora.Sensors
}

func newDataReader(macAddress, device string) func() (sensorData, error) {
	return func() (sensorData, error) {
		f := miflora.NewMiflora(macAddress, device)

		firmware, err := f.ReadFirmware()
		if err != nil {
			return sensorData{}, fmt.Errorf("can not read firmware: %s", err)
		}

		sensors, err := f.ReadSensors()
		if err != nil {
			return sensorData{}, fmt.Errorf("can not read sensors: %s", err)
		}

		return sensorData{
			Time:     time.Now(),
			Firmware: firmware,
			Sensors:  sensors,
		}, nil
	}
}
