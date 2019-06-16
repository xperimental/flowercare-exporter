package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricPrefix = "flowercare_"

	// Conversion factor from µS/cm to S/m
	factorConductivity = 0.0001
)

type flowercareCollector struct {
	Sensor          sensor
	RefreshDuration time.Duration
	ForgetDuration  time.Duration

	dataReader          func() (sensorData, error)
	cache               sensorData
	upMetric            prometheus.Gauge
	scrapeErrorsMetric  prometheus.Counter
	scrapeTimestampDesc *prometheus.Desc
	infoDesc            *prometheus.Desc
	batteryDesc         *prometheus.Desc
	conductivityDesc    *prometheus.Desc
	lightDesc           *prometheus.Desc
	moistureDesc        *prometheus.Desc
	temperatureDesc     *prometheus.Desc
}

func newCollector(dataReader func() (sensorData, error), refreshDuration time.Duration, sensorInfo sensor) *flowercareCollector {
	constLabels := prometheus.Labels{
		"macaddress": strings.ToLower(sensorInfo.MacAddress),
		"name":       sensorInfo.Name,
	}

	return &flowercareCollector{
		Sensor:          sensorInfo,
		RefreshDuration: refreshDuration,
		ForgetDuration:  refreshDuration * 2,

		dataReader: dataReader,
		upMetric: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        metricPrefix + "up",
			Help:        "Shows if data could be successfully retrieved by the collector.",
			ConstLabels: constLabels,
		}),
		scrapeErrorsMetric: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        metricPrefix + "scrape_errors_total",
			Help:        "Counts the number of scrape errors by this collector.",
			ConstLabels: constLabels,
		}),
		scrapeTimestampDesc: prometheus.NewDesc(
			metricPrefix+"scrape_timestamp",
			"Contains the timestamp when the last communication with the Bluetooth device happened.",
			nil, constLabels),
		infoDesc: prometheus.NewDesc(
			metricPrefix+"info",
			"Contains information about the Flower Care device.",
			[]string{"version"}, constLabels),
		batteryDesc: prometheus.NewDesc(
			metricPrefix+"battery_percent",
			"Battery level in percent.",
			nil, constLabels),
		conductivityDesc: prometheus.NewDesc(
			metricPrefix+"conductivity_sm",
			"Soil conductivity in Siemens/meter.",
			nil, constLabels),
		lightDesc: prometheus.NewDesc(
			metricPrefix+"brightness_lux",
			"Ambient lighting in lux.",
			nil, constLabels),
		moistureDesc: prometheus.NewDesc(
			metricPrefix+"moisture_percent",
			"Soil relative moisture in percent.",
			nil, constLabels),
		temperatureDesc: prometheus.NewDesc(
			metricPrefix+"temperature_celsius",
			"Ambient temperature in celsius.",
			nil, constLabels),
	}
}

func (c *flowercareCollector) Describe(ch chan<- *prometheus.Desc) {
	c.upMetric.Describe(ch)
	c.scrapeErrorsMetric.Describe(ch)

	ch <- c.scrapeTimestampDesc
	ch <- c.infoDesc
	ch <- c.batteryDesc
	ch <- c.conductivityDesc
	ch <- c.lightDesc
	ch <- c.moistureDesc
	ch <- c.temperatureDesc
}

func (c *flowercareCollector) Collect(ch chan<- prometheus.Metric) {
	c.upMetric.Collect(ch)
	c.scrapeErrorsMetric.Collect(ch)

	if time.Since(c.cache.Time) < c.ForgetDuration {
		if err := c.collectData(ch, c.cache); err != nil {
			log.Printf("Error collecting metrics: %s", err)
		}
	}
}

func (c *flowercareCollector) StartUpdate(ctx context.Context, wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		refresh := time.NewTicker(c.RefreshDuration)
		defer refresh.Stop()

		// Trigger first refresh
		go c.doRefresh()

		for {
			select {
			case <-ctx.Done():
				return
			case <-refresh.C:
				c.doRefresh()
			}
		}
	}()
}

func (c *flowercareCollector) doRefresh() {
	log.Debugf("Updating %q", c.Sensor)
	data, err := c.dataReader()
	if err != nil {
		log.Printf("Error updating %q: %s", c.Sensor, err)

		c.scrapeErrorsMetric.Inc()
		c.upMetric.Set(0)
	} else {
		c.upMetric.Set(1)
		c.cache = data
	}
}

func (c *flowercareCollector) collectData(ch chan<- prometheus.Metric, data sensorData) error {
	if err := sendMetric(ch, c.scrapeTimestampDesc, float64(data.Time.Unix())); err != nil {
		return err
	}

	if err := sendMetric(ch, c.infoDesc, 1, data.Firmware.Version); err != nil {
		return err
	}

	for _, metric := range []struct {
		Desc  *prometheus.Desc
		Value float64
	}{
		{
			Desc:  c.batteryDesc,
			Value: float64(data.Firmware.Battery),
		},
		{
			Desc:  c.conductivityDesc,
			Value: float64(data.Sensors.Conductivity) * factorConductivity,
		},
		{
			Desc:  c.lightDesc,
			Value: float64(data.Sensors.Light),
		},
		{
			Desc:  c.moistureDesc,
			Value: float64(data.Sensors.Moisture),
		},
		{
			Desc:  c.temperatureDesc,
			Value: data.Sensors.Temperature,
		},
	} {
		if err := sendMetric(ch, metric.Desc, metric.Value); err != nil {
			return err
		}
	}

	return nil
}

func sendMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels ...string) error {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, labels...)
	if err != nil {
		return fmt.Errorf("can not create metric %q: %s", desc, err)
	}
	ch <- m

	return nil
}
