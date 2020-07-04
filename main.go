package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/xperimental/flowercare-exporter/internal/config"
)

var (
	log = &logrus.Logger{
		Out: os.Stderr,
		Formatter: &logrus.TextFormatter{
			DisableTimestamp: true,
		},
		Hooks:        make(logrus.LevelHooks),
		Level:        logrus.InfoLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	}

	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	config, err := config.Parse(log)
	if err != nil {
		log.Fatalf("Error in configuration: %s", err)
	}

	log.SetLevel(logrus.Level(config.LogLevel))
	log.Infof("Bluetooth Device: %s", config.Device)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	go func() {
		sigCh := make(chan os.Signal)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		<-sigCh
		log.Debug("Got shutdown signal.")

		signal.Reset()
		cancel()
	}()

	reader, err := newQueuedDataReader(config.CooldownPeriod, config.Device)
	if err != nil {
		log.Fatalf("Error creating device: %s", err)
	}

	reader.Run(ctx, wg)

	for _, s := range config.Sensors {
		log.Infof("Sensor: %s", s)

		reader := reader.ReadFunc(s.MacAddress)
		collector := newCollector(reader, config.RefreshDuration, s)

		if err := prometheus.Register(collector); err != nil {
			log.Fatalf("Failed to register collector: %s", err)
		}

		collector.StartUpdate(ctx, wg)
	}

	versionMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: metricPrefix + "build_info",
		Help: "Contains build information as labels. Value set to 1.",
		ConstLabels: prometheus.Labels{
			"version": version,
			"commit":  commit,
			"date":    date,
		},
	})
	versionMetric.Set(1)
	prometheus.MustRegister(versionMetric)

	http.Handle("/metrics", promhttp.Handler())
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusFound))

	go func() {
		log.Infof("Listen on %s...", config.ListenAddr)
		log.Fatal(http.ListenAndServe(config.ListenAddr, nil))
	}()

	log.Info("Startup complete.")
	wg.Wait()
	log.Info("Shutdown complete.")
}
