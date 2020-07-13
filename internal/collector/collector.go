package collector

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/xperimental/flowercare-exporter/internal/config"
	"github.com/xperimental/flowercare-exporter/pkg/miflora"
)

const (
	// MetricPrefix contains the prefix used by all metrics emitted from this collector.
	MetricPrefix = "flowercare_"

	// Conversion factor from µS/cm to S/m
	factorConductivity = 0.0001
)

var (
	varLabelNames = []string{
		"macaddress",
		"name",
	}

	upDesc = prometheus.NewDesc(
		MetricPrefix+"up",
		"Shows if data could be successfully retrieved by the collector.",
		varLabelNames, nil)
	updatedTimestampDesc = prometheus.NewDesc(
		MetricPrefix+"updated_timestamp",
		"Contains the timestamp when the last communication with the Bluetooth device happened.",
		varLabelNames, nil)
	infoDesc = prometheus.NewDesc(
		MetricPrefix+"info",
		"Contains information about the Flower Care device.",
		append(varLabelNames, "version"), nil)
	batteryDesc = prometheus.NewDesc(
		MetricPrefix+"battery_percent",
		"Battery level in percent.",
		varLabelNames, nil)
	conductivityDesc = prometheus.NewDesc(
		MetricPrefix+"conductivity_sm",
		"Soil conductivity in Siemens/meter.",
		varLabelNames, nil)
	lightDesc = prometheus.NewDesc(
		MetricPrefix+"brightness_lux",
		"Ambient lighting in lux.",
		varLabelNames, nil)
	moistureDesc = prometheus.NewDesc(
		MetricPrefix+"moisture_percent",
		"Soil relative moisture in percent.",
		varLabelNames, nil)
	temperatureDesc = prometheus.NewDesc(
		MetricPrefix+"temperature_celsius",
		"Ambient temperature in celsius.",
		varLabelNames, nil)
)

// Flowercare implements a Prometheus collector that emits metrics of a Miflora sensor.
type Flowercare struct {
	Log           logrus.FieldLogger
	Source        func(macAddress string) (miflora.Data, error)
	Sensors       []config.Sensor
	StaleDuration time.Duration
}

// Describe implements prometheus.Collector
func (c *Flowercare) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- updatedTimestampDesc
	ch <- infoDesc
	ch <- batteryDesc
	ch <- conductivityDesc
	ch <- lightDesc
	ch <- moistureDesc
	ch <- temperatureDesc
}

// Collect implements prometheus.Collector
func (c *Flowercare) Collect(ch chan<- prometheus.Metric) {
	for _, s := range c.Sensors {
		c.collectSensor(ch, s)
	}
}

func (c *Flowercare) collectSensor(ch chan<- prometheus.Metric, s config.Sensor) {
	labels := []string{
		s.MacAddress,
		s.Name,
	}

	data, err := c.Source(s.MacAddress)
	if err != nil {
		c.Log.Errorf("Error getting data for %q: %s", s, err)
		c.sendMetric(ch, upDesc, 0, labels)

		return
	}
	c.sendMetric(ch, upDesc, 1, labels)
	c.sendMetric(ch, updatedTimestampDesc, float64(data.Time.Unix()), labels)
	c.sendMetric(ch, infoDesc, 1, append(labels, data.Firmware.Version))

	age := time.Since(data.Time)
	if age >= c.StaleDuration {
		c.Log.Debugf("Data for %q is stale: %s > %s", s, age, c.StaleDuration)
		return
	}

	c.collectData(ch, data, labels)
}

func (c *Flowercare) collectData(ch chan<- prometheus.Metric, data miflora.Data, labels []string) {
	for _, metric := range []struct {
		Desc  *prometheus.Desc
		Value float64
	}{
		{
			Desc:  batteryDesc,
			Value: float64(data.Firmware.Battery),
		},
		{
			Desc:  conductivityDesc,
			Value: float64(data.Sensors.Conductivity) * factorConductivity,
		},
		{
			Desc:  lightDesc,
			Value: float64(data.Sensors.Light),
		},
		{
			Desc:  moistureDesc,
			Value: float64(data.Sensors.Moisture),
		},
		{
			Desc:  temperatureDesc,
			Value: data.Sensors.Temperature,
		},
	} {
		c.sendMetric(ch, metric.Desc, metric.Value, labels)
	}
}

func (c *Flowercare) sendMetric(ch chan<- prometheus.Metric, desc *prometheus.Desc, value float64, labels []string) {
	m, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, value, labels...)
	if err != nil {
		c.Log.Errorf("can not create metric %q: %s", desc, err)
		return
	}

	ch <- m
}
