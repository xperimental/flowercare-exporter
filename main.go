package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/xperimental/flowercare-exporter/internal/collector"
	"github.com/xperimental/flowercare-exporter/internal/config"
	"github.com/xperimental/flowercare-exporter/internal/updater"
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

	provider, err := updater.New(log, config.Device)
	if err != nil {
		log.Fatalf("Error creating device: %s", err)
	}

	for _, s := range config.Sensors {
		log.Infof("Sensor: %s", s)
		provider.AddSensor(s)
	}

	c := &collector.Flowercare{
		Log:           log,
		Source:        provider.GetData,
		Sensors:       config.Sensors,
		StaleDuration: config.StaleDuration,
	}
	if err := prometheus.Register(c); err != nil {
		log.Fatalf("Failed to register collector: %s", err)
	}

	versionMetric := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: collector.MetricPrefix + "build_info",
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

	runLoop(config, provider)
	log.Info("Shutdown complete.")
}

func runLoop(cfg config.Config, provider *updater.Updater) {
	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	refresher := time.NewTicker(cfg.RefreshDuration)

	go provider.Update(ctx, time.Now())
	for {
		select {
		case <-sigCh:
			log.Debug("Got shutdown signal.")

			signal.Reset()
			cancel()
			return
		case now := <-refresher.C:
			go provider.Update(ctx, now)
		}
	}
}
