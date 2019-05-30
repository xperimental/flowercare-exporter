package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	log = logrus.New()
)

func main() {
	config, err := parseConfig()
	if err != nil {
		log.Fatalf("Error in configuration: %s", err)
	}

	log.Infof("Bluetooth Device: %s", config.Device)

	for _, s := range config.Sensors {
		log.Infof("Sensor: %s", s)

		reader := newDataReader(s.MacAddress, config.Device)
		collector := newCollector(reader, config.CacheDuration, s)

		if err := prometheus.Register(collector); err != nil {
			log.Fatalf("Failed to register collector: %s", err)
		}
	}

	http.Handle("/metrics", prometheus.UninstrumentedHandler())
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))

	log.Infof("Listen on %s...", config.ListenAddr)
	log.Fatal(http.ListenAndServe(config.ListenAddr, nil))
}
