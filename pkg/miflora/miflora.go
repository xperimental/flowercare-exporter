// Package miflora provides a function to read data from Miflora sensors using Bluetooth LE.
package miflora

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/go-ble/ble"
	"github.com/sirupsen/logrus"
)

var (
	firmwareCharacteristic = &ble.Characteristic{
		ValueHandle: 0x38,
	}
	realtimeReadingCharacteristic = &ble.Characteristic{
		ValueHandle: 0x33,
	}
	realtimeReadingValue = []byte{0xA0, 0x1F}
	sensorCharacteristic = &ble.Characteristic{
		ValueHandle: 0x35,
	}
)

// Data contains the data read from the sensor as well as a timestamp.
type Data struct {
	Time     time.Time
	Firmware Firmware
	Sensors  Sensors
}

// Firmware contains information about the device status.
type Firmware struct {
	Version string
	Battery byte
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (f *Firmware) UnmarshalBinary(data []byte) error {
	if len(data) < 3 {
		return fmt.Errorf("data not long enough: %d < 3", len(data))
	}

	f.Battery = data[0]
	f.Version = string(data[2:])
	return nil
}

// Sensors contains the sensor data.
type Sensors struct {
	Temperature  float64
	Moisture     byte
	Light        uint16
	Conductivity uint16
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (s *Sensors) UnmarshalBinary(data []byte) error {
	// TT TT ?? LL LL ?? ?? MM CC CC ?? ?? ?? ?? ?? ??
	if len(data) != 16 {
		return fmt.Errorf("invalid data length: %d != 10", len(data))
	}

	p := bytes.NewBuffer(data)
	var t int16

	if err := binary.Read(p, binary.LittleEndian, &t); err != nil {
		return fmt.Errorf("error reading data: %s", err)
	}

	p.Next(1)
	if err := binary.Read(p, binary.LittleEndian, &s.Light); err != nil {
		return fmt.Errorf("error reading data: %s", err)
	}

	p.Next(2)
	if err := binary.Read(p, binary.LittleEndian, &s.Moisture); err != nil {
		return fmt.Errorf("error reading data: %s", err)
	}
	if err := binary.Read(p, binary.LittleEndian, &s.Conductivity); err != nil {
		return fmt.Errorf("error reading data: %s", err)
	}

	s.Temperature = float64(t) / 10
	return nil
}

// ReadData uses a Bluetooth LE device to read data from the sensor identified using the MAC address.
func ReadData(ctx context.Context, log logrus.FieldLogger, device ble.Device, macAddress string) (Data, error) {
	addr := ble.NewAddr(macAddress)
	c, err := device.Dial(ctx, addr)
	if err != nil {
		return Data{}, fmt.Errorf("error dialing: %s", err)
	}

	firmwareRaw, err := c.ReadCharacteristic(firmwareCharacteristic)
	if err != nil {
		return Data{}, fmt.Errorf("error reading firmware info: %s", err)
	}

	var firmware Firmware
	if err := firmware.UnmarshalBinary(firmwareRaw); err != nil {
		return Data{}, fmt.Errorf("error parsing firmware info: %s", err)
	}
	log.Debugf("Firmware of %q: %#v", macAddress, firmware)

	if err := c.WriteCharacteristic(realtimeReadingCharacteristic, realtimeReadingValue, false); err != nil {
		return Data{}, fmt.Errorf("can not enable realtime reading: %s", err)
	}

	sensorsRaw, err := c.ReadCharacteristic(sensorCharacteristic)
	if err != nil {
		return Data{}, fmt.Errorf("error reading sensor data: %s", err)
	}

	var sensors Sensors
	if err := sensors.UnmarshalBinary(sensorsRaw); err != nil {
		return Data{}, fmt.Errorf("error parsing sensor data: %s", err)
	}
	log.Debugf("Sensors of %q: %#v", macAddress, sensors)

	return Data{
		Time:     time.Now(),
		Firmware: firmware,
		Sensors:  sensors,
	}, nil
}
