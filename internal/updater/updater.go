package updater

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/sirupsen/logrus"
	"github.com/xperimental/flowercare-exporter/internal/config"
	"github.com/xperimental/flowercare-exporter/pkg/miflora"
)

var (
	updaterTickDuration = 10 * time.Second
)

type data struct {
	Info config.Sensor
	Data *miflora.Data
}

type queueItem struct {
	Sensor    config.Sensor
	Time      time.Time
	LastRetry time.Duration
}

// Updater can be used to get data from a set of Miflora sensors and cache that data temporarily.
type Updater struct {
	log         logrus.FieldLogger
	retryConfig config.RetryConfig

	deviceName string
	device     ble.Device

	queueLock sync.RWMutex
	queue     map[string]queueItem

	dataLock sync.RWMutex
	dataMap  map[string]*data
}

// New creates a new Updater using the specified Bluetooth device.
func New(log logrus.FieldLogger, deviceName string, retryConfig config.RetryConfig) (*Updater, error) {
	device, err := linux.NewDeviceWithName(deviceName)
	if err != nil {
		return nil, err
	}

	return &Updater{
		log:         log,
		retryConfig: retryConfig,
		deviceName:  deviceName,
		device:      device,
		queue:       map[string]queueItem{},
		dataMap:     map[string]*data{},
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

// Start starts the updater queue. It will periodically check if it needs to update data of one or more sensors.
func (u *Updater) Start(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		ticker := time.NewTicker(updaterTickDuration)
		for {
			select {
			case <-ctx.Done():
				u.log.Debug("Shutting down updater.")
				return
			case now := <-ticker.C:
				next, ok := u.getNextQueueItem(now)
				if !ok {
					continue
				}
				u.log.Debugf("Queue item: %#v", next)

				err := u.updateSensor(ctx, next.Sensor)
				if err != nil {
					u.log.Errorf("Error updating sensor %q: %s", next, err)
					u.retryItem(next, now)
				}
			}
		}
	}()
}

// UpdateAll schedules an update for all registered sensors.
func (u *Updater) UpdateAll(now time.Time) {
	sensors := u.getSensors()

	for _, s := range sensors {
		u.scheduleUpdate(s)
	}
}

func (u *Updater) getSensors() []config.Sensor {
	u.dataLock.RLock()
	defer u.dataLock.RUnlock()

	result := []config.Sensor{}
	for _, d := range u.dataMap {
		result = append(result, d.Info)
	}

	return result
}

func (u *Updater) getNextQueueItem(now time.Time) (queueItem, bool) {
	u.queueLock.Lock()
	defer u.queueLock.Unlock()

	if len(u.queue) == 0 {
		return queueItem{}, false
	}
	u.log.Debugf("Queue length: %d", len(u.queue))

	items := []queueItem{}
	for _, i := range u.queue {
		items = append(items, i)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.Before(items[j].Time)
	})

	next := items[0]
	diff := next.Time.Sub(now)
	if diff > 0 {
		u.log.Debugf("Sensor %q is still waiting %s", next.Sensor, diff)
		return queueItem{}, false
	}

	delete(u.queue, next.Sensor.MacAddress)
	return items[0], true
}

func (u *Updater) scheduleUpdate(sensor config.Sensor) {
	u.queueLock.Lock()
	defer u.queueLock.Unlock()

	u.queue[sensor.MacAddress] = queueItem{
		Sensor:    sensor,
		Time:      time.Now(),
		LastRetry: 0,
	}
}

func (u *Updater) updateSensor(ctx context.Context, sensor config.Sensor) error {
	defer func(start time.Time) {
		elapsed := time.Since(start)
		u.log.Debugf("Updating %q took %s.", sensor, elapsed)
	}(time.Now())

	u.log.Debugf("Reading data for %q on %q", sensor.MacAddress, u.deviceName)
	data, err := miflora.ReadData(ctx, u.log, u.device, sensor.MacAddress)
	if err != nil {
		return fmt.Errorf("can not read data: %s", err)
	}

	u.dataLock.Lock()
	defer u.dataLock.Unlock()

	mapItem := u.dataMap[sensor.MacAddress]
	mapItem.Data = &data
	return nil
}

func (u *Updater) retryItem(item queueItem, now time.Time) {
	retryAfter := item.LastRetry
	if retryAfter < u.retryConfig.MinDuration {
		retryAfter = u.retryConfig.MinDuration
	} else {
		retryAfter = time.Duration(float64(retryAfter) * u.retryConfig.Factor)

		if retryAfter > u.retryConfig.MaxDuration {
			retryAfter = u.retryConfig.MaxDuration
		}
	}

	u.queueLock.Lock()
	defer u.queueLock.Unlock()

	u.log.Debugf("Retrying %q after %s", item.Sensor, retryAfter)
	u.queue[item.Sensor.MacAddress] = queueItem{
		Sensor:    item.Sensor,
		Time:      now.Add(retryAfter),
		LastRetry: retryAfter,
	}
}
