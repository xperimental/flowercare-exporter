package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/barnybug/miflora"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	metricPrefix = "flowercare_"

	// Conversion factor from µS/cm to S/m
	factorConductivity = 0.1
)

type flowercareCollector struct {
	MacAddress         string
	Device             string
	upMetric           prometheus.Gauge
	scrapeErrorsMetric prometheus.Counter
	infoDesc           *prometheus.Desc
	batteryDesc        *prometheus.Desc
	conductivityDesc   *prometheus.Desc
	lightDesc          *prometheus.Desc
	moistureDesc       *prometheus.Desc
	temperatureDesc    *prometheus.Desc
}

func newCollector(macAddress, device string) *flowercareCollector {
	constLabels := prometheus.Labels{
		"macaddress": strings.ToLower(macAddress),
	}

	return &flowercareCollector{
		MacAddress: macAddress,
		Device:     device,

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
}

func (c *flowercareCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.collectData(ch); err != nil {
		log.Printf("Error during scrape: %s", err)

		c.scrapeErrorsMetric.Inc()
		c.upMetric.Set(0)
	} else {
		c.upMetric.Set(1)
	}

	c.upMetric.Collect(ch)
	c.scrapeErrorsMetric.Collect(ch)
}

func (c *flowercareCollector) collectData(ch chan<- prometheus.Metric) error {
	f := miflora.NewMiflora(c.MacAddress, c.Device)

	firmware, err := f.ReadFirmware()
	if err != nil {
		return fmt.Errorf("can not read firmware: %s", err)
	}

	sensors, err := f.ReadSensors()
	if err != nil {
		return fmt.Errorf("can not read sensors: %s", err)
	}

	if err := sendMetric(ch, c.infoDesc, 1, firmware.Version); err != nil {
		return err
	}

	for _, metric := range []struct {
		Desc  *prometheus.Desc
		Value float64
	}{
		{
			Desc:  c.batteryDesc,
			Value: float64(firmware.Battery),
		},
		{
			Desc:  c.conductivityDesc,
			Value: float64(sensors.Conductivity) * factorConductivity,
		},
		{
			Desc:  c.lightDesc,
			Value: float64(sensors.Light),
		},
		{
			Desc:  c.moistureDesc,
			Value: float64(sensors.Moisture),
		},
		{
			Desc:  c.temperatureDesc,
			Value: sensors.Temperature,
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
