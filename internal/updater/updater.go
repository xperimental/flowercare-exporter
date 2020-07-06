package updater

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/sirupsen/logrus"
	"github.com/xperimental/flowercare-exporter/internal/config"
	"github.com/xperimental/flowercare-exporter/pkg/miflora"
)

type data struct {
	Info config.Sensor
	Data *miflora.Data
}

// Updater can be used to get data from a set of Miflora sensors and cache that data temporarily.
type Updater struct {
	log             logrus.FieldLogger
	refreshDuration time.Duration
	deviceName      string
	device          ble.Device

	dataLock sync.RWMutex
	dataMap  map[string]*data
}

// New creates a new Updater using the specified Bluetooth device.
func New(log logrus.FieldLogger, deviceName string) (*Updater, error) {
	device, err := linux.NewDeviceWithName(deviceName)
	if err != nil {
		return nil, err
	}

	return &Updater{
		log:        log,
		deviceName: deviceName,
		device:     device,
		dataMap:    map[string]*data{},
	}, nil
}

// AddSensor adds a sensor to the updater.
func (u *Updater) AddSensor(sensor config.Sensor) {
	u.dataLock.Lock()
	defer u.dataLock.Unlock()

	u.log.Debugf("Adding sensor %q", sensor)
	u.dataMap[sensor.MacAddress] = &data{
		Info: sensor,
	}
}

// GetData returns the latest data available for the sensor identified by its MAC address.
func (u *Updater) GetData(macAddress string) (miflora.Data, error) {
	u.dataLock.RLock()
	defer u.dataLock.RUnlock()

	d, ok := u.dataMap[macAddress]
	if !ok {
		return miflora.Data{}, fmt.Errorf("no sensor with MAC address registered: %s", macAddress)
	}

	if d.Data == nil {
		return miflora.Data{}, errors.New("no data available")
	}

	return *d.Data, nil
}

// Update starts an update run, which tries to get new data for all registered sensors.
func (u *Updater) Update(ctx context.Context, now time.Time) {
	u.log.Debugf("Starting update at %s", now.UTC())
	u.dataLock.Lock()
	defer u.dataLock.Unlock()

	for _, d := range u.dataMap {
		s := d.Info
		u.log.Debugf("Reading data for %q on %q", s.MacAddress, u.deviceName)

		data, err := miflora.ReadData(ctx, u.log, u.device, s.MacAddress)
		if err != nil {
			u.log.Errorf("Error updating data for %q: %s", s, err)
			continue
		}

		d.Data = &data
	}
}
