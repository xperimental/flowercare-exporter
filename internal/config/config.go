package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

type SensorList []Sensor

func (s *SensorList) String() string {
	if len(*s) == 0 {
		return ""
	}

	sensors := []string{}
	for _, sensor := range *s {
		sensors = append(sensors, sensor.String())
	}
	return fmt.Sprintf("%s", sensors)
}

func (s *SensorList) Type() string {
	return "address"
}

func (s *SensorList) Set(value string) error {
	sensor, err := parseSensor(value)
	if err != nil {
		return fmt.Errorf("can not parse sensor: %s", err)
	}

	*s = append(*s, sensor)
	return nil
}

type Sensor struct {
	Name       string
	MacAddress string
}

func (s Sensor) String() string {
	if s.Name == "" {
		return s.MacAddress
	}

	return fmt.Sprintf("%s (%s)", s.Name, s.MacAddress)
}

func parseSensor(value string) (Sensor, error) {
	if len(value) == 0 {
		return Sensor{}, errors.New("empty string")
	}

	tokens := strings.SplitN(value, "=", 2)
	if len(tokens) == 1 {
		return Sensor{
			MacAddress: tokens[0],
		}, nil
	}

	return Sensor{
		Name:       tokens[0],
		MacAddress: tokens[1],
	}, nil
}

type LogLevel logrus.Level

func (l *LogLevel) Type() string {
	return "level"
}

func (l *LogLevel) String() string {
	return fmt.Sprintf("%s", logrus.Level(*l))
}

func (l *LogLevel) Set(val string) error {
	level, err := logrus.ParseLevel(val)
	if err != nil {
		return err
	}

	*l = LogLevel(level)
	return nil
}

type Config struct {
	LogLevel        LogLevel
	ListenAddr      string
	Sensors         SensorList
	Device          string
	RefreshDuration time.Duration
	StaleDuration   time.Duration
	CooldownPeriod  time.Duration
}

func Parse(log logrus.FieldLogger) (Config, error) {
	result := Config{
		LogLevel:        LogLevel(logrus.InfoLevel),
		ListenAddr:      ":9294",
		Device:          "hci0",
		RefreshDuration: 2 * time.Minute,
		StaleDuration:   5 * time.Minute,
		CooldownPeriod:  30 * time.Second,
	}

	pflag.Var(&result.LogLevel, "log-level", "Minimum log level to show.")
	pflag.StringVarP(&result.ListenAddr, "addr", "a", result.ListenAddr, "Address to listen on for connections.")
	pflag.VarP(&result.Sensors, "sensor", "s", "MAC-address of sensor to collect data from. Can be specified multiple times.")
	pflag.StringVarP(&result.Device, "adapter", "i", result.Device, "Bluetooth device to use for communication.")
	pflag.DurationVarP(&result.RefreshDuration, "refresh-duration", "r", result.RefreshDuration, "Interval used for refreshing data from bluetooth devices.")
	pflag.DurationVar(&result.StaleDuration, "stale-duration", result.StaleDuration, "Duration after which data is considered stale and is not used for metrics anymore.")
	pflag.DurationVar(&result.CooldownPeriod, "cool-down-period", result.CooldownPeriod, "Time to wait between subsequent access to Bluetooth device.")
	pflag.Parse()

	if len(result.Sensors) == 0 {
		return result, errors.New("need to provide at least one sensor")
	}

	if len(result.Device) == 0 {
		return result, errors.New("need to provide a bluetooth device")
	}

	if result.RefreshDuration < time.Minute {
		log.Warnf("Refresh durations below one minute are discouraged: %s", result.RefreshDuration)
	}

	if result.StaleDuration < (2 * result.RefreshDuration) {
		return result, fmt.Errorf("stale duration needs to be at least %d", 2*result.RefreshDuration)
	}

	return result, nil
}
