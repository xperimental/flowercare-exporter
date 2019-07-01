package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/barnybug/miflora"
)

type sensorData struct {
	Time     time.Time
	Firmware miflora.Firmware
	Sensors  miflora.Sensors
}

func readData(macAddress, device string) (sensorData, error) {
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

type query struct {
	MacAddress string
	Device     string
	Result     chan queryResult
}

type queryResult struct {
	Data sensorData
	Err  error
}

type queuedReader struct {
	shutdown bool
	queryCh  chan query
}

func newQueuedDataReader() *queuedReader {
	return &queuedReader{
		queryCh: make(chan query, 1),
	}
}

func (r *queuedReader) Run(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()
		defer log.Debug("Shutdown reader loop.")

		log.Debug("Starting reader loop.")
		for {
			select {
			case <-ctx.Done():
				r.shutdown = true
				return
			case q := <-r.queryCh:
				log.Debugf("Reading data for %q on %q", q.MacAddress, q.Device)
				data, err := readData(q.MacAddress, q.Device)

				q.Result <- queryResult{
					Data: data,
					Err:  err,
				}
				close(q.Result)
			}
		}
	}()
}

func (r *queuedReader) ReadFunc(macAddress, device string) func() (sensorData, error) {
	log.Debugf("Creating reader for %q on %q", macAddress, device)
	return func() (sensorData, error) {
		if r.shutdown {
			return sensorData{}, errors.New("reader shut down")
		}

		q := query{
			MacAddress: macAddress,
			Device:     device,
			Result:     make(chan queryResult),
		}

		r.queryCh <- q

		res, ok := <-q.Result
		if !ok {
			return sensorData{}, errors.New("channel closed")
		}

		return res.Data, res.Err
	}
}
