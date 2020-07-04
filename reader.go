package main

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
	"github.com/xperimental/flowercare-exporter/pkg/miflora"
)

type query struct {
	MacAddress string
	Result     chan queryResult
}

type queryResult struct {
	Data miflora.Data
	Err  error
}

type queuedReader struct {
	cooldownPeriod time.Duration
	deviceName     string
	shutdown       bool
	queryCh        chan query
	device         ble.Device
}

func newQueuedDataReader(cooldownPeriod time.Duration, deviceName string) (*queuedReader, error) {
	device, err := linux.NewDeviceWithName(deviceName)
	if err != nil {
		return nil, err
	}

	return &queuedReader{
		cooldownPeriod: cooldownPeriod,
		deviceName:     deviceName,
		queryCh:        make(chan query, 1),
		device:         device,
	}, nil
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
				log.Debugf("Reading data for %q on %q", q.MacAddress, r.deviceName)
				data, err := miflora.ReadData(ctx, log, r.device, q.MacAddress)

				q.Result <- queryResult{
					Data: data,
					Err:  err,
				}
				close(q.Result)

				if r.cooldownPeriod > 0 {
					time.Sleep(r.cooldownPeriod)
				}
			}
		}
	}()
}

func (r *queuedReader) ReadFunc(macAddress string) func() (miflora.Data, error) {
	log.Debugf("Creating reader for %q on %q", macAddress, r.deviceName)
	return func() (miflora.Data, error) {
		if r.shutdown {
			return miflora.Data{}, errors.New("reader shut down")
		}

		q := query{
			MacAddress: macAddress,
			Result:     make(chan queryResult),
		}

		r.queryCh <- q

		res, ok := <-q.Result
		if !ok {
			return miflora.Data{}, errors.New("channel closed")
		}

		return res.Data, res.Err
	}
}
